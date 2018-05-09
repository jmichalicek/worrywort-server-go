package worrywort

import (
	"database/sql"
	"fmt"
	txdb "github.com/DATA-DOG/go-txdb"
	"github.com/jmoiron/sqlx"
	"os"
	"testing"
	"time"
)

func init() {
	dbUser, _ := os.LookupEnv("DATABASE_USER")
	dbPassword, _ := os.LookupEnv("DATABASE_PASSWORD")
	// we register an sql driver txdb
	connString := fmt.Sprintf("host=database port=5432 user=%s password=%s dbname=worrywort_test sslmode=disable", dbUser, dbPassword)
	txdb.Register("txdb", "postgres", connString)
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

// Test that NewBatch() returns a batch with the expected values
func TestNewBatch(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	brewedDate := time.Now().Add(time.Duration(1) * time.Minute)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute)
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())

	expectedBatch := Batch{batch: batch{ID: 1, Name: "Testing", BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5,
		VolumeInFermenter: 4.5, VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedBy: u,
		CreatedAt: createdAt, UpdatedAt: updatedAt, BrewNotes: "Brew notes", TastingNotes: "Taste notes",
		RecipeURL: "http://example.org/beer"}}
	b := NewBatch(1, "Testing", brewedDate, bottledDate, 5, 4.5, GALLON, 1.060, 1.020, u, createdAt, updatedAt,
		"Brew notes", "Taste notes", "http://example.org/beer")

	if b != expectedBatch {
		t.Errorf("Expected: %v\n\nGot: %v", expectedBatch, b)
	}
}

func TestNewFermenter(t *testing.T) {

	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	expected := Fermenter{fermenter{ID: 1, Name: "Ferm", Description: "A Fermenter", Volume: 5.0, VolumeUnits: GALLON,
		FermenterType: BUCKET, IsActive: true, IsAvailable: true, CreatedBy: u, CreatedAt: createdAt, UpdatedAt: updatedAt}}

	f := NewFermenter(1, "Ferm", "A Fermenter", 5.0, GALLON, BUCKET, true, true, u, createdAt, updatedAt)

	if f != expected {
		t.Errorf("Expected:\n%v\n\nGot:\n%v\n", expected, f)
	}
}

func TestNewThermometer(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	expected := Thermometer{thermometer: thermometer{ID: 1, Name: "Therm1", CreatedBy: u, CreatedAt: createdAt, UpdatedAt: updatedAt}}

	therm := NewThermometer(1, "Therm1", u, createdAt, updatedAt)

	if therm != expected {
		t.Errorf("Expected:\n%v\n\nGot:\n%v\n", expected, therm)
	}

}

func TestNewTemperatureMeasurement(t *testing.T) {
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	b := NewBatch(1, "Testing", time.Now(), time.Now(), 5, 4.5, GALLON, 1.060, 1.020, u, time.Now(), time.Now(),
		"Brew notes", "Taste notes", "http://example.org/beer")
	f := NewFermenter(1, "Ferm", "A Fermenter", 5.0, GALLON, BUCKET, true, true, u, time.Now(), time.Now())
	therm := NewThermometer(1, "Therm1", u, time.Now(), time.Now())

	createdAt := time.Now()
	updatedAt := time.Now()
	timeRecorded := time.Now()
	expected := TemperatureMeasurement{temperatureMeasurement{ID: "shouldbeauuid", Temperature: 64.26, Units: FAHRENHEIT,
		TimeRecorded: timeRecorded, Batch: b, Thermometer: therm, Fermenter: f, CreatedBy: u, CreatedAt: createdAt,
		UpdatedAt: updatedAt}}

	m := NewTemperatureMeasurement(
		"shouldbeauuid", 64.26, FAHRENHEIT, b, therm, f, timeRecorded, createdAt, updatedAt, u)

	if m != expected {
		t.Errorf("Expected:\n%v\n\nGot:\n%v\n", expected, m)
	}
}

func TestFindBatch(t *testing.T) {
	// Set up the db using sql.Open() and sqlx.NewDb() rather than sqlx.Open() so that the custom
	// `txdb` db type may be used with Open() but can still be registered as postgres with sqlx
	// so that sqlx' Rebind() functions.

	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := NewUser(0, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = SaveUser(db, u)

	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	createdAt := time.Now().Round(time.Microsecond)
	updatedAt := time.Now().Round(time.Microsecond)
	// THe values when returned by postgres will be microsecond accuracy, but golang default
	// is nanosecond, so we round these for easy comparison
	brewedDate := time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute).Round(time.Microsecond)
	b := NewBatch(0, "Testing", brewedDate, bottledDate, 5, 4.5, GALLON, 1.060, 1.020, u, createdAt, updatedAt,
		"Brew notes", "Taste notes", "http://example.org/beer")
	b, err = SaveBatch(db, b)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	batchArgs := make(map[string]interface{})
	batchArgs["created_by_user_id"] = u.ID()
	batchArgs["id"] = b.ID()
	found, err := FindBatch(batchArgs, db)
	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	} else if !b.StrictEqual(*found) {
		t.Errorf("Expected: %v\nGot: %v\n", b, *found)
	}

	// var count int = 0
	// err = db.QueryRow("SELECT COUNT(id) FROM users").Scan(&count)
	// if err != nil {
	// 	t.Fatalf("failed to count users: %s", err)
	// }
	// if count != 3 {
	// 	t.Fatalf("expected 3 users to be in database, but got %d", count)
	// }
}

func TestInsertBatch(t *testing.T) {}
func TestUpdateBatch(t *testing.T) {}
func TestSaveBatch(t *testing.T)   {}
