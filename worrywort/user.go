package worrywort

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"
)

// Models and functions for user management

// The bcrypt cost to use for hashing the password
// good info on cost here https://security.stackexchange.com/a/83382
const DefaultPasswordHashCost int = 13

var UserNotFoundError error = errors.New("User not found")

type User struct {
	// really could use email as the pk for the db, but fudging it because I've been trained by ORMs
	// TODO: Considering having a separate username from the email
	Id        *int32 `db:"id"`
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string `db:"email"`
	Password  string `db:"password"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (u User) queryColumns() []string {
	// TODO: Way to dynamically build this using the `db` tag and reflection/introspection
	return []string{"id", "first_name", "last_name", "email", "password", "created_at", "updated_at"}
}

// TODO: Remove NewUser()
// Returns a new user
func NewUser(id *int32, email, firstName, lastName string, createdAt, updatedAt time.Time) User {
	return User{Id: id, Email: email, FirstName: firstName, LastName: lastName, CreatedAt: createdAt,
		UpdatedAt: updatedAt}
}

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
	u.Password = string(passwdBytes)
	return u, nil
}

// Save the User to the database.  If User.Id is 0
// then an insert is performed, otherwise an update on the User matching that id.
func SaveUser(db *sqlx.DB, u User) (User, error) {
	// TODO: TEST CASE
	if u.Id == nil || *u.Id == 0 {
		return InsertUser(db, u)
	} else {
		return UpdateUser(db, u)
	}
}

// Inserts the passed in User into the database.
// Returns a new copy of the user with any updated values set upon success.
// Returns the same, unmodified User and errors on error
func InsertUser(db *sqlx.DB, u User) (User, error) {
	// TODO: TEST CASE
	var updatedAt time.Time
	var createdAt time.Time
	// var userId *int32 = nil
	userId := new(int32)

	query := db.Rebind(`INSERT INTO users (email, first_name, last_name, password, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW()) RETURNING id, created_at, updated_at`)
	// TODO: just use StructScan?  Or at least scan right into user.Id?
	err := db.QueryRow(
		query, u.Email, u.FirstName, u.LastName, u.Password).Scan(userId, &createdAt, &updatedAt)
	if err != nil {
		return u, err
	}

	u.Id = userId
	u.CreatedAt = createdAt
	u.UpdatedAt = updatedAt
	return u, nil
}

// Saves the passed in user to the database using an UPDATE
// Returns a new copy of the user with any updated values set upon success.
// Returns the same, unmodified User and errors on error
func UpdateUser(db *sqlx.DB, u User) (User, error) {
	// TODO: TEST CASE
	var updatedAt time.Time
	query := db.Rebind(`UPDATE users SET email = ?, first_name = ?, last_name = ?, password = ?, updated_at = NOW()
		WHERE id = ?) RETURNING updated_at`)
	err := db.QueryRow(
		query, u.Email, u.FirstName, u.LastName, u.Password, u.Id).Scan(&updatedAt)
	if err != nil {
		return u, err
	}
	u.UpdatedAt = updatedAt
	return u, nil
}

// TODO: get all these lookup/find/etc named consistently
// Looks up the user by id in the database and returns a new User
func LookupUser(id int32, db *sqlx.DB) (*User, error) {
	// TODO: rename this FindUser() and implement other query stuff?
	// TODO: make this return nil if user is not found
	// or keep that separate?
	u := User{}
	query := db.Rebind(
		`SELECT id, first_name, last_name, email, password, created_at, updated_at, password FROM users WHERE id=?`)
	err := db.Get(&u, query, id)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Looks up the username (or email, as the case is for now) and verifies that the password
// matches that of the user.
// TODO: Just return a pointer to the user, nil if no user found or do a django-like AnonymousUser
// and make an interface for User and AnonymousUser
func AuthenticateLogin(username, password string, db *sqlx.DB) (*User, error) {
	u := new(User)
	u.Id = nil

	query := db.Rebind(
		"SELECT id, email, first_name, last_name, created_at, updated_at, password FROM users WHERE email = ?")
	err := db.Get(u, query, username)
	// I believe due to postgres having user_id be not null, our id is always a pointer to 0 after this
	// if the user was not found, which throws things off.
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, UserNotFoundError
		} else {
			return nil, err
		}
	}

	pwdErr := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if pwdErr != nil {
		return nil, pwdErr
	}

	return u, nil
}

// Uses a token as passed in authentication headers by a user to look them up
func LookupUserByToken(tokenStr string, db *sqlx.DB) (User, error) {
	// TODO: Is there a good way to abstract this so that token data could optionally
	// be stored in redis while other data is in postgres?  If two separate lookups
	// are done even for db then it is easy.

	// TODO: Considering making this taken token id and actual token as separate params
	// for explicitness that token is passed in has 2 parts
	tokenParts := strings.SplitN(tokenStr, ":", 2)
	if len(tokenParts) != 2 {
		return User{}, TokenFormatError
	}

	tokenId := tokenParts[0]
	tokenSecret := tokenParts[1]
	token := AuthToken{}
	query := db.Rebind(
		`SELECT t.token_id, t.token, t.scope, t.expires_at, t.created_at, t.updated_at, u.id "user.id", u.first_name "user.first_name", u.last_name "user.last_name", ` +
			`u.email "user.email", u.created_at "user.created_at", u.updated_at "user.updated_at", u.password "user.password" FROM user_authtokens t LEFT JOIN users u ON t.user_id = u.id ` +
			`WHERE t.token_id = ? AND (t.expires_at IS NULL OR t.expires_at > ?)`)
	err := db.Get(&token, query, tokenId, time.Now())

	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, InvalidTokenError
		}
		return User{}, err
	}

	if token == (AuthToken{}) {
		return User{}, InvalidTokenError
	}

	// could do this in the sql, but it keeps the hashing code all closer together
	if !token.Compare(tokenSecret) {
		return User{}, InvalidTokenError
	}

	return token.User, nil
}

func FindUser(params map[string]interface{}, db *sqlx.DB) (*User, error) {
	user := User{}
	var values []interface{}
	var where []string
	for _, k := range []string{"id", "email"} {
		if v, ok := params[k]; ok {
			values = append(values, v)
			// TODO: Deal with values from sensor OR user table
			where = append(where, fmt.Sprintf("u.%s = ?", k))
		}
	}

	selectCols := ""
	// as in BatchesForUser, this now seems dumb
	// queryCols := []string{"id", "name", "created_at", "updated_at", "user_id"}
	// If I need this many places, maybe make a const
	for _, k := range []string{"id", "email", "first_name", "last_name", "password", "created_at", "updated_at"} {
		selectCols += fmt.Sprintf("u.%s, ", k)
	}

	// TODO: Can I easily dynamically add in joining and attaching the User to this without overcomplicating the code?
	q := `SELECT ` + strings.Trim(selectCols, ", ") + ` FROM users u WHERE ` + strings.Join(where, " AND ")

	query := db.Rebind(q)
	err := db.Get(&user, query, values...)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
