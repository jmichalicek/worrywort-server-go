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
	"github.com/neelance/graphql-go/gqltesting"
	// "golang.org/x/crypto/bcrypt"
	"testing"
	"time"
	// "fmt"
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

	rows := sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "created_at", "updated_at", "password"}).
		AddRow(user.ID(), user.Email(), user.FirstName(), user.LastName(), user.CreatedAt(), user.UpdatedAt(), hashedPassword)
	mock.ExpectQuery(`^SELECT (.+) FROM users WHERE email = \?`).WithArgs(user.Email()).WillReturnRows(rows)

	gqltesting.RunTests(t, []*gqltesting.Test{
		// TODO: mock db query to return expected token!
		{
			Schema: worrywortSchema,
			Query: `
				mutation Login($username: String!, $password: String!) {
					login(username: $username, password: $password) {
						token
					}
				}
			`,
			Variables: map[string]interface{}{
				"username": "user@example.com",
				"password": "password",
			},
			ExpectedResult: `
				{
					"login": {
						"token": "THISISWRONG"
					}
				}
			`,
		},
	})
}
