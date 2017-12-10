package graphqlApi

import (
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	graphql "github.com/neelance/graphql-go"
	"testing"
	"time"
)

// Test that the schema parses, the same as is done at runtime when starting worrywortd.
// Any issues here would probably also be caught by integration tests on worrywortd ensuring
// http routing, responses, etc.
func TestUserResolver(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", createdAt, updatedAt)
	r := userResolver{u: u}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = r.ID()
		expected := graphql.ID("1")
		if ID != expected {
			t.Errorf("Expected: %v but Got: %v", expected, ID)
		}
	})

	t.Run("FirstName()", func(t *testing.T) {
		var firstName string = r.FirstName()
		expected := "Justin"
		if firstName != expected {
			t.Errorf("Expected: %v but got: %v", expected, firstName)
		}
	})

	t.Run("LastName()", func(t *testing.T) {
		var lastName string = r.LastName()
		expected := "Michalicek"
		if lastName != expected {
			t.Errorf("Expected: %v but got: %v", expected, lastName)
		}
	})

	t.Run("Email()", func(t *testing.T) {
		var email string = r.Email()
		expected := "user@example.com"
		if email != expected {
			t.Errorf("Expected: %v but got: %v", expected, email)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var dt string = r.CreatedAt()
		expected := u.CreatedAt().Format(time.RFC3339)
		if dt != expected {
			t.Errorf("Expected: %v but got: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt string = r.UpdatedAt()
		expected := u.UpdatedAt().Format(time.RFC3339)
		if dt != expected {
			t.Error("Expected: %v but got %v", expected, dt)
		}
	})
}
