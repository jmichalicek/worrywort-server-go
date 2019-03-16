package graphqlApi_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	txdb "github.com/DATA-DOG/go-txdb"
	"github.com/google/uuid"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/graphqlApi"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

const DefaultUserKey string = "user"

func TestMain(m *testing.M) {
	dbUser, _ := os.LookupEnv("DATABASE_USER")
	dbPassword, _ := os.LookupEnv("DATABASE_PASSWORD")
	dbHost, _ := os.LookupEnv("DATABASE_HOST")
	// we register an sql driver txdb
	connString := fmt.Sprintf("host=%s port=5432 user=%s dbname=worrywort_test sslmode=disable", dbHost,
		dbUser)
	if dbPassword != "" {
		connString += fmt.Sprintf(" password=%s", dbPassword)
	}
	txdb.Register("txdb", "postgres", connString)
	retCode := m.Run()
	os.Exit(retCode)
}

func setUpTestDb() (*sqlx.DB, error) {
	_db, err := sql.Open("txdb", "one")
	if err != nil {
		return nil, err
	}

	db := sqlx.NewDb(_db, "postgres")
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Make a standard, generic batch for testing
// optionally attach the user
func makeTestBatch(u worrywort.User, attachUser bool) worrywort.Batch {
	b := worrywort.Batch{Name: "Testing", BrewedDate: addMinutes(time.Now(), 1),
		BottledDate: addMinutes(time.Now(), 10), VolumeBoiled: 5, VolumeInFermentor: 4.5, VolumeUnits: worrywort.GALLON,
		OriginalGravity: 1.060, FinalGravity: 1.020, UserId: u.Id, BrewNotes: "Brew notes", TastingNotes: "Taste notes",
		RecipeURL: "http://example.org/beer"}
	if attachUser {
		b.CreatedBy = &u
	}
	return b
}

// utility to add a given number of minutes to a time.Time and round to match
// what postgres returns
func addMinutes(d time.Time, increment int) time.Time {
	return d.Add(time.Duration(increment) * time.Minute).Round(time.Microsecond)
}

func TestLoginMutation(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	var worrywortSchema = graphql.MustParseSchema(graphqlApi.Schema, graphqlApi.NewResolver(db))
	user := worrywort.NewUser(nil, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	// This is the hash for the password `password`
	// var hashedPassword string = "$2a$13$pPg7mwPA.VFf3W9AUZyMGO0Q2nhoh/979F/TZ8ED.iqVubLe.TDmi"
	user, err = worrywort.SetUserPassword(user, "password", bcrypt.MinCost)
	user, err = worrywort.SaveUser(db, user)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}
	// hashedPassword := user.Password()

	t.Run("Test valid email and password returns a token", func(t *testing.T) {
		// This is all based on https://github.com/neelance/graphql-go/blob/master/gqltesting/testing.go#L38
		// but allows for more flexible checking of the response
		variables := map[string]interface{}{
			"username": "user@example.com",
			"password": "password",
		}
		query := `
			mutation Login($username: String!, $password: String!) {
				login(username: $username, password: $password) {
					token
				}
			}
		`
		operationName := ""
		context := context.Background()
		resultData := worrywortSchema.Exec(context, query, operationName, variables)

		// example:
		// {"login":{"token":"c9d103e1-8320-45fd-8ac6-245d59c01b3d:HRXG69cqTv1kyG6zmsJo0tJNsEKmeCqWH5WeH3H-_IyTHZ46ivz0KyTTfUgun1CNCV3n1HLwizvAET1I2DwJiA=="}}
		// the hash, the part of the token after the colon, is a base64 encoded sha512 sum
		type loginPayload struct {
			Token               string `json:"token"`
		}

		type loginResponse struct {
			Login loginPayload `json:"login"`
		}

		var result loginResponse
		err = json.Unmarshal(resultData.Data, &result)
		// t.Fatalf("\n\n\n*****************\n%v\n\n\n", spew.Sdump(result))
		if err != nil {
			t.Fatalf("%v", result)
		}

		// Make sure that the token really was inserted into the db
		parts := strings.Split(result.Login.Token, ":")
		tokenId := parts[0]
		newToken := worrywort.AuthToken{}
		query = db.Rebind(
			`SELECT t.id, t.token, t.scope, t.expires_at, t.created_at, t.updated_at, u.id "user.id",
				u.first_name "user.first_name", u.last_name "user.last_name", u.email "user.email",
				u.created_at "user.created_at", u.updated_at "user.updated_at", u.password "user.password"
			 FROM user_authtokens t
			 INNER JOIN users u ON t.user_id = u.id
			 WHERE t.id = ?`)

		err := db.Get(&newToken, query, tokenId)
		if err != nil {
			t.Errorf("Error looking up newly created token: %v", err)
		}

		if newToken == (worrywort.AuthToken{}) {
			t.Errorf("Expected auth token with id %s to be saved to database", tokenId)
		}

		if !cmp.Equal(newToken.User, user) {
			t.Errorf("Expected: - | Got +\n%s", cmp.Diff(newToken, user))
			// t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})
}

func TestBatchQuery(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.NewUser(nil, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = worrywort.SaveUser(db, u)

	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	u2 := worrywort.NewUser(nil, "user2@example.com", "Justin", "M", time.Now(), time.Now())
	u2, err = worrywort.SaveUser(db, u2)

	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)

	// TODO: Can this become global to these tests?
	var worrywortSchema = graphql.MustParseSchema(graphqlApi.Schema, graphqlApi.NewResolver(db))

	b := makeTestBatch(u, true)
	b, err = worrywort.SaveBatch(db, b)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	b2 := makeTestBatch(u, true)
	b2, err = worrywort.SaveBatch(db, b2)

	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	u2batch := makeTestBatch(u2, true)
	u2batch, err = worrywort.SaveBatch(db, u2batch)

	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	t.Run("Test query for batch(id: ID!) which exists returns the batch", func(t *testing.T) {
		variables := map[string]interface{}{
			"id": b.Uuid,
		}
		query := `
			query getBatch($id: ID!) {
				batch(id: $id) {
					__typename
					id
				}
			}
		`
		operationName := ""
		result := worrywortSchema.Exec(ctx, query, operationName, variables)

		var expected interface{}
		err := json.Unmarshal([]byte(fmt.Sprintf(`{"batch": {"__typename": "Batch", "id": "%s"}}`, b.Uuid)), &expected)
		if err != nil {
			t.Fatalf("%v", err)
		}

		var actual interface{}
		err = json.Unmarshal(result.Data, &actual)
		if err != nil {
			t.Fatalf("%v", err)
		}

		if !cmp.Equal(expected, actual) {
			t.Errorf("Expected: - | Got +\n%s", cmp.Diff(expected, actual))
			// t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})

	t.Run("Test query for batch(id: ID!) which does not exist returns null", func(t *testing.T) {
		badUuid := uuid.New().String()
		variables := map[string]interface{}{
			"id": badUuid,
		}
		query := `
			query getBatch($id: ID!) {
				batch(id: $id) {
					id
				}
			}
		`
		operationName := ""
		result := worrywortSchema.Exec(ctx, query, operationName, variables)

		expected := `{"batch":null}`
		if expected != string(result.Data) {
			t.Errorf("Expected: %s\nGot: %s", expected, result.Data)
		}
	})

	t.Run("Test batches() query returns the users batches", func(t *testing.T) {
		// could stop at batches() just to see that it returns the correct type
		// and then know that is correct from the struct level testing of each type
		// but want to see that the user filter works anyway
		query := `
			query getBatches {
				batches {
					__typename
					edges {
						__typename
						node {
							__typename
							id
						}
					}
				}
			}
		`
		operationName := ""
		result := worrywortSchema.Exec(ctx, query, operationName, nil)

		var expected interface{}
		err := json.Unmarshal(
			[]byte(
				fmt.Sprintf(
					`{"batches": {
						"__typename":"BatchConnection",
						"edges": [{"__typename": "BatchEdge","node": {"__typename":"Batch","id":"%s"}},
								  {"__typename": "BatchEdge","node": {"__typename":"Batch","id":"%s"}}]}}`,
					b.Uuid, b2.Uuid)), &expected)
		if err != nil {
			t.Fatalf("%v", err)
		}

		var actual interface{}
		err = json.Unmarshal(result.Data, &actual)
		if err != nil {
			t.Fatalf("%v", err)
		}

		if !cmp.Equal(expected, actual) {
			t.Errorf("Expected: - | Got +\n%s", cmp.Diff(expected, actual))
			// t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})

	t.Run("Test batches() query when not authenticated", func(t *testing.T) {
		// TODO: This WILL start returning a 403 once I correct how auth works
		query := `
			query getBatches {
				batches {
					__typename
					edges {
						__typename
						node {
							__typename
							id
						}
					}
				}
			}
		`
		operationName := ""
		ctx2 := context.Background()
		ctx2 = context.WithValue(ctx2, "db", db)
		result := worrywortSchema.Exec(ctx2, query, operationName, nil)

		var expected interface{}
		err := json.Unmarshal(
			[]byte(
				fmt.Sprintf(
					`{"batches": {"__typename":"BatchConnection","edges": []}}`)), &expected)
		if err != nil {
			t.Fatalf("%v", err)
		}

		var actual interface{}
		err = json.Unmarshal(result.Data, &actual)
		if err != nil {
			t.Fatalf("%v", err)
		}

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})
}

func TestCreateTemperatureMeasurementMutation(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.NewUser(nil, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = worrywort.SaveUser(db, u)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor, err := worrywort.SaveSensor(db, worrywort.Sensor{UserId: u.Id, Name: "Test Sensor", CreatedBy: &u})
	if err != nil {
		t.Fatalf("%v", err)
	}

	u2 := worrywort.NewUser(nil, "user2@example.com", "Justin", "M", time.Now(), time.Now())
	u2, err = worrywort.SaveUser(db, u2)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	// TODO: Can this become global to these tests?
	var worrywortSchema = graphql.MustParseSchema(graphqlApi.Schema, graphqlApi.NewResolver(db))
	t.Run("Test measurement is created with valid data", func(t *testing.T) {
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"sensorId":    strconv.Itoa(int(*sensor.Id)),
				"units":       "FAHRENHEIT",
				"temperature": 70.0,
				"recordedAt":  "2018-10-14T15:26:00+00:00",
			},
		}
		query := `
			mutation addMeasurement($input: CreateTemperatureMeasurementInput!) {
				createTemperatureMeasurement(input: $input) {
					__typename
					temperatureMeasurement {
						__typename
						id
					}
				}
			}`

		operationName := ""
		ctx := context.Background()
		ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)
		ctx = context.WithValue(ctx, "db", db)
		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)

		// Some structs so that the json can be unmarshalled
		type tm struct {
			Typename string `json:"__typename"`
			Id       string `json:"id"`
		}
		type createTemperatureMeasurementPayload struct {
			Typename               string `json:"__typename"`
			TemperatureMeasurement tm     `json:"temperatureMeasurement"`
		}

		type createTemperatureMeasurement struct {
			CreateTemperatureMeasurement createTemperatureMeasurementPayload `json:"createTemperatureMeasurement"`
		}

		var result createTemperatureMeasurement
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v", result)
		}

		// Test the returned graphql types
		if result.CreateTemperatureMeasurement.Typename != "CreateTemperatureMeasurementPayload" {
			t.Errorf("createTemperatureMeasurement returned unexpected type: %s", result.CreateTemperatureMeasurement.Typename)
		}

		if result.CreateTemperatureMeasurement.TemperatureMeasurement.Typename != "TemperatureMeasurement" {
			t.Errorf("createTemperatureMeasurement returned unexpected type for TemperatureMeasurement: %s", result.CreateTemperatureMeasurement.TemperatureMeasurement.Typename)
		}

		// Look up the object in the db to be sure it was created
		var measurementId string = result.CreateTemperatureMeasurement.TemperatureMeasurement.Id
		// TODO: implement FindTemperatureMeasurement
		// measurement, err := worrywort.FindTemperatureMeasurement(db,
		// 	map[string]interface{}{"user_id": u.Id, "id": measurementId})
		measurement := &worrywort.TemperatureMeasurement{}

		selectCols := fmt.Sprintf("tm.user_id, tm.sensor_id")
		q := `SELECT tm.temperature, tm.units,  ` + strings.Trim(selectCols, ", ") + ` from temperature_measurements tm WHERE tm.id = ? AND tm.user_id = ? AND tm.sensor_id = ?`
		query = db.Rebind(q)
		err = db.Get(measurement, query, measurementId, u.Id, sensor.Id)

		if err == sql.ErrNoRows {
			t.Error("Measurement was not saved to the database. Query returned no results.")
		} else if err != nil {
			t.Errorf("%v", err)
		}

	})
}

func TestSensorQuery(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.NewUser(nil, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = worrywort.SaveUser(db, u)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)

	u2 := worrywort.NewUser(nil, "user2@example.com", "Justin", "M", time.Now(), time.Now())
	u2, err = worrywort.SaveUser(db, u2)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	// TODO: Can this become global to these tests?
	var worrywortSchema = graphql.MustParseSchema(graphqlApi.Schema, graphqlApi.NewResolver(db))
	sensor1, err := worrywort.SaveSensor(db, worrywort.Sensor{Name: "Sensor 1", UserId: u.Id})
	sensor2, err := worrywort.SaveSensor(db, worrywort.Sensor{Name: "Sensor 2", UserId: u.Id})
	// Need one owned by another user to ensure it does not show up
	_, err = worrywort.SaveSensor(db, worrywort.Sensor{Name: "Sensor 2", UserId: u2.Id})
	if err != nil {
		t.Fatalf("%v", err)
	}

	t.Run("Test query for sensor(id: ID!) which exists returns the sensor", func(t *testing.T) {
		variables := map[string]interface{}{
			"id": strconv.Itoa(int(*sensor1.Id)),
		}
		query := `
			query getSensor($id: ID!) {
				sensor(id: $id) {
					__typename
					id
				}
			}
		`
		operationName := ""
		result := worrywortSchema.Exec(ctx, query, operationName, variables)

		var expected interface{}
		err := json.Unmarshal([]byte(fmt.Sprintf(`{"sensor": {"__typename": "Sensor", "id": "%d"}}`, *sensor1.Id)), &expected)
		if err != nil {
			t.Fatalf("%v", err)
		}

		var resultData interface{}
		err = json.Unmarshal(result.Data, &resultData)
		if err != nil {
			t.Fatalf("%v", resultData)
		}

		if !cmp.Equal(expected, resultData) {
			t.Errorf("Expected: - | Got +\n%s", cmp.Diff(expected, resultData))
			// t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})

	t.Run("Test query for sensor(id: ID!) which does not exist returns null", func(t *testing.T) {
		variables := map[string]interface{}{
			"id": "-1",
		}
		query := `
			query getSensor($id: ID!) {
				sensor(id: $id) {
					__typename
					id
				}
			}
		`
		operationName := ""
		result := worrywortSchema.Exec(ctx, query, operationName, variables)

		expected := `{"sensor":null}`
		if expected != string(result.Data) {
			t.Errorf("Expected: %s\nGot: %s", expected, result.Data)
		}
	})

	t.Run("Test sensors() query returns the users sensors", func(t *testing.T) {
		query := `
			query getSensors {
				sensors {
					__typename
					edges {
						__typename
						node {
							__typename
							id
						}
					}
				}
			}
		`
		operationName := ""
		result := worrywortSchema.Exec(ctx, query, operationName, nil)
		var expected interface{}
		err := json.Unmarshal(
			[]byte(
				fmt.Sprintf(
					`{"sensors": {"__typename":"SensorConnection","edges": [{"__typename": "SensorEdge","node": {"__typename":"Sensor","id":"%d"}},{"__typename": "SensorEdge","node": {"__typename":"Sensor","id":"%d"}}]}}`,
					*sensor1.Id, *sensor2.Id)), &expected)
		if err != nil {
			t.Fatalf("%v", err)
		}

		var actual interface{}
		err = json.Unmarshal(result.Data, &actual)
		if err != nil {
			t.Fatalf("%v", err)
		}

		if !cmp.Equal(expected, actual) {
			t.Errorf("Expected: - | Got +\n%s", cmp.Diff(expected, actual))
			// t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})
}

func TestCreateBatchMutation(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.NewUser(nil, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = worrywort.SaveUser(db, u)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	var worrywortSchema = graphql.MustParseSchema(graphqlApi.Schema, graphqlApi.NewResolver(db))
	t.Run("Test batch is created with valid data", func(t *testing.T) {
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"name":     "Test Batch",
				"brewedAt": "2018-10-14T15:26:00+00:00",
			},
		}
		query := `
			mutation addBatch($input: CreateBatchInput!) {
				createBatch(input: $input) {
					__typename
					batch {
						__typename
						id
					}
				}
			}`

		operationName := ""
		ctx := context.Background()
		ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)
		ctx = context.WithValue(ctx, "db", db)
		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)

		// Some structs so that the json can be unmarshalled.
		// Unsure why I am doing it this way here and not the same interface{} unmarshall as before.
		type payloadBatch struct {
			Typename string `json:"__typename"`
			Id       string `json:"id"`
		}

		type cbPayload struct {
			Typename string       `json:"__typename"`
			Batch    payloadBatch `json:"batch"`
		}

		type createBatch struct {
			CreateBatch cbPayload `json:"createBatch"`
		}

		var result createBatch
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v: %v", result, resultData)
		}

		// Test the returned graphql types
		if result.CreateBatch.Typename != "CreateBatchPayload" {
			t.Errorf("createBatch returned unexpected type: %s", result.CreateBatch.Typename)
		}

		if result.CreateBatch.Batch.Typename != "Batch" {
			t.Errorf("createBatch returned unexpected type for Batch: %s", result.CreateBatch.Batch.Typename)
		}

		// Look up the object in the db to be sure it was created
		var batchId string = result.CreateBatch.Batch.Id
		batch, err := worrywort.FindBatch(map[string]interface{}{"user_id": *u.Id, "uuid": batchId}, db)

		if err == sql.ErrNoRows {
			t.Error("Batch was not saved to the database. Query returned no results.")
		} else if err != nil {
			t.Errorf("Error: %v and Batch: %v", err, batch)
		}
		// TODO: Really should maybe make sure all properties were inserted
	})
}

func TestAssociateSensorToBatchMutation(t *testing.T) {

	query := `
		mutation associateSensorToBatch($input: AssociateSensorToBatchInput!) {
			associateSensorToBatch(input: $input) {
				__typename
				batchSensorAssociation {
					__typename
					id
				}
			}
		}`

	// Structs for marshalling json to so that actual values can easily be checked and used
	type payloadAssoc struct {
		Typename string `json:"__typename"`
		Id       string `json:"id"`
	}

	type payload struct {
		Typename string        `json:"__typename"`
		Assoc    *payloadAssoc `json:"BatchSensorAssociation"`
	}

	type createAssoc struct {
		Pl payload `json:"associateSensorToBatch"`
	}

	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.NewUser(nil, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = worrywort.SaveUser(db, u)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor, err := worrywort.SaveSensor(db, worrywort.Sensor{UserId: u.Id, Name: "Test Sensor", CreatedBy: &u})
	if err != nil {
		t.Fatalf("%v", err)
	}

	batch, err := worrywort.SaveBatch(
		db, worrywort.Batch{UserId: u.Id, CreatedBy: &u, Name: "Test batch"})
	if err != nil {
		t.Fatalf("%v", err)
	}

	u2 := worrywort.NewUser(nil, "user2@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u2, err = worrywort.SaveUser(db, u2)

	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor2, err := worrywort.SaveSensor(
		db, worrywort.Sensor{UserId: u2.Id, Name: "Test Sensor 2", CreatedBy: &u2})
	if err != nil {
		t.Fatalf("%v", err)
	}

	batch2, err := worrywort.SaveBatch(
		db, worrywort.Batch{UserId: u2.Id, CreatedBy: &u2, Name: "Test batch 2"})
	if err != nil {
		t.Fatalf("%v", err)
	}

	var worrywortSchema = graphql.MustParseSchema(graphqlApi.Schema, graphqlApi.NewResolver(db))
	operationName := ""
	ctx := context.Background()
	ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)
	ctx = context.WithValue(ctx, "db", db)

	cleanAssociations := func() {
		q := `DELETE FROM batch_sensor_association WHERE sensor_id = ? AND batch_id = ?`
		q = db.Rebind(q)
		_, err := db.Exec(q, sensor.Id, batch.Id)
		if err != nil {
			panic(err)
		}
	}

	t.Run("associate to batch", func(t *testing.T) {
		defer cleanAssociations()
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"batchId":     batch.Uuid,
				"sensorId":    strconv.Itoa(int(*sensor.Id)),
				"description": "It is associated",
			},
		}

		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)
		var result createAssoc
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v: %v", result, resultData)
		}

		// Test the returned graphql types
		if result.Pl.Typename != "AssociateSensorToBatchPayload" {
			t.Errorf("associateBatchToSensor returned unexpected type: %s", result.Pl.Typename)
		}

		if result.Pl.Assoc.Typename != "BatchSensorAssociation" {
			t.Errorf("associateBatchToSensor returned unexpected type for Assoc: %s", result.Pl.Assoc.Typename)
		}

		// Make sure it was really created in the db
		newAssoc, err := worrywort.FindBatchSensorAssociation(
			map[string]interface{}{"id": result.Pl.Assoc.Id, "batch_id": *batch.Id, "sensor_id": *sensor.Id}, db)

		if err == sql.ErrNoRows {
			t.Error("BatchSensor was not saved to the database. Query returned no results.")
		} else if err != nil {
			t.Errorf("Error: %v and BatchSensor: %v", err, newAssoc)
		}
	})

	t.Run("Test reassociate to batch", func(t *testing.T) {
		defer cleanAssociations()

		previousAssoc, err := worrywort.AssociateBatchToSensor(batch, sensor, "Testing", nil, db)
		if err != nil {
			t.Fatalf("%v", err)
		}
		n := time.Now()
		previousAssoc.DisassociatedAt = &n
		previousAssoc, err = worrywort.UpdateBatchSensorAssociation(*previousAssoc, db)
		if err != nil {
			t.Fatalf("%v", err)
		}
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"batchId":     batch.Uuid,
				"sensorId":    strconv.Itoa(int(*sensor.Id)),
				"description": "It is associated",
			},
		}

		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)
		var result createAssoc
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v: %v", result, resultData)
		}

		// Test the returned graphql types
		if result.Pl.Typename != "AssociateSensorToBatchPayload" {
			t.Errorf("associateBatchToSensor returned unexpected type: %s", result.Pl.Typename)
		}

		if result.Pl.Assoc.Typename != "BatchSensorAssociation" {
			t.Errorf("associateBatchToSensor returned unexpected type for Assoc: %s", result.Pl.Assoc.Typename)
		}

		if result.Pl.Assoc.Id == previousAssoc.Id {
			t.Errorf("associateBatchToSensor returned previous association id. New association was expected.")
		}
	})

	t.Run("Sensor already associated to same batch", func(t *testing.T) {
		defer cleanAssociations()
		_, err := worrywort.AssociateBatchToSensor(batch, sensor, "Testing", nil, db)
		if err != nil {
			t.Fatalf("%v", err)
		}

		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"batchId":     batch.Uuid,
				"sensorId":    strconv.Itoa(int(*sensor.Id)),
				"description": "It is associated",
			},
		}

		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)
		var result createAssoc
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v: %v", result, resultData)
		}
		if result.Pl.Assoc != nil {
			t.Errorf("Expected null payload in response, got: %v", resultData.Data)
		}

		if resultData.Errors[0].Message != "Sensor already associated to Batch." {
			t.Errorf("Expected query error `Sensor already associated to Batch.`, Got: %v", resultData.Errors)
		}
	})

	t.Run("Batch not owned by user", func(t *testing.T) {
		defer cleanAssociations()
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"batchId":     batch2.Uuid,
				"sensorId":    strconv.Itoa(int(*sensor.Id)),
				"description": "It is associated",
			},
		}

		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)
		var result createAssoc
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v: %v", result, resultData)
		}
		if result.Pl.Assoc != nil {
			t.Errorf("Expected null payload in response, got: %v", resultData.Data)
		}

		if resultData.Errors[0].Message != "Specified Batch does not exist." {
			t.Errorf("Expected query error `Specified Batch does not exist.`, Got: %v", resultData.Errors)
		}
	})

	t.Run("Sensor not owned by user", func(t *testing.T) {
		defer cleanAssociations()
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"batchId":     batch.Uuid,
				"sensorId":    strconv.Itoa(int(*sensor2.Id)),
				"description": "It is associated",
			},
		}

		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)
		var result createAssoc
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v: %v", result, resultData)
		}
		if result.Pl.Assoc != nil {
			t.Errorf("Expected null payload in response, got: %v", resultData.Data)
		}

		if resultData.Errors[0].Message != "Specified Sensor does not exist." {
			t.Errorf("Expected query error `Specified Batch does not exist.`, Got: %v", resultData.Errors)
		}
	})
}

func TestUpdateBatchSensorAssociationMutation(t *testing.T) {

	query := `
		mutation updateBatchSensorAssociation($input: UpdateBatchSensorAssociationInput!) {
			updateBatchSensorAssociation(input: $input) {
				__typename
				batchSensorAssociation {
					__typename
					id
				}
			}
		}`

	// Structs for marshalling json to so that actual values can easily be checked and used
	type payloadAssoc struct {
		Typename string `json:"__typename"`
		Id       string `json:"id"`
	}

	type payload struct {
		Typename string        `json:"__typename"`
		Assoc    *payloadAssoc `json:"BatchSensorAssociation"`
	}

	type createAssoc struct {
		Pl payload `json:"updateBatchSensorAssociation"`
	}

	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.NewUser(nil, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = worrywort.SaveUser(db, u)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor, err := worrywort.SaveSensor(db, worrywort.Sensor{UserId: u.Id, Name: "Test Sensor", CreatedBy: &u})
	if err != nil {
		t.Fatalf("%v", err)
	}

	batch, err := worrywort.SaveBatch(
		db, worrywort.Batch{UserId: u.Id, CreatedBy: &u, Name: "Test batch"})
	if err != nil {
		t.Fatalf("%v", err)
	}

	u2 := worrywort.NewUser(nil, "user2@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u2, err = worrywort.SaveUser(db, u2)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	batch2, err := worrywort.SaveBatch(
		db, worrywort.Batch{UserId: u2.Id, CreatedBy: &u, Name: "Test batch 2"})
	if err != nil {
		t.Fatalf("%v", err)
	}

	sensor2, err := worrywort.SaveSensor(db, worrywort.Sensor{UserId: u2.Id, Name: "Test Sensor 2", CreatedBy: &u2})
	if err != nil {
		t.Fatalf("%v", err)
	}

	batch2, err = worrywort.SaveBatch(
		db, worrywort.Batch{UserId: u2.Id, CreatedBy: &u2, Name: "Test batch 2"})
	if err != nil {
		t.Fatalf("%v", err)
	}

	assoc1, _ := worrywort.AssociateBatchToSensor(batch, sensor, "Description", nil, db)
	assoc2, _ := worrywort.AssociateBatchToSensor(batch, sensor2, "Description", nil, db)
	assoc3, _ := worrywort.AssociateBatchToSensor(batch2, sensor, "Description", nil, db)

	var worrywortSchema = graphql.MustParseSchema(graphqlApi.Schema, graphqlApi.NewResolver(db))

	operationName := ""
	ctx := context.Background()
	ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)
	ctx = context.WithValue(ctx, "db", db)

	// TODO: may want to reset after each test
	// cleanAssociations := func() {
	// 	q := `DELETE FROM batch_sensor_association WHERE sensor_id = ? AND batch_id = ?`
	// 	q = db.Rebind(q)
	// 	_, err := db.Exec(q, sensor.Id, batch.Id)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }

	t.Run("Update with description and disassociatedAt", func(t *testing.T) {
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"id":              assoc1.Id,
				"description":     "Updated description",
				"associatedAt":    "2019-01-01T12:01:01Z",
				"disassociatedAt": "2019-01-01T12:02:01Z",
			},
		}

		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)
		var result createAssoc
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v: %v", result, resultData)
		}

		// Test the returned graphql types
		if result.Pl.Typename != "UpdateBatchSensorAssociationPayload" {
			t.Errorf("updateBatchSensorAssociation returned unexpected type: %s", result.Pl.Typename)
		}

		if result.Pl.Assoc.Typename != "BatchSensorAssociation" {
			t.Errorf("updateBatchSensorAssociation returned unexpected type for Assoc: %s", result.Pl.Assoc.Typename)
		}

		// Make sure it was really created in the db
		newAssoc, _ := worrywort.FindBatchSensorAssociation(
			map[string]interface{}{"id": assoc1.Id}, db)
		// TODO: I do not like this, but it's the easiest way to compare just what I want here. Maybe I should write
		// custom equals() methods and I am sure go-cmp filterPath or filterValue can deal with this.
		newAssoc.Batch = nil
		newAssoc.Sensor = nil

		assocAt, _ := time.Parse(time.RFC3339, "2019-01-01T12:01:01Z")
		disassocAt, _ := time.Parse(time.RFC3339, "2019-01-01T12:02:01Z")
		expected := worrywort.BatchSensor{
			Id: assoc1.Id, Description: "Updated description",
			AssociatedAt: assocAt, DisassociatedAt: &disassocAt, CreatedAt: assoc1.CreatedAt, UpdatedAt: newAssoc.UpdatedAt,
			SensorId: assoc1.SensorId, BatchId: assoc1.BatchId}

		if !cmp.Equal(*newAssoc, expected, nil) {
			t.Errorf(cmp.Diff(newAssoc, expected, nil))
		}
	})

	t.Run("Update with blank description and null disassociatedAt", func(t *testing.T) {
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"id":              assoc1.Id,
				"description":     nil,
				"associatedAt":    "2019-01-01T12:01:01Z",
				"disassociatedAt": nil,
			},
		}

		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)
		var result createAssoc
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v: %v", result, resultData)
		}

		// Test the returned graphql types
		if result.Pl.Typename != "UpdateBatchSensorAssociationPayload" {
			t.Errorf("updateBatchSensorAssociation returned unexpected type: %s", result.Pl.Typename)
		}

		if result.Pl.Assoc.Typename != "BatchSensorAssociation" {
			t.Errorf("updateBatchSensorAssociation returned unexpected type for Assoc: %s", result.Pl.Assoc.Typename)
		}

		// Make sure it was really created in the db
		newAssoc, _ := worrywort.FindBatchSensorAssociation(
			map[string]interface{}{"id": assoc1.Id}, db)
		// TODO: I do not like this, but it's the easiest way to compare just what I want here. Maybe I should write
		// custom equals() methods and I am sure go-cmp filterPath or filterValue can deal with this.
		newAssoc.Batch = nil
		newAssoc.Sensor = nil

		assocAt, _ := time.Parse(time.RFC3339, "2019-01-01T12:01:01Z")
		expected := worrywort.BatchSensor{
			Id: assoc1.Id, Description: "",
			AssociatedAt: assocAt, DisassociatedAt: nil, CreatedAt: assoc1.CreatedAt, UpdatedAt: newAssoc.UpdatedAt,
			SensorId: assoc1.SensorId, BatchId: assoc1.BatchId}

		if !cmp.Equal(*newAssoc, expected, nil) {
			t.Errorf(cmp.Diff(newAssoc, expected, nil))
		}

	})

	t.Run("Batch not owned by user", func(t *testing.T) {
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"id":              assoc2.Id,
				"description":     "It is associated",
				"associatedAt":    "2019-01-01T12:01:01Z",
				"disassociatedAt": "2019-01-01T12:02:01Z",
			},
		}

		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)
		var result createAssoc
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v: %v", result, resultData)
		}
		if result.Pl.Assoc != nil {
			t.Errorf("Expected null payload in response, got: %v", resultData.Data)
		}

		if resultData.Errors[0].Message != "BatchSensorAssociation does not exist." {
			t.Errorf("Expected query error `BatchSensorAssociation does not exist.`, Got: %v", resultData.Errors)
		}
	})

	t.Run("Sensor not owned by user", func(t *testing.T) {
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"id":              assoc3.Id,
				"description":     "It is associated",
				"associatedAt":    "2019-01-01T12:01:01Z",
				"disassociatedAt": "2019-01-01T12:02:01Z",
			},
		}

		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)
		var result createAssoc
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v: %v", result, resultData)
		}
		if result.Pl.Assoc != nil {
			t.Errorf("Expected null payload in response, got: %v", resultData.Data)
		}

		if resultData.Errors[0].Message != "BatchSensorAssociation does not exist." {
			t.Errorf("Expected query error `BatchSensorAssociation does not exist.`, Got: %v", resultData.Errors)
		}
	})
}
