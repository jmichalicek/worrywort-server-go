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
var UserNotFoundError error = errors.New("User not found")

type user struct {
	// really could use email as the pk for the db, but fudging it because I've been trained by ORMs
	// TODO: Considering having a separate username from the email
	ID        int64  `db:"id"`
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string `db:"email"`
	Password  string `db:"password"`

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
func (u User) Password() string { return u.user.Password }

// SetUserPassword hashes the given password and returns a new user with the password set to the bcrypt hashed value
// using the given hashCost.  If hashCost is less than bcrypt.MinCost then worrywort.DefaultPasswordHashCost is used.
func SetUserPassword(u User, password string, hashCost int) (User, error) {
	// TODO: abstract this out to allow for easily using a different hashing algorithm
	// or changing the hash cost, such as to something very low for tests?
	if hashCost <= bcrypt.MinCost {
		hashCost = DefaultPasswordHashCost
	}
	passwdBytes, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	if err != nil {
		return u, err
	}
	u.user.Password = string(passwdBytes)
	return u, nil
}

// super incomplete
func (u User) Save(db *sqlx.DB) error {
	var query string
	var createdAt time.Time
	var err error = nil
	if u.ID() != 0 {
		query = "UPDATE users SET email = ?, first_name = ?, last_name = ?, password = ?, updated_at = ? WHERE id = ?)"
		createdAt = u.CreatedAt()
		_, err = db.Exec(query, u.Email(), u.FirstName(), u.LastName(), u.Password(), createdAt, time.Now(), u.ID())
	} else {
		query = "INSERT INTO users (email, first_name, last_name, password, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)"
		createdAt = time.Now()
		_, err = db.Exec(query, u.Email(), u.FirstName(), u.LastName(), u.Password(), createdAt, time.Now())
	}
	return err
}

// TODO: Needs to accept set of changes somehow... I think a map of string[interface{}]
func UpdateUser(user User) {

}

// Looks up the user by id in the database and returns a new User
func LookupUser(id int64, db *sqlx.DB) (User, error) {
	u := User{}
	// if I have understood correctly, different DBs use a different parameterization token (the ? below).
	// By default sqlx just passes whatever you type and you need to manually use the correct token...
	// ? for mysql, $1..$N for postgres, etc.
	// db.Rebind() will update the string to use the correct bind.
	query := db.Rebind("SELECT id, first_name, last_name, email, password, created_at, updated_at, password FROM users WHERE id=?")
	err := db.Get(&u, query, id)
	if err != nil {
		return User{}, err
	}
	// this seems dumb, but it ensures correctness by using the standard NewUser interface
	return u, nil
}

// Looks up the username (or email, as the case is for now) and verifies that the password
// matches that of the user.
func AuthenticateLogin(username, password string, db *sqlx.DB) (User, error) {
	u := User{}

	query := db.Rebind(
		"SELECT id, email, first_name, last_name, created_at, updated_at, password FROM users WHERE email = ?")
	err := db.Get(&u, query, username)

	if u == (User{}) {
		return u, UserNotFoundError //errors.New(UserNotFoundError)
	}

	if err != nil {
		return User{}, err
	}

	pwdErr := bcrypt.CompareHashAndPassword([]byte(u.Password()), []byte(password))
	if pwdErr != nil {
		return User{}, pwdErr
	}

	return u, nil
}

// Uses a token as passed in authentication headers by a user to look them up
func LookupUserByToken(tokenStr string, db *sqlx.DB) (User, error) {
	// TODO: Is there a good way to abstract this so that token data could optionally
	// be stored in redis while other data is in postgres?  If two separate lookups
	// are done even for db then it is easy.

	// TODO: Considering making this taken token id and actual token as separate params
	// for explicitness
	// token is passed in has 2 parts
	tokenParts := strings.SplitN(tokenStr, ":", 2)

	if len(tokenParts) != 2 {
		return User{}, errors.New(TokenFormatError) // should return an error about invalid token probably
	}

	tokenId := tokenParts[0]
	tokenSecret := tokenParts[1]
	token := authToken{}

	query := db.Rebind(
		"SELECT t.token_id, t.token, t.scope, t.expires_at, t.created_at, t.updated_at u.id, u.first_name, u.last_name, " +
			"u.email, u.created_at, u.updated_at, u.password FROM user_authtokens t LEFT JOIN users u ON t.user_id = u.id " +
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

	return token.User, nil
}
