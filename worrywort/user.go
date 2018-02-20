package worrywort

import (
	"errors"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

// Models and functions for user management

// The bcrypt cost to use for hashing the password
// good info on cost here https://security.stackexchange.com/a/83382
const DefaultPasswordHashCost int = 13

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

// Returns a new user
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
func (u User) SetPassword(password string, db *sqlx.DB) error {
	// Sets the hashed user password in the database

	// TODO: abstract this out to allow for easily using a different hashing algorithm
	// or changing the hash cost, such as to something very low for tests?
	passwdBytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultPasswordHashCost)
	if err != nil {
		return err
	}

	// TODO: Now set the password.  Just save here or set it on the user and have a separate save?
	// I think for now just save here
	setPass := `INSERT INTO users (password) VALUES (?)`
	// db.MustExec(setPass, string(passwdBytes))
	_, err = db.Exec(setPass, string(passwdBytes))
	if err != nil {
		return err
	}
	return nil
}

// Looks up the user by id in the database and returns a new User
func LookupUser(id int64, db *sqlx.DB) (User, error) {
	u := user{}
	// if I have understood correctly, different DBs use a different parameterization token (the ? below).
	// By default sqlx just passes whatever you type and you need to manually use the correct token...
	// ? for mysql, $1..$N for postgres, etc.
	// db.Rebind() will update the string to use the correct bind.
	query := db.Rebind("SELECT id, first_name, last_name, email, created_at, updated_at FROM users WHERE id=?")
	err := db.Get(&u, query, id)
	if err != nil {
		return User{}, err
	}
	// this seems dumb, but it ensures correctness by using the standard NewUser interface
	return NewUser(u.ID, u.Email, u.FirstName, u.LastName, u.CreatedAt, u.UpdatedAt), nil
}

// Uses a token as passed in authentication headers by a user to look them up
func LookupUserByToken(tokenStr string, db *sqlx.DB) (User, error) {
	// TODO: Really need to hash these tokens, possibly with pbkdf2 and a user configurable salt.
	// May switch user to pbkdf2 at same time just to only use 1 hash lib
	// token is passed in as 2 parts
	tokenParts := strings.SplitN(tokenStr, ":", 2)

	if len(tokenParts) != 2 {
		return User{}, errors.New(TokenFormatError) // should return an error about invalid token probably
	}

	tokenId := tokenParts[0]
	tokenSecret := tokenParts[1]
	token := authToken{}

	query := db.Rebind(
		"SELECT t.token_id, t.token, t.scope, t.expires_at, t.created_at, t.updated_at u.id, u.first_name, u.last_name, " +
			"u.email, u.created_at, u.updated_at FROM user_authtokens t LEFT JOIN users u ON t.user_id = u.id " +
			"WHERE t.token_id = ? AND (t.expires_at IS NULL OR t.expires_at < ?)")
	err := db.Get(&token, query, tokenId, time.Now())

	if err != nil {
		return User{}, err
	}

	if token == (authToken{}) {
		return User{}, errors.New(InvalidTokenError)
	}

	pwdErr := bcrypt.CompareHashAndPassword([]byte(token.Token), []byte(tokenSecret))
	if pwdErr != nil {
		return User{}, pwdErr
	}

	// Seems a bit silly since token.User is already a User{} struct, but this ensures that any other init
	// which may be needed actually happens.
	// I wonder if there's a way to make sqlx do that?
	return NewUser(token.User.ID(), token.User.Email(), token.User.FirstName(), token.User.LastName(),
		token.User.CreatedAt(), token.User.UpdatedAt()), nil
}
