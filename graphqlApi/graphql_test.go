package graphqlApi_test

import (
	"context"
	"database/sql"
	"fmt"
	txdb "github.com/DATA-DOG/go-txdb"
	"github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/graphqlApi"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	"os"
	"regexp"
	"strconv"
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

		if newToken.User().ID() != user.ID() {
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

	createdAt := time.Now().Round(time.Microsecond)
	updatedAt := time.Now().Round(time.Microsecond)
	brewedDate := time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute).Round(time.Microsecond)

	b := worrywort.NewBatch(0, "Testing", brewedDate, bottledDate, 5, 4.5, worrywort.GALLON, 1.060, 1.020, u, createdAt, updatedAt,
		"Brew notes", "Taste notes", "http://example.org/beer")
	b, err = worrywort.SaveBatch(db, b)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	b2 := worrywort.NewBatch(0, "Testing 2", time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond),
													 time.Now().Add(time.Duration(5) * time.Minute).Round(time.Microsecond), 5, 4.5,
													 worrywort.GALLON, 1.060, 1.020, u, createdAt, updatedAt, "Brew notes", "Taste notes",
													 "http://example.org/beer")
	b2, err = worrywort.SaveBatch(db, b2)

	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	u2batch := worrywort.NewBatch(0, "Testing 2", time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond),
													 time.Now().Add(time.Duration(5) * time.Minute).Round(time.Microsecond), 5, 4.5,
													 worrywort.GALLON, 1.060, 1.020, u2, createdAt, updatedAt, "Brew notes", "Taste notes",
													 "http://example.org/beer")
	u2batch, err = worrywort.SaveBatch(db, u2batch)

	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	t.Run("Test query for batch(id: ID!) which exists returns the batch", func(t *testing.T) {
		variables := map[string]interface{}{
			"id": strconv.Itoa(b.ID()),
		}
		query := `
			query getBatch($id: ID!) {
				batch(id: $id) {
					id
					createdAt
					brewNotes
					brewedDate
					bottledDate
					volumeBoiled
					volumeInFermenter
					volumeUnits
					tastingNotes
					finalGravity
					originalGravity
					recipeURL
					createdBy {
						id
						email
						firstName
						lastName
					}
				}
			}
		`
		operationName := ""
		ctx := context.Background()
		ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)
		result := worrywortSchema.Exec(ctx, query, operationName, variables)

		// This is the dumbest date formatting I have ever seen
		expected := fmt.Sprintf(
			`{"batch":{"id":"%d","createdAt":"%s","brewNotes":"Brew notes","brewedDate":"%s","bottledDate":"%s","volumeBoiled":5,"volumeInFermenter":4.5,"volumeUnits":"<worrywort.VolumeUnitType Value>","tastingNotes":"Taste notes","finalGravity":1.02,"originalGravity":1.06,"recipeURL":"http://example.org/beer","createdBy":{"id":"%d","email":"user@example.com","firstName":"Justin","lastName":"Michalicek"}}}`,
			b.ID(), createdAt.Format("2006-01-02T15:04:05Z"), brewedDate.Format("2006-01-02T15:04:05Z"), bottledDate.Format("2006-01-02T15:04:05Z"), u.ID())

		if expected != string(result.Data) {
			t.Errorf("Expected: %s\nGot: %s", expected, result.Data)
		}
	})

	t.Run("Test query for batch(id: ID!) which does not exist returns null", func(t *testing.T) {
		variables := map[string]interface{}{
			"id": "fake",
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
		ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)
		result := worrywortSchema.Exec(ctx, query, operationName, nil)

		// Could make the dicts and compare json, then spacing would not matter like it does here.
		expected := fmt.Sprintf(`{"batches":[{"__typename":"Batch","id":"%d"},{"__typename":"Batch","id":"%d"}]}`, b.ID(), b2.ID())
		if expected != string(result.Data) {
			t.Errorf("Expected: %s\nGot: %s", expected, result.Data)
		}
	})
}
