package graphqlApi_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	txdb "github.com/DATA-DOG/go-txdb"
	"github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/graphqlApi"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"
)

// from https://github.com/DATA-DOG/go-sqlmock#matching-arguments-like-timetime
// stuff for matching times.  Need to improve this to match now-ish times
type AnyTime struct{}

// Match satisfies sqlmock.Argument interface
func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

// TODO: remove setUpDb() and use setUpTestDB() with a real txdb for the login test
func setUpDb() (*sqlx.DB, *sql.DB, sqlmock.Sqlmock, error) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		return nil, mockDB, mock, err
		// t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	// TODO: This may need to be a pointer
	return sqlxDB, mockDB, mock, nil
}

func TestMain(m *testing.M) {
	dbUser, _ := os.LookupEnv("DATABASE_USER")
	dbPassword, _ := os.LookupEnv("DATABASE_PASSWORD")
	dbHost, _ := os.LookupEnv("DATABASE_HOST")
	// we register an sql driver txdb
	connString := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=worrywort_test sslmode=disable", dbHost,
		dbUser, dbPassword)
	txdb.Register("txdb", "postgres", connString)
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

	sqlxDB, mockDB, mock, err := setUpDb()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	// should this close sqlxdb instead?
	defer mockDB.Close()
	var worrywortSchema = graphql.MustParseSchema(graphqlApi.Schema, graphqlApi.NewResolver(sqlxDB))
	user := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	var hashedPassword string = "$2a$13$pPg7mwPA.VFf3W9AUZyMGO0Q2nhoh/979F/TZ8ED.iqVubLe.TDmi"

	t.Run("Test valid email and password returns a token", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "created_at", "updated_at", "password"}).
			AddRow(user.ID(), user.Email(), user.FirstName(), user.LastName(), user.CreatedAt(), user.UpdatedAt(), hashedPassword)
		mock.ExpectQuery(`^SELECT (.+) FROM users WHERE email = \?`).WithArgs(user.Email()).WillReturnRows(rows)

		insertResult := sqlmock.NewResult(1, 1)
		mock.ExpectExec(`^INSERT INTO user_authtokens \(token_id, token, expires_at, updated_at, scope, user_id\) VALUES \(\?, \?, \?, \?, \?, \?\)`).
			WillReturnResult(insertResult) // should test args before WillReturnResult
		// WithArgs(tokenid, token, nil, AnyTime{}, worrywort.TOKEN_SCOPE_ALL, user.ID()).
		// do not know what tokenid and token are to test
		// but could maybe add an AnyString{}

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

		// TODO: start using data-dog sql-txdb so that sql queries are actually tested.
		// that will also allow me to grab the token id as below and verify that it actually exists, matching what
		// was returned.  Currently I can just verify that there was an insert.
		// subMatches := matcher.FindStringSubmatch(string(result.Data))
		// tokenId := subMatches[1]
		// tokenStr := subMatches[2]
		// "INSERT INTO user_authtokens (id, token, expires_at, updated_at, scope, user_id) VALUES (?, ?, ?, ?, ?, ?, ?)"
		// var lastInsertID, affected int
		// insertResult := sqlmock.NewResult(lastInsertID, affected)

	})
}

func TestBatchQuery(t *testing.T) {
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

	t.Run("Test query for batch which exists returns the batch", func(t *testing.T) {
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
		const DefaultUserKey string = "user"
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

	t.Run("Test query for batch which does not exist returns null", func(t *testing.T) {
		variables := map[string]interface{}{
			"id": "fake",
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
		const DefaultUserKey string = "user"
		ctx = context.WithValue(ctx, authMiddleware.DefaultUserKey, u)
		result := worrywortSchema.Exec(ctx, query, operationName, variables)

		expected := `{"batch":null}`
		if expected != string(result.Data) {
			t.Errorf("Expected: %s\nGot: %s", expected, result.Data)
		}
	})

}
