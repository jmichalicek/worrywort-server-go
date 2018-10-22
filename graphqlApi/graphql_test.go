package graphqlApi_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	txdb "github.com/DATA-DOG/go-txdb"
	"github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/graphqlApi"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

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
	b := worrywort.Batch{Name: "Testing", BrewedDate: addMinutes(time.Now(), 1), BottledDate: addMinutes(time.Now(), 10), VolumeBoiled: 5,
		VolumeInFermentor: 4.5, VolumeUnits: worrywort.GALLON, OriginalGravity: 1.060, FinalGravity: 1.020,
		UserId: sql.NullInt64{Int64: int64(u.Id), Valid: true}, BrewNotes: "Brew notes",
		TastingNotes: "Taste notes", RecipeURL: "http://example.org/beer"}
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
	user := worrywort.NewUser(0, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
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
		result := worrywortSchema.Exec(context, query, operationName, variables)

		// example:
		// {"login":{"token":"c9d103e1-8320-45fd-8ac6-245d59c01b3d:HRXG69cqTv1kyG6zmsJo0tJNsEKmeCqWH5WeH3H-_IyTHZ46ivz0KyTTfUgun1CNCV3n1HLwizvAET1I2DwJiA=="}}
		// the hash, the part of the token after the colon, is a base64 encoded sha512 sum
		// Testing this pattern this far may be a bit overtesting.  Could just test for any string as the token.
		expected := `\{"login":\{"token":"(.+):([-A-Za-z0-9/+_]+=*)"\}\}`
		matcher := regexp.MustCompile(expected)
		// TODO: capture the token and make sure there's an entry in the db for it.
		matched := matcher.Match(result.Data)

		if !matched {
			t.Errorf("\nExpected response to match pattern: %s\nGot: %s", expected, result.Data)
		}

		// Make sure that the token really was inserted into the db
		subMatches := matcher.FindStringSubmatch(string(result.Data))
		tokenId := subMatches[1]
		// tokenSecret := subMatches[2]
		newToken := worrywort.AuthToken{}
		query = db.Rebind(
			`SELECT t.token_id, t.token, t.scope, t.expires_at, t.created_at, t.updated_at, u.id "user.id", u.first_name "user.first_name", u.last_name "user.last_name", ` +
				`u.email "user.email", u.created_at "user.created_at", u.updated_at "user.updated_at", u.password "user.password" FROM user_authtokens t LEFT JOIN users u ON t.user_id = u.id ` +
				`WHERE t.token_id = ?`)
		err := db.Get(&newToken, query, tokenId)

		if err != nil {
			t.Errorf("Error looking up newly created token: %v", err)
		}

		if newToken == (worrywort.AuthToken{}) {
			t.Errorf("Expected auth token with id %s to be saved to database", tokenId)
		}

		if newToken.User.Id != user.Id {
			t.Errorf("Expected auth token to be associated with user %v but it is associated with %v", user, newToken.User)
		}
	})
}

func TestBatchQuery(t *testing.T) {
	const DefaultUserKey string = "user"
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.NewUser(0, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = worrywort.SaveUser(db, u)

	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	u2 := worrywort.NewUser(0, "user2@example.com", "Justin", "M", time.Now(), time.Now())
	u2, err = worrywort.SaveUser(db, u2)

	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

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
			"id": strconv.Itoa(b.Id),
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
		ctx := context.Background()
		ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)
		result := worrywortSchema.Exec(ctx, query, operationName, variables)

		var expected interface{}
		err := json.Unmarshal([]byte(fmt.Sprintf(`{"batch": {"__typename": "Batch", "id": "%d"}}`, b.Id)), &expected)
		if err != nil {
			t.Fatalf("%v", err)
		}

		var f interface{}
		err = json.Unmarshal(result.Data, &f)
		if err != nil {
			t.Fatalf("%v", f)
		}

		if !reflect.DeepEqual(expected, f) {
			t.Errorf("\nExpected: %v\nGot: %v", expected, f)
		}
	})

	t.Run("Test query for batch(id: ID!) which does not exist returns null", func(t *testing.T) {
		variables := map[string]interface{}{
			"id": "-1",
		}
		query := `
			query getBatch($id: ID!) {
				batch(id: $id) {
					id
				}
			}
		`
		operationName := ""
		ctx := context.Background()
		ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)
		result := worrywortSchema.Exec(ctx, query, operationName, variables)

		expected := `{"batch":null}`
		if expected != string(result.Data) {
			t.Errorf("Expected: %s\nGot: %s", expected, result.Data)
		}
	})

	t.Run("Test batches() query returns the users batches", func(t *testing.T) {
		query := `
			query getBatches {
				batches {
					__typename
					id
				}
			}
		`
		operationName := ""
		ctx := context.Background()
		ctx = context.WithValue(ctx, "db", db)
		ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)
		result := worrywortSchema.Exec(ctx, query, operationName, nil)

		var expected interface{}
		err := json.Unmarshal(
			[]byte(fmt.Sprintf(`{"batches":[{"__typename":"Batch","id":"%d"},{"__typename":"Batch","id":"%d"}]}`, b.Id, b2.Id)),
			&expected)
		if err != nil {
			t.Fatalf("%v", err)
		}

		var f interface{}
		err = json.Unmarshal(result.Data, &f)
		if err != nil {
			t.Fatalf("%v", f)
		}

		if !reflect.DeepEqual(expected, f) {
			t.Errorf("\nExpected: %v\nGot: %v", expected, f)
		}
	})

	t.Run("Test batches() query when not authenticated", func(t *testing.T) {
		query := `
			query getBatches {
				batches {
					__typename
					id
				}
			}
		`
		operationName := ""
		ctx := context.Background()
		// ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)
		result := worrywortSchema.Exec(ctx, query, operationName, nil)

		var expected interface{}
		// should we actually return auth errors rather than nothing?
		err := json.Unmarshal([]byte(`{"batches":[]}`), &expected)
		if err != nil {
			t.Fatalf("%v", err)
		}

		var f interface{}
		err = json.Unmarshal(result.Data, &f)
		if err != nil {
			t.Fatalf("%v", f)
		}

		if !reflect.DeepEqual(expected, f) {
			t.Errorf("\nExpected: %v\nGot: %v", expected, f)
		}
	})
}

func TestCreateTemperatureMeasurementMutation(t *testing.T) {
	const DefaultUserKey string = "user"
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.NewUser(0, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = worrywort.SaveUser(db, u)
	userId := sql.NullInt64{Valid: true, Int64: int64(u.Id)}

	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor, err := worrywort.SaveTemperatureSensor(db, worrywort.TemperatureSensor{UserId: userId, Name: "Test Sensor", CreatedBy: &u})
	if err != nil {
		t.Fatalf("%v", err)
	}
	sensorId := sql.NullInt64{Valid: true, Int64: int64(sensor.Id)}

	batch, err := worrywort.SaveBatch(
		db, worrywort.Batch{UserId: userId, CreatedBy: &u, Name: "Test batch"})
	if err != nil {
		t.Fatalf("%v", err)
	}
	batchId := sql.NullInt64{Valid: true, Int64: int64(batch.Id)}

	u2 := worrywort.NewUser(0, "user2@example.com", "Justin", "M", time.Now(), time.Now())
	u2, err = worrywort.SaveUser(db, u2)

	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	// TODO: Can this become global to these tests?
	var worrywortSchema = graphql.MustParseSchema(graphqlApi.Schema, graphqlApi.NewResolver(db))
	t.Run("Test measurement is created with valid data", func(t *testing.T) {
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"batchId":             strconv.Itoa(int(batchId.Int64)),
				"temperatureSensorId": strconv.Itoa(int(sensorId.Int64)),
				"units":               "FAHRENHEIT",
				"temperature":         70.0,
				"recordedAt":          "2018-10-14T15:26:00+00:00",
			},
		}
		query := `
			mutation addMeasurement($input: CreateTemperatureMeasurementInput) {
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

		selectCols := fmt.Sprintf("tm.user_id, tm.temperature_sensor_id")
		q := `SELECT tm.temperature, tm.units,  ` + strings.Trim(selectCols, ", ") + ` from temperature_measurements tm WHERE tm.id = ? AND tm.user_id = ? AND tm.temperature_sensor_id = ?`
		query = db.Rebind(q)
		err = db.Get(measurement, query, measurementId, userId, sensorId)

		if err == sql.ErrNoRows {
			t.Error("Measurement was not saved to the database. Query returned no results.")
		} else if err != nil {
			t.Errorf("%v", err)
		}

	})
}
