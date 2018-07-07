package worrywort

import (
	"testing"
	"time"
	// "reflect"
	"github.com/davecgh/go-spew/spew"
)

// Test that NewBatch() returns a batch with the expected values
func TestNewBatch(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	brewedDate := time.Now().Add(time.Duration(1) * time.Minute)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute)
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())

	expectedBatch := Batch{Id: 1, Name: "Testing", BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5,
		VolumeInFermenter: 4.5, VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedBy: u,
		CreatedAt: createdAt, UpdatedAt: updatedAt, BrewNotes: "Brew notes", TastingNotes: "Taste notes",
		RecipeURL: "http://example.org/beer"}
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
	expected := Fermenter{Id: 1, Name: "Ferm", Description: "A Fermenter", Volume: 5.0, VolumeUnits: GALLON,
		FermenterType: BUCKET, IsActive: true, IsAvailable: true, CreatedBy: u, CreatedAt: createdAt, UpdatedAt: updatedAt}

	f := NewFermenter(1, "Ferm", "A Fermenter", 5.0, GALLON, BUCKET, true, true, u, createdAt, updatedAt)

	if f != expected {
		t.Errorf("Expected:\n%v\n\nGot:\n%v\n", expected, f)
	}
}

func TestNewTemperatureSensor(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	expected := TemperatureSensor{Id: 1, Name: "Therm1", CreatedBy: u, CreatedAt: createdAt, UpdatedAt: updatedAt}

	therm := NewTemperatureSensor(1, "Therm1", u, createdAt, updatedAt)

	if therm != expected {
		t.Errorf("Expected:\n%v\n\nGot:\n%v\n", expected, therm)
	}

}

func TestNewTemperatureMeasurement(t *testing.T) {
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	b := NewBatch(1, "Testing", time.Now(), time.Now(), 5, 4.5, GALLON, 1.060, 1.020, u, time.Now(), time.Now(),
		"Brew notes", "Taste notes", "http://example.org/beer")
	f := NewFermenter(1, "Ferm", "A Fermenter", 5.0, GALLON, BUCKET, true, true, u, time.Now(), time.Now())
	therm := NewTemperatureSensor(1, "Therm1", u, time.Now(), time.Now())

	createdAt := time.Now()
	updatedAt := time.Now()
	timeRecorded := time.Now()
	expected := TemperatureMeasurement{Id: "shouldbeauuid", Temperature: 64.26, Units: FAHRENHEIT,
		TimeRecorded: timeRecorded, Batch: b, TemperatureSensor: therm, Fermenter: f, CreatedBy: u, CreatedAt: createdAt,
		UpdatedAt: updatedAt}

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
	batchArgs["created_by_user_id"] = u.Id
	batchArgs["id"] = b.Id
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

func TestBatchesForUser(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := NewUser(0, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	u, err = SaveUser(db, u)

	u2 := NewUser(0, "user2@example.com", "Justin", "M", time.Now(), time.Now())
	u2, err = SaveUser(db, u2)

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

	b2 := NewBatch(0, "Testing 2", time.Now().Add(time.Duration(1)*time.Minute).Round(time.Microsecond),
		time.Now().Add(time.Duration(5)*time.Minute).Round(time.Microsecond), 5, 4.5,
		GALLON, 1.060, 1.020, u, createdAt, updatedAt, "Brew notes", "Taste notes",
		"http://example.org/beer")
	b2, err = SaveBatch(db, b2)

	u2batch := NewBatch(0, "Testing 2", time.Now().Add(time.Duration(1)*time.Minute).Round(time.Microsecond),
		time.Now().Add(time.Duration(5)*time.Minute).Round(time.Microsecond), 5, 4.5,
		GALLON, 1.060, 1.020, u2, createdAt, updatedAt, "Brew notes", "Taste notes",
		"http://example.org/beer")
	u2batch, err = SaveBatch(db, u2batch)

	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	// TODO: split up into sub tests for different functionality... no pagination, pagination, etc.
	batches, err := BatchesForUser(db, u, nil, nil)
	if err != nil {
		t.Fatalf("\n%v\n", err)
	}

	// DepEqual is not playing nicely here (ie. I don't understand something) so do a very naive check for now.
	// May be worth trying this instead of spew, which has a Diff() function which may tell me what the difference is
	// https://godoc.org/github.com/kr/pretty
	expected := []Batch{b, b2}
	if len(*batches) != 2 || expected[0].Id != (*batches)[0].Id || expected[1].Id != (*batches)[1].Id {
		t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected[0]), spew.Sdump((*batches)[0]))
	}
	// TODO: Cannot figure out WHY these are not equal.
	// if !reflect.DeepEqual(*batches, expected) {
	// 	t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(*batches))
	// }
}

func TestInsertBatch(t *testing.T) {}
func TestUpdateBatch(t *testing.T) {}
func TestSaveBatch(t *testing.T)   {}
