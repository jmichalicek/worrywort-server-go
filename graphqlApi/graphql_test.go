package graphqlApi_test

import (
	// "context"
	"context"
	"database/sql"
	"database/sql/driver"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/graphqlApi"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	// "fmt"
	"regexp"
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
