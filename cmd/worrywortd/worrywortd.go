package main

import (
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/graphqlApi"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	graphql "github.com/neelance/graphql-go"
	"github.com/neelance/graphql-go/relay"
	"log"
	"net/http"
	"time"
	"os"
)

var schema *graphql.Schema

func init() {
	schema = graphql.MustParseSchema(graphqlApi.Schema, &graphqlApi.Resolver{})
}

// TODO: Make this REALLY look up user in db after db layer is added
// Looks up the user based on a token
func tokenAuthUserLookup(token string) worrywort.User {
	return worrywort.NewUser(1, "jmichalicek@gmail.com", "Justin", "Michalicek", time.Now(), time.Now())
}

func main() {
	tokenAuthHandler := authMiddleware.NewTokenAuthHandler(tokenAuthUserLookup)
	http.Handle("/", tokenAuthHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(graphqlApi.Graphiql)
	})))

	// Does this need a Schema pointer?
	// can we do non-relay
	http.Handle("/graphql", tokenAuthHandler(&relay.Handler{Schema: schema}))
	uri, uriSet := os.LookupEnv("WORRYWORTD_HOST")
	if !uriSet {
		uri = ":8080"
	}
	log.Fatal(http.ListenAndServe(uri, nil))
}
