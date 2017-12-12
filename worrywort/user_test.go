package worrywort

import (
	"testing"
	"time"
)

func TestNewUser(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", createdAt, updatedAt)

	expectedUser := User{user{ID: 1, Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek",
		CreatedAt: createdAt, UpdatedAt: updatedAt}}
	if u != expectedUser {
		t.Errorf("Expected:\n\n%v\n\nGot:\n\n%v", expectedUser, u)
	}
}
