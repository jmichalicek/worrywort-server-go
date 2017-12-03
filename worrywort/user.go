package worrywort

import "time"
// Models and functions for user management


type User struct {
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
func NewUser(id int64, email, firstName, lastName string, createdAt, updatedAt time.Time) User {
	return User{id: id, email: email, firstName: firstName, lastName: lastName, createdAt: createdAt,
		updatedAt: updatedAt}
}

func (u User) Id() int64 { return u.id }
func (u User) FirstName() string    { return u.firstName }
func (u User) LastName() string     { return u.lastName }
func (u User) Email() string        { return u.email }
func (u User) CreatedAt() time.Time { return u.createdAt }
func (u User) UpdatedAt() time.Time { return u.updatedAt }
