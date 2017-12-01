package worrywort

import "time"
// Models and functions for user management

// Naming things is hard. It fits idiomatic naming scheme, though.
type Userer interface {
	Id() int64
	FirstName() string
	LastName() string
	Email() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
}


type user struct {
	// really could use email as the pk for the db, but fudging it because I've been trained by ORMs
	id        int64
	firstName string
	lastName  string
	email     string

	// passwordHash?
	createdAt time.Time
	updatedAt time.Time
}

// Should this return user or Userer?
func NewUser(id int64, email, firstName, lastName string, createdAt, updatedAt time.Time) user {
	return user{id: id, email: email, firstName: firstName, lastName: lastName, createdAt: createdAt,
		updatedAt: updatedAt}
}

func (u user) Id() int64 { return u.id }
func (u user) FirstName() string    { return u.firstName }
func (u user) LastName() string     { return u.lastName }
func (u user) Email() string        { return u.email }
func (u user) CreatedAt() time.Time { return u.createdAt }
func (u user) UpdatedAt() time.Time { return u.updatedAt }
