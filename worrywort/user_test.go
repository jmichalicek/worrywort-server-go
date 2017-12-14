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

func TestUserStruct(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now().Add(time.Hour * time.Duration(1))
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", createdAt, updatedAt)

	t.Run("ID()", func(t *testing.T) {
		var actual int64 = u.ID()
		expected := u.user.ID
		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})

	t.Run("LastName()", func(t *testing.T) {
		var actual string = u.LastName()
		expected := u.user.LastName
		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})

	t.Run("FirstName()", func(t *testing.T) {
		var actual string = u.FirstName()
		expected := u.user.FirstName
		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})

	t.Run("Email()", func(t *testing.T) {
		var actual string = u.Email()
		expected := u.user.Email
		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var actual time.Time = u.CreatedAt()
		expected := u.user.CreatedAt
		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var actual time.Time = u.UpdatedAt()
		expected := u.user.UpdatedAt
		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})
}
