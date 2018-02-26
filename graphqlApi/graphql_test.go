package graphqlApi_test

import (
	// "context"
	"database/sql"
	"database/sql/driver"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmichalicek/worrywort-server-go/graphqlApi"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"github.com/neelance/graphql-go"
	// "github.com/neelance/graphql-go/gqltesting"
	// "golang.org/x/crypto/bcrypt"
	"context"
	// "fmt"
	"testing"
	"time"
	// "encoding/json"
	"regexp"
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

	// Not sure I like this over just using the build in run with a name,
	// but this will work for now.
	// This also might belong as well under the cmd/worrywortd tests since that is what REALLY needs to work.

	user := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	// using hashedPassword in the returned sql is nto working
	var hashedPassword string = "$2a$13$pPg7mwPA.VFf3W9AUZyMGO0Q2nhoh/979F/TZ8ED.iqVubLe.TDmi"
	// user, err = worrywort.SetUserPassword(user, "password", bcrypt.MinCost)
	// fmt.Print(user.Password())
	// if err != nil {
	// 	t.Errorf("Unexpected error hashing password: %v", err)
	// }

	// tokenId := "tokenid"
	// tokenKey := "secret"
	// token, _ := worrywort.NewToken(tokenId, tokenKey, user, 0, 10)

	t.Run("Test valid email and password returns a token", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "created_at", "updated_at", "password"}).
			AddRow(user.ID(), user.Email(), user.FirstName(), user.LastName(), user.CreatedAt(), user.UpdatedAt(), hashedPassword)
		mock.ExpectQuery(`^SELECT (.+) FROM users WHERE email = \?`).WithArgs(user.Email()).WillReturnRows(rows)

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
		// {"login":{"token":"c9d103e1-8320-45fd-8ac6-245d59c01b3d:$2a$10$1FiMdC5apU32nePwLqoynutvAUhTRP5iRj5VZBoqOEZuXPIFaLeJ."}}
		// the actual hash part of the bcrypt hash is 53 characters made up of uppercase and lowercase US alphabet,
		// 0-9, and then / (forward slash), and . (period)
		// the $2a$10$ indicates bcrypt and the version of it as 2a
		// and a hashcost of 10
		// Testing this pattern this far may be a bit overtesting.  Could just test for any string as the token.
		expected := `\{"login":\{"token":".+:\$2a\$10\$[A-Za-z0-9/.]{53}"\}\}`
		// TODO: capture the token and make sure there's an entry in the db for it.
		matched, err := regexp.Match(expected, result.Data)
		if err != nil {
			t.Errorf(err.Error())
		}
		if !matched {
			t.Errorf("\nExpected respose to match pattern: %s\nGot: %s", expected, result.Data)
		}
	})
}
