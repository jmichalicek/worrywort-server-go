// Common functions for the tests which should only exist once live here
package worrywort

import (
	"database/sql"
	"fmt"
	txdb "github.com/DATA-DOG/go-txdb"
	"github.com/jmoiron/sqlx"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	dbUser, _ := os.LookupEnv("DATABASE_USER")
	dbPassword, _ := os.LookupEnv("DATABASE_PASSWORD")
	dbHost, _ := os.LookupEnv("DATABASE_HOST")
	// we register an sql driver txdb
	connString := fmt.Sprintf("host=%s port=5432 user=%s dbname=worrywort_test sslmode=disable", dbHost,
		dbUser)
	if dbPassword != "" {
		connString += fmt.Sprintf(" password=%s", dbPassword)
	}
	txdb.Register("txdb", "postgres", connString)
	retCode := m.Run()
	os.Exit(retCode)
}

func setUpTestDb() (*sqlx.DB, error) {
	_db, err := sql.Open("txdb", "one")
	if err != nil {
		return nil, err
	}

	db := sqlx.NewDb(_db, "postgres")
	if err != nil {
		return nil, err
	}

	return db, nil
}
