package main

import (
	"github.com/jmichalicek/worrywort-server-go/graphqlApi"
	graphql "github.com/neelance/graphql-go"
	"github.com/neelance/graphql-go/relay"
	"log"
	"net/http"
)

var schema *graphql.Schema


func init() {
	schema = graphql.MustParseSchema(graphqlApi.Schema, &graphqlApi.Resolver{})
}

func main() {
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(graphqlApi.Graphiql)
	}))

	// Does this need a Schema pointer?
	// can we do non-relay
	http.Handle("/query", &relay.Handler{Schema: schema})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
