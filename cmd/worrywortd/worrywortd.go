package main

import (
	"context"
	"fmt"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/graphqlApi"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmichalicek/worrywort-server-go/restapi"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"github.com/go-chi/chi"
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

func AddContext(ctx context.Context, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

func mainOld() {

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

	tokenAuthHandler := authMiddleware.NewTokenAuthHandler(newTokenAuthLookup(db))
	// authRequiredHandler := authMiddleware.NewLoginRequiredHandler()

	// Does this need a Schema pointer?
	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	// can add logging similarly

	// TODO: need to manually handle CORS?
	// https://github.com/graph-gophers/graphql-go/issues/74#issuecomment-289098639
	// http.Handle("/graphql", AddContext(ctx, tokenAuthHandler(authRequiredHandler(&relay.Handler{Schema: schema}))))
	http.Handle("/graphql", AddContext(ctx, tokenAuthHandler(&relay.Handler{Schema: schema})))
	// may pull in gorilla mux or something lighter like chi if this gets more complex
	// http.HandleFunc("/api/v1//measurement", AddContext(ctx, tokenAuthHandler(authRequiredHandler(  restapi.insertMeasurement()  ))))
	// http.Handle("/api/v1//measurement", AddContext(ctx, tokenAuthHandler(  restapi.InsertMeasurement  )))
	uri, uriSet := os.LookupEnv("WORRYWORTD_HOST")
	if !uriSet {
		uri = ":8080"
	}
	log.Printf("WorryWort now listening on %s\n", uri)
	log.Fatal(http.ListenAndServe(uri, nil))
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

	tokenAuthHandler := authMiddleware.NewTokenAuthHandler(newTokenAuthLookup(db))
	authRequiredHandler := authMiddleware.NewLoginRequiredHandler()

	// Does this need a Schema pointer?
	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	// can add logging similarly


	r := chi.NewRouter()
	r.Handle("/graphql", AddContext(ctx, tokenAuthHandler(&relay.Handler{Schema: schema})))
	// want to use r.Post() but I am not quite smart enough
	r.Handle("/api/v1/measurement",
		AddContext(ctx,
			tokenAuthHandler( authRequiredHandler( restapi.MeasurementHandler{} )),
		),
	)

	// TODO: need to manually handle CORS?
	// https://github.com/graph-gophers/graphql-go/issues/74#issuecomment-289098639
	uri, uriSet := os.LookupEnv("WORRYWORTD_HOST")
	if !uriSet {
		uri = ":8080"
	}
	log.Printf("WorryWort now listening on %s\n", uri)
	log.Fatal(http.ListenAndServe(uri, r))
}
