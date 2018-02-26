package main

import (
	"fmt"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/graphqlApi"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	graphql "github.com/neelance/graphql-go"
	"github.com/neelance/graphql-go/relay"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var schema *graphql.Schema

// func init() {
// 	schema = graphql.MustParseSchema(graphqlApi.Schema, NewResolver(nil))
// }

// TODO: Make this REALLY look up user in db after db layer is added
// Looks up the user based on a token
// func tokenAuthUserLookup(token string) (worrywort.User, error) {
// 	return worrywort.NewUser(1, "jmichalicek@gmail.com", "Justin", "Michalicek", time.Now(), time.Now()), nil
// }


// Returns a function for looking up a user by token for authMiddleware.NewTokenAuthHandler()
// which closes over the db needed to look up the user
func newTokenAuthLookup(db *sqlx.DB) func(token string) (worrywort.User, error) {
	return func(token string) (worrywort.User, error) {
		// use token to get the user.
		// return worrywort.NewUser(1, "jmichalicek@gmail.com", "Justin", "Michalicek", time.Now(), time.Now())
		return worrywort.LookupUserByToken(token, db)
	}
}

func main() {

	// For now, force postgres
	// TODO: write something to parse db uri?
	// Using LookupEnv because I will probably add some sane defaults... such as localhost for
	dbName, _ := os.LookupEnv("DATABASE_NAME")
	dbUser, _ := os.LookupEnv("DATABASE_USER")
	dbPassword, _ := os.LookupEnv("DATABASE_PASSWORD")
	dbHost, _ := os.LookupEnv("DATABASE_HOST")
	dbPort, dbPortSet := os.LookupEnv("DATABASE_PORT")
	if !dbPortSet {
		dbPort = "5432" // again, assume postgres
	}
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, _ := sqlx.Connect("postgres", connectionString)
	schema = graphql.MustParseSchema(graphqlApi.Schema, graphqlApi.NewResolver(db))

	tokenAuthHandler := authMiddleware.NewTokenAuthHandler(newTokenAuthLookup(db))
	// Does this need a Schema pointer?
	// can we do non-relay
	http.Handle("/graphql", tokenAuthHandler(&relay.Handler{Schema: schema}))
	uri, uriSet := os.LookupEnv("WORRYWORTD_HOST")
	if !uriSet {
		uri = ":8080"
	}
	log.Printf("WorryWort now listening on %s\n", uri)
	log.Fatal(http.ListenAndServe(uri, nil))
}
