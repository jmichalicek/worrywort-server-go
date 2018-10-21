package main

import (
	"context"
	"fmt"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/graphqlApi"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
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

func AddContext(ctx context.Context, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(ctx))
	})
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
	// based on https://github.com/OscarYuen/go-graphql-starter/blob/f8ff416af2213ef93ef5f459904d6a403ab25843/server.go
	// can I just addContext to relay.hanlder or do I need a custom handler and then I can attach db there
	// try it.  My understanding is that this can be frowned upon... but it seems nicer than
	// having to pass the db around through EVERY resolver I create/use
	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	// can add logging similarly

	http.Handle("/graphql", AddContext(ctx, tokenAuthHandler(&relay.Handler{Schema: schema})))
	uri, uriSet := os.LookupEnv("WORRYWORTD_HOST")
	if !uriSet {
		uri = ":8080"
	}
	log.Printf("WorryWort now listening on %s\n", uri)
	log.Fatal(http.ListenAndServe(uri, nil))
}
