package main

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/graphqlApi"
	"github.com/jmichalicek/worrywort-server-go/restapi"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	// "github.com/davecgh/go-spew/spew"
)

var schema *graphql.Schema

// Returns a function for looking up a user by token for authMiddleware.NewTokenAuthHandler()
// which closes over the db needed to look up the user
func newTokenAuthLookup(db *sqlx.DB) func(token string) (*worrywort.User, error) {
	return func(token string) (*worrywort.User, error) {
		// TODO: return the token? That could be more useful in many places than just the user.
		t, err := worrywort.AuthenticateUserByToken(token, db)
		return &t.User, err
	}
}

func main() {
	// For now, force postgres
	// TODO: write something to parse db uri?
	// I suspect this already would and I just didn't read the docs correctly.
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

	// could do a middleware in this style to add db to the context like I used to, but more middleware friendly.
	// Could also do that to add a logger, etc. For now, that stuff is getting attached to each handler
	tokenAuthHandler := authMiddleware.NewTokenAuthHandler(newTokenAuthLookup(db))
	authRequiredHandler := authMiddleware.NewLoginRequiredHandler()

	// Not really sure I needed to switch to Chi here instead of the built in stuff.
	r := chi.NewRouter()
	r.Use(middleware.Compress(5, "text/html", "application/javascript"))
	r.Use(middleware.Logger)
	r.Use(tokenAuthHandler)
	r.Handle("/graphql", &graphqlApi.Handler{Db: db, Handler: &relay.Handler{Schema: schema}})
	r.Method("POST", "/api/v1/measurement", authRequiredHandler(&restapi.MeasurementHandler{Db: db}))
	// TODO: need to manually handle CORS? Chi has some cors stuff, yay
	// https://github.com/graph-gophers/graphql-go/issues/74#issuecomment-289098639
	uri, uriSet := os.LookupEnv("WORRYWORTD_HOST")
	if !uriSet {
		uri = ":8080"
	}
	log.Printf("WorryWort now listening on %s\n", uri)
	log.Fatal(http.ListenAndServe(uri, r))
}
