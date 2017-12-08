package worrywort

import (
	"testing"
	"time"
)

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
		t.Errorf("Expected:\n\n%v\n\nGot:\n\n%v", expectedBatch, b)
	}
}

func TestNewFermenter(t *testing.T) {

	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	expected := Fermenter{fermenter{ID: 1, Name: "Ferm", Description: "A Fermenter", Volume: 5.0, VolumeUnits: GALLON,
		FermenterType: BUCKET, IsActive: true, IsAvailable: true, CreatedBy: u, CreatedAt: time.Now(),
		UpdatedAt: time.Now()}}

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
