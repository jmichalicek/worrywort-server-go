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

	expectedBatch := Batch{id: 1, name: "Testing", brewedDate: brewedDate, bottledDate: bottledDate, volumeBoiled: 5,
		volumeInFermenter: 4.5, volumeUnits: GALLON, originalGravity: 1.060, finalGravity: 1.020, createdBy: u, createdAt: createdAt, updatedAt: updatedAt,
		brewNotes: "Brew notes", tastingNotes: "Taste notes", recipeURL: "http://example.org/beer"}
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
	expected := Fermenter{id: 1, name: "Ferm", description: "A Fermenter", volume: 5.0, volumeUnits: GALLON,
		fermenterType: BUCKET, isActive: true, isAvailable: true, createdBy: u, createdAt: time.Now(), updatedAt: time.Now()}

	f := NewFermenter(1, "Ferm", "A Fermenter", 5.0, GALLON, BUCKET, true, true, u, createdAt, updatedAt)

	if f != expected {
		t.Errorf("Expected:\n%v\n\nGot:\n%v\n", expected, f)
	}
}

func TestNewThermometer(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	expected := Thermometer{id: 1, name: "Therm1", createdBy: u, createdAt: createdAt, updatedAt: updatedAt}
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
	expected := TemperatureMeasurement{id: "shouldbeauuid", temperature: 64.26, units: FAHRENHEIT,
		timeRecorded: timeRecorded, batch: b, thermometer: therm, fermenter: f, createdBy: u, createdAt: createdAt,
		updatedAt: updatedAt}

	m := NewTemperatureMeasurement(
		"shouldbeauuid", 64.26, FAHRENHEIT, b, therm, f, timeRecorded, createdAt, updatedAt, u)

	if m != expected {
		t.Errorf("Expected:\n%v\n\nGot:\n%v\n", expected, m)
	}
}
