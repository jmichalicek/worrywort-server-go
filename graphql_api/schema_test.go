package graphql_api

import (
	graphql "github.com/graph-gophers/graphql-go"
	"testing"
)

// Test that the schema parses, the same as is done at runtime when starting worrywortd.
// Any issues here would probably also be caught by integration tests on worrywortd ensuring
// http routing, responses, etc.
func TestParseSchema(t *testing.T) {
	graphql.MustParseSchema(Schema, &Resolver{})
}
