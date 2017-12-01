package worrywort

import (
	"reflect"
	"testing"
	"time"
)

func TestNewUser(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", createdAt, updatedAt)

	expectedUser := user{id: 1, email: "user@example.com", firstName: "Justin", lastName: "Michalicek",
		createdAt: createdAt, updatedAt: updatedAt}
	if u != expectedUser {
		t.Errorf("Expected:\n\n%v\n\nGot:\n\n%v", expectedUser, u)
	}
}

func TestUserImplementsUserer(t *testing.T) {
	// TODO: What exactly does this line to get the type of the interface do?
	// for now I just copied examples and played around to verify that what makes sense to me does not work
	// example at https://stackoverflow.com/a/34698753/482999
	// If I understand https://stackoverflow.com/a/37468354/482999 correctly
	// This is type casting the value nil to be a pointer to the Userer interface,
	// getting the Type of that and from there you use Elem() to get the type of the thing
	// being referenced.
	usererType := reflect.TypeOf((*Userer)(nil)).Elem()
	if !reflect.TypeOf(user{}).Implements(usererType) {
		t.Error("worrywort.user does not implement Interface worrywort.Userer.")
	}
}
