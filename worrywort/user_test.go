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
	usererType := reflect.TypeOf((*Userer)(nil)).Elem()
	if !reflect.TypeOf(user{}).Implements(usererType) {
		t.Error("worrywort.user does not implement Interface worrywort.Userer.")
	}
}
