package graphql_test

import (
	"context"
	"testing"
	"time"

	"github.com/neelance/graphql-go"
	"github.com/neelance/graphql-go/gqltesting"
)

var worrywortSchema = graphql.MustParseSchema(Schema, &Resolver{})

func TestLoginMutation(t *testing.T) {

	// Not sure I like this over just using the build in run with a name,
	// but this will work for now.
	// This also might belong as well under the cmd/worrywortd tests since that is what REALLY needs to work.

	// gqltesting.RunTests(t, []*gqltesting.Test{
	// 	// TODO: mock db query to return expected token!
	// 	{
	// 		Schema: worrywortSchema,
	// 		Query:`
	// 			mutation Login($username: Username!, $password: Password!) {
	// 				login(username: $username, password: $password) {
	// 					token
	// 				}
	// 			}
	// 		`,
	// 		Variables: map[string][interface{}{
	// 			"username": "user@example.com",
	// 			"password": "password"
	// 		},
	// 		ExpectedResult: `
	// 			{
	// 				"authToken": {
	// 					"token": "THISISWRONG"
	// 				}
	// 			}
	// 		`
	// 	}
	// })
}
