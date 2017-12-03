package worrywort

import (
	"reflect"
	"testing"
	"time"
	"net/url"
)

// Test that NewBatch() returns a batch with the expected values
func TestNewBatch(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	brewedDate := time.Now().Add(time.Duration(1) * time.Minute)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute)
	recipeURL, _ := url.Parse("http://example.org/beer")
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())

	expectedBatch := Batch{id: 1, name: "Testing", brewedDate: brewedDate, bottledDate: bottledDate, volumeBoiled: 5,
		volumeInFermenter: 4.5, volumeUnits: GALLON, originalGravity: 1.060, finalGravity: 1.020, createdBy: u, createdAt: createdAt, updatedAt: updatedAt,
		brewNotes: "Brew notes", tastingNotes: "Taste notes", recipeURL: *recipeURL}
	b := NewBatch(1, "Testing", brewedDate, bottledDate, 5, 4.5, GALLON, 1.060, 1.020,u, createdAt, updatedAt,
		"Brew notes", "Taste notes", *recipeURL)

	if b != expectedBatch {
		t.Errorf("Expected:\n\n%v\n\nGot:\n\n%v", expectedBatch, b)
	}
}
