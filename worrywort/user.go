package worrywort

import "time"
// Models and functions for user management

type user struct {
	// really could use email as the pk for the db, but fudging it because I've been trained by ORMs
	ID int64
	FirstName string
	LastName string
	Email string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	user
}

// Should this return user or Userer?
func NewUser(id int64, email, firstName, lastName string, createdAt, updatedAt time.Time) User {
	return User{user{ID: id, Email: email, FirstName: firstName, LastName: lastName, CreatedAt: createdAt,
		UpdatedAt: updatedAt}}
}

func (u User) ID() int64 { return u.user.ID }
func (u User) FirstName() string    { return u.user.FirstName }
func (u User) LastName() string     { return u.user.LastName }
func (u User) Email() string        { return u.user.Email }
func (u User) CreatedAt() time.Time { return u.user.CreatedAt }
func (u User) UpdatedAt() time.Time { return u.user.UpdatedAt }
