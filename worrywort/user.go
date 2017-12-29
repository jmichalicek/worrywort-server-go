package worrywort

import (
	"github.com/jmoiron/sqlx"
	"time"
)

// Models and functions for user management

type user struct {
	// really could use email as the pk for the db, but fudging it because I've been trained by ORMs
	ID        int64  `db:"id"`
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string `db:"email"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type User struct {
	user
}

// Should this return user or Userer?
func NewUser(id int64, email, firstName, lastName string, createdAt, updatedAt time.Time) User {
	return User{user{ID: id, Email: email, FirstName: firstName, LastName: lastName, CreatedAt: createdAt,
		UpdatedAt: updatedAt}}
}

func (u User) ID() int64            { return u.user.ID }
func (u User) FirstName() string    { return u.user.FirstName }
func (u User) LastName() string     { return u.user.LastName }
func (u User) Email() string        { return u.user.Email }
func (u User) CreatedAt() time.Time { return u.user.CreatedAt }
func (u User) UpdatedAt() time.Time { return u.user.UpdatedAt }

// Looks up the user by id in the database and returns a new User
func LookupUser(id int64, db *sqlx.DB) (User, error) {
	// TODO: Test cases for LookupNewUser
	u := user{}
	err := db.Get(&u, "SELECT id, first_name, last_name, email, created_at, updated_at FROM users WHERE id=$1", id)
	if err == nil {
		return User{}, err
	}
	// this seems dumb, but it ensures correctness by using the standard NewUser interface
	return NewUser(u.ID, u.FirstName, u.LastName, u.Email, u.CreatedAt, u.UpdatedAt), nil
}
