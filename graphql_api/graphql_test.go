package graphql_api_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	txdb "github.com/DATA-DOG/go-txdb"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/graph-gophers/graphql-go"
	graphqlErrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/graphql_api"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	// "log"
)

const DefaultUserKey string = "user"

// Helper structs for deserializing responses, etc.
type node struct {
	Typename string `json:"__typename"`
	Id       string `json:"id"`
}

type edge struct {
	Typename string `json:"__typename"`
	Cursor   string `json:"cursor"`
	Node     node   `json:"Node"`
}

type pageInfo struct {
	HasNextPage     bool `json:"hasNextPage"`
	HasPreviousPage bool `json:"hasPreviousPage"`
}

type connection struct {
	Typename string   `json:"__typename"`
	PageInfo pageInfo `json:"pageInfo"`
	Edges    []edge   `json:"Edges"`
}

var encodedOffset1 string = base64.StdEncoding.EncodeToString([]byte(graphql_api.MakeOffsetCursorP(1)))
var encodedOffset2 string = base64.StdEncoding.EncodeToString([]byte(graphql_api.MakeOffsetCursorP(2)))

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
	bottledDate := addMinutes(time.Now(), 10)
	b := worrywort.Batch{Name: "Testing", BrewedDate: addMinutes(time.Now(), 1),
		BottledDate: &bottledDate, VolumeBoiled: 5, VolumeInFermentor: 4.5, VolumeUnits: worrywort.GALLON,
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

	var worrywortSchema = graphql.MustParseSchema(graphql_api.Schema, graphql_api.NewResolver(db))
	user := worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	// This is the hash for the password `password`
	// var hashedPassword string = "$2a$13$pPg7mwPA.VFf3W9AUZyMGO0Q2nhoh/979F/TZ8ED.iqVubLe.TDmi"
	err = worrywort.SetUserPassword(&user, "password", bcrypt.MinCost)
	err = user.Save(db)
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
			Token string `json:"token"`
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
			`SELECT t.id, t.token, t.scope, t.expires_at, t.created_at, t.updated_at, u.id "user.id", u.uuid "user.uuid",
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
			t.Errorf("Expected: - | Got +\n%s", cmp.Diff(newToken.User, user))
			// t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})
}

func TestCurrentUserQuery(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	var worrywortSchema = graphql.MustParseSchema(graphql_api.Schema, graphql_api.NewResolver(db))
	query := `
		query currentUser {
			currentUser {
				__typename
				id
			}
		}
	`
	operationName := ""

	t.Run("Authenticated", func(t *testing.T) {
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
		result := worrywortSchema.Exec(ctx, query, operationName, nil)
		var expected interface{}
		err = json.Unmarshal([]byte(fmt.Sprintf(`{"currentUser": {"__typename": "User", "id": "%s"}}`, u.UUID)), &expected)
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
		}
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, nil)
		result := worrywortSchema.Exec(ctx, query, operationName, nil)
		var expected interface{}
		err = json.Unmarshal([]byte(`{"currentUser": null}`), &expected)
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
		}
	})
}

func TestBatchQuery(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	u2 := worrywort.User{Email: "user2@example.com", FirstName: "Justin", LastName: "M"}
	err = u2.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	// TODO: use my middleware here to add the user to the context?
	ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)

	// TODO: Can this become global to these tests?
	var worrywortSchema = graphql.MustParseSchema(graphql_api.Schema, graphql_api.NewResolver(db))

	b := makeTestBatch(u, true)
	err = b.Save(db)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	b2 := makeTestBatch(u, true)
	err = b2.Save(db)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	u2batch := makeTestBatch(u2, true)
	err = u2batch.Save(db)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	t.Run("Test query for batch(id: ID!) which exists returns the batch", func(t *testing.T) {
		variables := map[string]interface{}{
			"id": b.UUID,
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
		err := json.Unmarshal([]byte(fmt.Sprintf(`{"batch": {"__typename": "Batch", "id": "%s"}}`, b.UUID)), &expected)
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
		badUUID := uuid.New().String()
		variables := map[string]interface{}{
			"id": badUUID,
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

	t.Run("Unauthenticated batch() returns null", func(t *testing.T) {
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, nil)
		variables := map[string]interface{}{
			"id": b.UUID,
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

	// TODO: Id's in the `expected` here will change once I start properly base64 encoding and maybe
	// adding type info
	// TODO: again, maybe make structs for the responses
	var response_1 = `{"batches": {
		"__typename":"BatchConnection",
		"pageInfo": {"hasNextPage": %s, "hasPreviousPage": %s},
		"edges": [{"__typename": "BatchEdge", "cursor": "%s", "node": {"__typename":"Batch","id":"%s"}}]}}`
	var response_2 = `{"batches": {
		"__typename":"BatchConnection",
		"pageInfo": {"hasNextPage": false, "hasPreviousPage": false},
		"edges": [{"__typename": "BatchEdge", "cursor": "%s", "node": {"__typename":"Batch","id":"%s"}},
					{"__typename": "BatchEdge", "cursor": "%s",  "node": {"__typename":"Batch","id":"%s"}}]}}`
	after0cursor := fmt.Sprintf("%s", base64.StdEncoding.EncodeToString([]byte(graphql_api.MakeOffsetCursorP(1))))
	after1cursor := fmt.Sprintf("%s", base64.StdEncoding.EncodeToString([]byte(graphql_api.MakeOffsetCursorP(2))))
	var testargs = []struct {
		Name      string
		Variables map[string]interface{}
		Expected  []byte
	}{
		{
			Name:      "Batches()",
			Variables: map[string]interface{}{},
			Expected:  []byte(fmt.Sprintf(response_2, after0cursor, b.UUID, after1cursor, b2.UUID))},
		{
			Name:      "Batches(first: 1)",
			Variables: map[string]interface{}{"first": 1},
			Expected:  []byte(fmt.Sprintf(response_1, "true", "false", after0cursor, b.UUID))},
		{
			Name:      "Batches(after: FIRST_CURSOR)",
			Variables: map[string]interface{}{"after": after0cursor},
			Expected:  []byte(fmt.Sprintf(response_1, "false", "false", after1cursor, b2.UUID))},
	}

	for _, qt := range testargs {
		t.Run(qt.Name, func(t *testing.T) {
			var expected interface{}
			err = json.Unmarshal(qt.Expected, &expected)
			if err != nil {
				t.Errorf("%v", err)
			}

			query := `query getBatches($first: Int $after: String) {
					batches(first: $first after: $after) {
						__typename
						pageInfo {hasPreviousPage hasNextPage}
						edges {
							__typename cursor
							node { __typename id }
						}
					}
				}`
			operationName := ""
			ctx := context.Background()
			ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
			ctx = context.WithValue(ctx, "db", db)
			resultData := worrywortSchema.Exec(ctx, query, operationName, qt.Variables)
			var result interface{}
			err = json.Unmarshal(resultData.Data, &result)
			if err != nil {
				t.Fatalf("%v: %v", result, resultData)
			}
			if !cmp.Equal(expected, result) {
				t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(expected, result))
			}
		})
	}

	t.Run("Test batches() query when not authenticated", func(t *testing.T) {
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

		var actual interface{}
		err = json.Unmarshal(result.Data, &actual)
		if err != nil {
			t.Fatalf("%v", err)
		}

		if actual != nil {
			t.Fatalf("Expected nil, Got: +\n%s", spew.Sdump(actual))
		}
	})
}

func TestCreateTemperatureMeasurementMutation(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor := worrywort.Sensor{UserId: u.Id, Name: "Test Sensor", CreatedBy: &u}
	if err := sensor.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	u2 := worrywort.User{Email: "user2@example.com", FirstName: "Justin", LastName: "M"}
	if err = u2.Save(db); err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	// TODO: Can this become global to these tests?
	var worrywortSchema = graphql.MustParseSchema(graphql_api.Schema, graphql_api.NewResolver(db))
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
	ctx = context.WithValue(ctx, "db", db)

	cleanMeasurements := func() {
		q := `DELETE FROM temperature_measurements;`
		q = db.Rebind(q)
		_, err := db.Exec(q)
		if err != nil {
			panic(err)
		}
	}
	defer cleanMeasurements()
	t.Run("Test measurement is created with valid data", func(t *testing.T) {
		defer cleanMeasurements()
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)

		// Some structs so that the json can be unmarshalled
		type createTemperatureMeasurementPayload struct {
			Typename               string `json:"__typename"`
			TemperatureMeasurement node   `json:"temperatureMeasurement"`
		}

		type createTemperatureMeasurement struct {
			CreateTemperatureMeasurement createTemperatureMeasurementPayload `json:"createTemperatureMeasurement"`
		}

		var result createTemperatureMeasurement
		if err = json.Unmarshal(resultData.Data, &result); err != nil {
			t.Fatalf("Error: %s for result %v", err, result)
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
		measurement, err := worrywort.FindTemperatureMeasurement(
			map[string]interface{}{"user_id": *u.Id, "id": measurementId}, db)
		if err == sql.ErrNoRows {
			t.Error("Measurement was not saved to the database. Query returned no results.")
		} else if err != nil {
			t.Errorf("%v", err)
		}
		if measurement == nil {
			t.Errorf("Measurement lookup returned nil")
		}
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		defer cleanMeasurements()
		// cleanMeasurements()
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, nil)
		result := worrywortSchema.Exec(ctx, query, operationName, variables)
		var expected interface{}
		err = json.Unmarshal([]byte(`{"createTemperatureMeasurement": null}`), &expected)
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
		}

		// Make sure that it really was not created
		tm, err := worrywort.FindTemperatureMeasurements(map[string]interface{}{"user_id": u.Id}, db)
		if err != nil {
			t.Errorf("Unexpected error: %s", err)
		}
		if len(tm) != 0 {
			t.Errorf("Expected no results but got: %s", spew.Sdump(tm))
		}

	})
}

func TestSensorQuery(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	// TODO: This whole create 2 users and setup context is done A LOT here. can it be de-duplicated?
	u := worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)

	u2 := worrywort.User{Email: "user2@example.com", FirstName: "Justin", LastName: "M"}
	err = u2.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	// TODO: Can this become global to these tests?
	var worrywortSchema = graphql.MustParseSchema(graphql_api.Schema, graphql_api.NewResolver(db))
	sensor1 := worrywort.Sensor{Name: "Sensor 1", UserId: u.Id}
	if err := sensor1.Save(db); err != nil {
		t.Fatalf("%v", err)
	}
	sensor2 := worrywort.Sensor{Name: "Sensor 2", UserId: u.Id}
	if err = sensor2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}
	// Need one owned by another user to ensure it does not show up
	sensor3 := worrywort.Sensor{Name: "Sensor 3", UserId: u2.Id}
	if err = sensor3.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	t.Run("Test query for sensor(id: ID!) which exists returns the sensor", func(t *testing.T) {
		variables := map[string]interface{}{
			"id": sensor1.UUID,
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
		err := json.Unmarshal([]byte(fmt.Sprintf(`{"sensor": {"__typename": "Sensor", "id": "%s"}}`, sensor1.UUID)), &expected)
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
			"id": uuid.New().String(),
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

	t.Run("Sensors()", func(t *testing.T) {
		type sensorsResponse struct {
			SensorConnection connection `json:"sensors"`
		}

		var testmatrix = []struct {
			name      string
			variables map[string]interface{}
			expected  sensorsResponse
		}{
			// basic filters
			// This is ok for now, but really don't want to write one test per potential filter as those grow
			// will at least add user uuid probably.
			{"Unfiltered", map[string]interface{}{},
				sensorsResponse{
					connection{
						Typename: "SensorConnection",
						PageInfo: pageInfo{false, false},
						Edges: []edge{
							edge{Typename: "SensorEdge", Cursor: encodedOffset1,
								Node: node{Typename: "Sensor", Id: sensor1.UUID}},
							edge{Typename: "SensorEdge", Cursor: encodedOffset2,
								Node: node{Typename: "Sensor", Id: sensor2.UUID},
							},
						},
					},
				},
			},
			// Pagination tests
			{"sensors(first: 1)", map[string]interface{}{"first": 1},
				sensorsResponse{
					connection{
						Typename: "SensorConnection",
						PageInfo: pageInfo{HasNextPage: true, HasPreviousPage: false},
						Edges: []edge{
							edge{Typename: "SensorEdge", Cursor: encodedOffset1,
								Node: node{Typename: "Sensor", Id: sensor1.UUID}},
						},
					},
				},
			},
			{"sensors(after: <encodedOffset1>)", map[string]interface{}{"after": encodedOffset1},
				sensorsResponse{
					connection{
						Typename: "SensorConnection",
						PageInfo: pageInfo{false, false},
						Edges: []edge{
							edge{Typename: "SensorEdge", Cursor: encodedOffset2,
								Node: node{Typename: "Sensor", Id: sensor2.UUID}},
						},
					},
				},
			},
		}

		for _, tm := range testmatrix {
			t.Run(tm.name, func(t *testing.T) {
				query := `
					query getSensors($first: Int $after: String) {
						sensors(first: $first after: $after) {
							__typename
							pageInfo {hasNextPage hasPreviousPage}
							edges {
								cursor
								__typename
								node {
									__typename
									id
								}
							}
						}
					}`
				operationName := ""
				ctx := context.Background()
				ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
				ctx = context.WithValue(ctx, "db", db)
				resultData := worrywortSchema.Exec(ctx, query, operationName, tm.variables)

				var result sensorsResponse
				if err = json.Unmarshal(resultData.Data, &result); err != nil {
					t.Fatalf("Error: %s for result %v", err, result)
				}

				if err != nil {
					t.Fatalf("%v: %v", result, resultData)
				}
				if !cmp.Equal(tm.expected, result) {
					t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(tm.expected, result))
				}
			})
		}

		t.Run("Unauthenticated", func(t *testing.T) {
			query := `
				query getSensors($first: Int $after: String) {
					sensors(first: $first after: $after) {
						__typename
						pageInfo {hasNextPage hasPreviousPage}
						edges {
							cursor
							__typename
							node {
								__typename
								id
							}
						}
					}
				}`
			operationName := ""
			ctx := context.Background()
			ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, nil)
			ctx = context.WithValue(ctx, "db", db)
			result := worrywortSchema.Exec(ctx, query, operationName, nil)

			// TODO: Make this error checking reusable
			e := graphqlErrors.QueryError{Message: "User must be authenticated"}
			expectedErrors := []*graphqlErrors.QueryError{&e}
			cmpOpts := []cmp.Option{
				cmpopts.IgnoreFields(e, "ResolverError", "Path", "Extensions", "Rule", "Locations"),
			}
			if !cmp.Equal(expectedErrors, result.Errors, cmpOpts...) {
				t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(expectedErrors, result.Errors, cmpOpts...))
			}
			// End error checking
			var actual interface{}
			err = json.Unmarshal(result.Data, &actual)
			if err != nil {
				t.Fatalf("%v", err)
			}
			if actual != nil {
				t.Errorf("Expected nil, Got: %s", spew.Sdump(actual))
			}
		})
	})
}

func TestCreateBatchMutation(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	var worrywortSchema = graphql.MustParseSchema(graphql_api.Schema, graphql_api.NewResolver(db))
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
		ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
		ctx = context.WithValue(ctx, "db", db)
		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)

		// Some structs so that the json can be unmarshalled.
		type cbPayload struct {
			Typename string `json:"__typename"`
			Batch    node   `json:"batch"`
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

	t.Run("Unauthenticated", func(t *testing.T) {
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
		ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, nil)
		ctx = context.WithValue(ctx, "db", db)
		result := worrywortSchema.Exec(ctx, query, operationName, variables)

		// TODO: Make this error checking reusable
		e := graphqlErrors.QueryError{Message: "User must be authenticated"}
		expectedErrors := []*graphqlErrors.QueryError{&e}
		cmpOpts := []cmp.Option{
			cmpopts.IgnoreFields(e, "ResolverError", "Path", "Extensions", "Rule", "Locations"),
		}
		if !cmp.Equal(expectedErrors, result.Errors, cmpOpts...) {
			t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(expectedErrors, result.Errors, cmpOpts...))
		}
		// End error checking
		var expected interface{}
		err = json.Unmarshal([]byte(`{"createBatch": null}`), &expected)
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
		}

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

	type payload struct {
		Typename string `json:"__typename"`
		Assoc    *node  `json:"BatchSensorAssociation"`
	}

	type createAssoc struct {
		Pl payload `json:"associateSensorToBatch"`
	}

	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor := worrywort.Sensor{UserId: u.Id, Name: "Test Sensor", CreatedBy: &u}
	if err := sensor.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	batch := worrywort.Batch{UserId: u.Id, CreatedBy: &u, Name: "Test batch"}
	if err := batch.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	u2 := worrywort.User{Email: "user2@example.com", FirstName: "Justin", LastName: "Michalicek"}
	if err := u2.Save(db); err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor2 := worrywort.Sensor{UserId: u2.Id, Name: "Test Sensor 2", CreatedBy: &u2}
	if err := sensor2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	batch2 := worrywort.Batch{UserId: u2.Id, CreatedBy: &u2, Name: "Test batch 2"}
	if err = batch2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	var worrywortSchema = graphql.MustParseSchema(graphql_api.Schema, graphql_api.NewResolver(db))
	operationName := ""
	ctx := context.Background()
	ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
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
				"batchId":     batch.UUID,
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

		previousAssoc, err := worrywort.AssociateBatchToSensor(&batch, &sensor, "Testing", nil, db)
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
				"batchId":     batch.UUID,
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
		_, err := worrywort.AssociateBatchToSensor(&batch, &sensor, "Testing", nil, db)
		if err != nil {
			t.Fatalf("%v", err)
		}

		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"batchId":     batch.UUID,
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
				"batchId":     batch2.UUID,
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
				"batchId":     batch.UUID,
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

	t.Run("Unauthenticated", func(t *testing.T) {
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, nil)
		defer cleanAssociations()
		variables := map[string]interface{}{
			"input": map[string]interface{}{
				"batchId":     batch.UUID,
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

		if resultData.Errors[0].Message != "User must be authenticated" {
			t.Errorf("Expected query error `User must be authenticated`, Got: %v", resultData.Errors)
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

	u := worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor := worrywort.Sensor{UserId: u.Id, Name: "Test Sensor", CreatedBy: &u}
	if err := sensor.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	batch := worrywort.Batch{UserId: u.Id, CreatedBy: &u, Name: "Test batch"}
	if err = batch.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	u2 := worrywort.User{Email: "user2@example.com", FirstName: "Justin", LastName: "Michalicek"}
	if err = u2.Save(db); err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	batch2 := worrywort.Batch{UserId: u2.Id, CreatedBy: &u2, Name: "Test batch 2"}
	if err = batch2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	sensor2 := worrywort.Sensor{UserId: u2.Id, Name: "Test Sensor 2", CreatedBy: &u2}
	if err := sensor2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	// assoc2 and assoc3 test some bad states which should not happen, but making sure the api handles it
	// safely in case they somehow do.
	// TODO: really should be t.Fatal if there's an error here...
	assoc1, _ := worrywort.AssociateBatchToSensor(&batch, &sensor, "Description", nil, db)
	assoc2, _ := worrywort.AssociateBatchToSensor(&batch, &sensor2, "Description", nil, db)
	assoc3, _ := worrywort.AssociateBatchToSensor(&batch2, &sensor, "Description", nil, db)

	var worrywortSchema = graphql.MustParseSchema(graphql_api.Schema, graphql_api.NewResolver(db))

	operationName := ""
	ctx := context.Background()
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

	// TODOL make a test table
	t.Run("Update with description and disassociatedAt", func(t *testing.T) {
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
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
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
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
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
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
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
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

	t.Run("Unauthenticated", func(t *testing.T) {
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, nil)
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

		if resultData.Errors[0].Message != "User must be authenticated" {
			t.Errorf("Expected query error `User must be authenticated`, Got: %v", resultData.Errors)
		}
	})
}

func TestBatchSensorAssociationsQuery(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor := worrywort.Sensor{UserId: u.Id, Name: "Test Sensor", CreatedBy: &u}
	if err := sensor.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	batch := worrywort.Batch{UserId: u.Id, CreatedBy: &u, Name: "Test batch"}
	if err = batch.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	batch2 := worrywort.Batch{UserId: u.Id, CreatedBy: &u, Name: "Test batch 2"}
	if err = batch2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	sensor2 := worrywort.Sensor{UserId: u.Id, Name: "Test Sensor 2", CreatedBy: &u}
	if err := sensor2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	assoc1, _ := worrywort.AssociateBatchToSensor(&batch, &sensor, "Description", nil, db)
	assoc2, _ := worrywort.AssociateBatchToSensor(&batch2, &sensor2, "Description", nil, db)
	assoc1cursor := fmt.Sprintf("%s", base64.StdEncoding.EncodeToString([]byte(graphql_api.MakeOffsetCursorP(1))))
	assoc2cursor := fmt.Sprintf("%s", base64.StdEncoding.EncodeToString([]byte(graphql_api.MakeOffsetCursorP(2))))

	// if needed, make actual types for the returned structs to unmarshal to
	var testargs = []struct {
		Name      string
		Variables map[string]interface{}
		Expected  []byte
	}{
		{
			Name:      "BatchSensorAssociation()",
			Variables: map[string]interface{}{},
			Expected: []byte(fmt.Sprintf(`{"batchSensorAssociations": {
					"__typename": "BatchSensorAssociationConnection",
				  	"pageInfo": {"hasPreviousPage": false, "hasNextPage": false},
					"edges": [
						{"__typename": "BatchSensorAssociationEdge", "cursor": "%s", "node": {"__typename":"BatchSensorAssociation","id":"%s"}},
						{"__typename": "BatchSensorAssociationEdge", "cursor": "%s", "node": {"__typename":"BatchSensorAssociation","id":"%s"}}
					]}}`,
				assoc1cursor, assoc1.Id, assoc2cursor, assoc2.Id))},
		{
			Name:      "BatchSensorAssociation(first: 1)",
			Variables: map[string]interface{}{"first": 1},
			Expected: []byte(fmt.Sprintf(
				`{"batchSensorAssociations": {
				  "__typename":"BatchSensorAssociationConnection",
				  "pageInfo": {"hasPreviousPage": false, "hasNextPage": true},
				  "edges": [
				  	{"__typename": "BatchSensorAssociationEdge", "cursor": "%s", "node": {"__typename":"BatchSensorAssociation","id":"%s"}}
				  ]}}`,
				assoc1cursor, assoc1.Id))},
		{
			Name:      "BatchSensorAssociation(after: FIRST_CURSOR)",
			Variables: map[string]interface{}{"after": assoc1cursor},
			Expected: []byte(fmt.Sprintf(
				`{"batchSensorAssociations": {
				  "__typename":"BatchSensorAssociationConnection",
				  "pageInfo": {"hasPreviousPage": false, "hasNextPage": false},
				  "edges": [
				  	{"__typename": "BatchSensorAssociationEdge", "cursor": "%s", "node": {"__typename":"BatchSensorAssociation","id":"%s"}}
				  ]}}`,
				assoc2cursor, assoc2.Id))},
		{
			Name:      "BatchSensorAssociation(batchId: BATCH_1_ID)",
			Variables: map[string]interface{}{"batchId": batch.UUID},
			Expected: []byte(fmt.Sprintf(
				`{"batchSensorAssociations": {
				  "__typename":"BatchSensorAssociationConnection",
				  "pageInfo": {"hasPreviousPage": false, "hasNextPage": false},
				  "edges": [
				  	{"__typename": "BatchSensorAssociationEdge", "cursor": "%s", "node": {"__typename":"BatchSensorAssociation","id":"%s"}}
				  ]}}`,
				assoc1cursor, assoc1.Id))},
		{
			Name:      "BatchSensorAssociation(sensorId: SENSOR2_ID)",
			Variables: map[string]interface{}{"sensorId": sensor2.UUID},
			Expected: []byte(fmt.Sprintf(
				`{"batchSensorAssociations": {
				  "__typename":"BatchSensorAssociationConnection",
				  "pageInfo": {"hasPreviousPage": false, "hasNextPage": false},
				  "edges": [
				  	{"__typename": "BatchSensorAssociationEdge", "cursor": "%s", "node": {"__typename":"BatchSensorAssociation","id":"%s"}}
				  ]}}`,
				assoc1cursor, assoc2.Id))}, // cursor is badly named - if only assoc2 is returned, it has that cursor
	}

	var worrywortSchema = graphql.MustParseSchema(graphql_api.Schema, graphql_api.NewResolver(db))
	query := strings.Trim(`
			query getBatchSensorAssociations($first: Int $after: String $batchId: ID $sensorId: ID) {
				batchSensorAssociations(first: $first after: $after batchId: $batchId sensorId: $sensorId) {
					__typename pageInfo {hasPreviousPage hasNextPage}
					edges {
						__typename cursor node { __typename id }
					}
				}
			}`, " ")
	operationName := ""
	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	for _, qt := range testargs {
		t.Run(qt.Name, func(t *testing.T) {
			var expected interface{}
			err = json.Unmarshal(qt.Expected, &expected)
			if err != nil {
				t.Errorf("%s", err)
			}
			ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
			resultData := worrywortSchema.Exec(ctx, query, operationName, qt.Variables)
			var result interface{}
			err = json.Unmarshal(resultData.Data, &result)
			if err != nil {
				t.Fatalf("%v: %v", result, resultData)
			}
			if !cmp.Equal(expected, result) {
				t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(expected, result))
			}
		})
	}
	t.Run("Unauthenticated", func(t *testing.T) {
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, nil)
		resultData := worrywortSchema.Exec(ctx, query, operationName, nil)
		var result interface{}
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v: %v", result, resultData)
		}
		if result != nil {
			t.Errorf("Expected nil, Got: %s", spew.Sdump(result))
		}

	})
}

func TestTemperatureMeasurementsQuery(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	// TODO: must be a good way to shorten this setup model creation... function which takes count of
	// users to create, etc. I suppose.
	u := worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	u2 := worrywort.User{Email: "user2@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u2.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	s1 := worrywort.Sensor{Name: "Test Sensor", UserId: u.Id}
	if err := s1.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	s2 := worrywort.Sensor{Name: "Test Sensor", UserId: u2.Id}
	if err := s2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	// TODO: make batch, associate with sensor, and test
	b := makeTestBatch(u, false)
	if err := b.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	assoc1, err := worrywort.AssociateBatchToSensor(&b, &s1, "", &b.BrewedDate, db)
	if err != nil {
		t.Fatalf("%v", err)
	}

	m1 := worrywort.TemperatureMeasurement{UserId: u.Id, SensorId: s1.Id, Temperature: 70.0, Units: worrywort.FAHRENHEIT,
		RecordedAt: addMinutes(b.BrewedDate, -1)}
	if err := m1.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	m2 := worrywort.TemperatureMeasurement{UserId: u.Id, SensorId: s1.Id, Temperature: 70.0, Units: worrywort.FAHRENHEIT,
		RecordedAt: addMinutes(assoc1.AssociatedAt, 1)}
	if err := m2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	m3 := worrywort.TemperatureMeasurement{UserId: u2.Id, SensorId: s2.Id, Temperature: 71.0, Units: worrywort.FAHRENHEIT,
		RecordedAt: time.Now().Round(time.Microsecond)}
	if err := m3.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	type tm struct {
		Typename string `json:"__typename"`
		Id       string `json:"id"`
	}
	type temperatureMeasurementEdge struct {
		Typename string `json:"__typename"`
		Cursor   string `json:"cursor"`
		Node     tm     `json:"Node"`
	}

	type pageInfo struct {
		HasNextPage     bool `json:"hasNextPage"`
		HasPreviousPage bool `json:"hasPreviousPage"`
	}

	type temperatureMeasurementsConnection struct {
		Typename string                       `json:"__typename"`
		PageInfo pageInfo                     `json:"pageInfo"`
		Edges    []temperatureMeasurementEdge `json:"Edges"`
	}

	type tmResponse struct {
		TemperatureMeasurements temperatureMeasurementsConnection `json:"temperatureMeasurements"`
	}

	var testmatrix = []struct {
		name      string
		variables map[string]interface{}
		expected  tmResponse
	}{
		// basic filters
		// This is ok for now, but really don't want to write one test per potential filter as those grow
		// will at least add user uuid probably.
		{"Unfiltered", map[string]interface{}{},
			tmResponse{
				temperatureMeasurementsConnection{
					Typename: "TemperatureMeasurementConnection",
					PageInfo: pageInfo{false, false},
					Edges: []temperatureMeasurementEdge{
						temperatureMeasurementEdge{Typename: "TemperatureMeasurementEdge", Cursor: encodedOffset1,
							Node: tm{Typename: "TemperatureMeasurement", Id: m1.Id}},
						temperatureMeasurementEdge{Typename: "TemperatureMeasurementEdge", Cursor: encodedOffset2,
							Node: tm{Typename: "TemperatureMeasurement", Id: m2.Id},
						},
					},
				},
			},
		},
		{"temperatureMeasurements(batchId: <batch1>)", map[string]interface{}{"batchId": b.UUID},
			tmResponse{
				temperatureMeasurementsConnection{
					Typename: "TemperatureMeasurementConnection",
					PageInfo: pageInfo{false, false},
					Edges: []temperatureMeasurementEdge{
						temperatureMeasurementEdge{Typename: "TemperatureMeasurementEdge", Cursor: encodedOffset1,
							Node: tm{Typename: "TemperatureMeasurement", Id: m2.Id}},
					},
				},
			},
		},
		{"temperatureMeasurements(sensorId: <s1.UUID>)", map[string]interface{}{"sensorId": s1.UUID},
			tmResponse{
				temperatureMeasurementsConnection{
					Typename: "TemperatureMeasurementConnection",
					PageInfo: pageInfo{false, false},
					Edges: []temperatureMeasurementEdge{
						temperatureMeasurementEdge{Typename: "TemperatureMeasurementEdge", Cursor: encodedOffset1,
							Node: tm{Typename: "TemperatureMeasurement", Id: m1.Id}},
						temperatureMeasurementEdge{Typename: "TemperatureMeasurementEdge", Cursor: encodedOffset2,
							Node: tm{Typename: "TemperatureMeasurement", Id: m2.Id},
						},
					},
				},
			},
		},
		// Pagination tests
		{"temperatureMeasurements(first: 1)", map[string]interface{}{"first": 1},
			tmResponse{
				temperatureMeasurementsConnection{
					Typename: "TemperatureMeasurementConnection",
					PageInfo: pageInfo{HasNextPage: true, HasPreviousPage: false},
					Edges: []temperatureMeasurementEdge{
						temperatureMeasurementEdge{Typename: "TemperatureMeasurementEdge", Cursor: encodedOffset1,
							Node: tm{Typename: "TemperatureMeasurement", Id: m1.Id}},
					},
				},
			},
		},
		{"temperatureMeasurements(after: <encodedOffset1>)", map[string]interface{}{"after": encodedOffset1},
			tmResponse{
				temperatureMeasurementsConnection{
					Typename: "TemperatureMeasurementConnection",
					PageInfo: pageInfo{false, false},
					Edges: []temperatureMeasurementEdge{
						temperatureMeasurementEdge{Typename: "TemperatureMeasurementEdge", Cursor: encodedOffset2,
							Node: tm{Typename: "TemperatureMeasurement", Id: m2.Id}},
					},
				},
			},
		},
		// // todo: add a batch_uuid test validating if the measurement is AFTER the disassociation
	}

	var worrywortSchema = graphql.MustParseSchema(graphql_api.Schema, graphql_api.NewResolver(db))
	for _, tm := range testmatrix {
		t.Run(tm.name, func(t *testing.T) {
			query := strings.Trim(`
					query getTemperatureMeasurements($first: Int $after: String $batchId: ID $sensorId: ID) {
						temperatureMeasurements(first: $first after: $after batchId: $batchId sensorId: $sensorId) {
							__typename pageInfo {hasPreviousPage hasNextPage}
							edges {
								__typename cursor node { __typename id }
							}
						}
					}`, " ")
			operationName := ""
			ctx := context.Background()
			ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
			ctx = context.WithValue(ctx, "db", db)
			resultData := worrywortSchema.Exec(ctx, query, operationName, tm.variables)

			var result tmResponse
			if err = json.Unmarshal(resultData.Data, &result); err != nil {
				t.Fatalf("Error: %s for result %v", err, result)
			}

			if err != nil {
				t.Fatalf("%v: %v", result, resultData)
			}
			if !cmp.Equal(tm.expected, result) {
				t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(tm.expected, result))
			}
		})
	}
}

func TestCreateSensorMutation(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	var worrywortSchema = graphql.MustParseSchema(graphql_api.Schema, graphql_api.NewResolver(db))
	// Some structs so that the json can be unmarshalled.
	type payload struct {
		Typename string `json:"__typename"`
		Sensor   node   `json:"sensor"`
	}

	type createSensor struct {
		CreateSensor *payload `json:"createSensor"`
	}

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"name": "My Sensor",
		},
	}
	query := `
		mutation addSensor($input: CreateSensorInput!) {
			createSensor(input: $input) {
				__typename
				sensor {
					__typename
					id
				}
			}
		}`

	operationName := ""
	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)

	t.Run("Test sensor is created with valid data", func(t *testing.T) {
		ctx := context.WithValue(ctx, authMiddleware.DefaultUserKey, &u)
		resultData := worrywortSchema.Exec(ctx, query, operationName, variables)
		var result createSensor
		err = json.Unmarshal(resultData.Data, &result)
		if err != nil {
			t.Fatalf("%v: %v", result, resultData)
		}

		// Test the returned graphql types
		if result.CreateSensor.Typename != "CreateSensorPayload" {
			t.Errorf("createBatch returned unexpected type: %s", result.CreateSensor.Typename)
		}

		if result.CreateSensor.Sensor.Typename != "Sensor" {
			t.Errorf("createSensor returned unexpected type for Sensor: %s", result.CreateSensor.Sensor.Typename)
		}

		// Look up the object in the db to be sure it was created
		var sensorId string = result.CreateSensor.Sensor.Id
		sensor, err := worrywort.FindSensor(map[string]interface{}{"user_id": *u.Id, "uuid": sensorId}, db)

		if err == sql.ErrNoRows {
			t.Error("Sensor was not saved to the database. Query returned no results.")
		} else if err != nil {
			t.Errorf("Error: %v and Sensor: %v", err, sensor)
		}
		// TODO: Really should maybe make sure all properties were inserted
	})

	t.Run("Unauthenticated", func(t *testing.T) {
		ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, nil)
		result := worrywortSchema.Exec(ctx, query, operationName, variables)

		// TODO: Make this error checking reusable
		e := graphqlErrors.QueryError{Message: "User must be authenticated"}
		expectedErrors := []*graphqlErrors.QueryError{&e}
		cmpOpts := []cmp.Option{
			cmpopts.IgnoreFields(e, "ResolverError", "Path", "Extensions", "Rule", "Locations"),
		}
		if !cmp.Equal(expectedErrors, result.Errors, cmpOpts...) {
			t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(expectedErrors, result.Errors, cmpOpts...))
		}
		// End error checking
		var expected interface{}
		err = json.Unmarshal([]byte(`{"createSensor": null}`), &expected)
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
		}

	})
}
