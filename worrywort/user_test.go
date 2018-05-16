package worrywort

import (
	"database/sql"
	"database/sql/driver"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	"testing"
	"time"
)

// from https://github.com/DATA-DOG/go-sqlmock#matching-arguments-like-timetime
// stuff for matching times.  Need to improve this to match now-ish times
type AnyTime struct{}

// Match satisfies sqlmock.Argument interface
func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

func TestNewUser(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", createdAt, updatedAt)

	expectedUser := User{user{ID: 1, Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek",
		CreatedAt: createdAt, UpdatedAt: updatedAt}}
	if u != expectedUser {
		t.Errorf("Expected:\n\n%v\n\nGot:\n\n%v", expectedUser, u)
	}
}

func TestUserStruct(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now().Add(time.Hour * time.Duration(1))
	u := NewUser(1, "user@example.com", "Justin", "Michalicek", createdAt, updatedAt)

	t.Run("ID()", func(t *testing.T) {
		var actual int = u.ID()
		expected := u.user.ID
		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})

	t.Run("LastName()", func(t *testing.T) {
		var actual string = u.LastName()
		expected := u.user.LastName
		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})

	t.Run("FirstName()", func(t *testing.T) {
		var actual string = u.FirstName()
		expected := u.user.FirstName
		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})

	t.Run("Email()", func(t *testing.T) {
		var actual string = u.Email()
		expected := u.user.Email
		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var actual time.Time = u.CreatedAt()
		expected := u.user.CreatedAt
		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var actual time.Time = u.UpdatedAt()
		expected := u.user.UpdatedAt
		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})

	t.Run("SetUserPassword()", func(t *testing.T) {
		password := "password"
		// Not really part of User, but whatever for now.
		// I believe the password hashing makes this test slow.  Should do like Django
		// and use faster hashing for tests, perhaps, or reduce bcrypt cost at least
		updatedUser, err := SetUserPassword(u, "password", bcrypt.MinCost)
		if err != nil {
			t.Errorf("Unexpected error hashing password: %v", err)
		}

		if bcrypt.CompareHashAndPassword([]byte(updatedUser.Password()), []byte(password)) != nil {
			t.Errorf("SetUserPassword() did not hash and set the password as expected")
		}
	})
}

func TestLookupUser(t *testing.T) {
	// TODO: Consider using DATA-DOG/txdb to use a real db, which could matter when doing things
	// using postgres specific functionality, etc.
	// final return is err... might want to deal with that?
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	user := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())

	t.Run("Test valid user id returns user", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "created_at", "updated_at", "password"}).
			AddRow(user.ID(), user.Email(), user.FirstName(), user.LastName(), user.CreatedAt(), user.UpdatedAt(), user.Password())
		mock.ExpectQuery(`^SELECT (.+) FROM users WHERE id=\?`).WithArgs(user.ID()).WillReturnRows(rows)
		actual, err := LookupUser(1, sqlxDB)

		if err != nil {
			t.Errorf("LookupUser() returned error %v", err)
		}

		if user != actual {
			t.Errorf("Expected: %v, got: %v", user, actual)
		}
	})

	t.Run("Test invalid user id returns empty user", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "created_at", "updated_at", "password"})
		mock.ExpectQuery(`^SELECT (.+) FROM users WHERE id=?`).WithArgs(1).WillReturnRows(rows)
		actual, err := LookupUser(1, sqlxDB)
		expected := User{}

		if err != sql.ErrNoRows {
			t.Errorf("LookupUser() expected error: %v, but returned %v", sql.ErrNoRows, err)
		}

		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})
}

func TestLookupUserByToken(t *testing.T) {
	// TODO: Consider using DATA-DOG/txdb to use a real db, which could matter when doing things
	// using postgres specific functionality, etc. and allows testing to ensure that the query
	// returns expected data rather than this where we force expected returned rows, assuming that the sql is correct.
	// final return is err... might want to deal with that?
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	user := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	tokenId := "tokenid"
	tokenKey := "secret"
	token := NewToken(tokenId, tokenKey, user, TOKEN_SCOPE_ALL)

	t.Run("Test valid token returns user", func(t *testing.T) {
		tokenStr := tokenId + ":" + tokenKey

		// approximately correct with join for single query but need to figure out how to make sqlx handle This
		// looks like it should at https://github.com/jmoiron/sqlx/issues/131
		rows := sqlmock.NewRows(
			[]string{"token_id", "token", "scope", "expires_at", "created_at", "updated_at", "user.id", "user.email",
				"user.first_name", "user.last_name", "user.created_at", "user.updated_at", "user.password"}).
			AddRow(token.ID(), token.Token(), token.Scope(), token.ExpiresAt(), token.CreatedAt(), token.UpdatedAt(), user.ID(),
				user.Email(), user.FirstName(), user.LastName(), user.CreatedAt(), user.UpdatedAt(), user.Password())
		mock.ExpectQuery(`^SELECT (.+) FROM user_authtokens t LEFT JOIN users u ON t.user_id = u.id WHERE t.token_id = \? AND \(t.expires_at IS NULL OR t.expires_at > \?\)`).
			WithArgs(tokenId, AnyTime{}).WillReturnRows(rows)

		actual, err := LookupUserByToken(tokenStr, sqlxDB)

		if err != nil {
			t.Errorf("TestLookupUserByToken() returned error %v", err)
		}

		if user != actual {
			t.Errorf("Expected: %v, got: %v", user, actual)
		}
	})

	t.Run("Test invalid token returns empty user", func(t *testing.T) {

		tokenStr := "tokenid:tokenstr"

		tokenRows := sqlmock.NewRows([]string{"token_id", "token", "scope", "expires_at", "created_at", "updated_at", "user.id",
			"user.email", "user.first_name", "user.last_name", "user.created_at", "user.updated_at"})
		mock.ExpectQuery(`^SELECT (.+) FROM user_authtokens t LEFT JOIN users u ON t.user_id = u.id WHERE t.token_id = \? AND \(t.expires_at IS NULL OR t.expires_at > \?\)`).
			WithArgs(tokenId, AnyTime{}).WillReturnRows(tokenRows)

		actual, err := LookupUserByToken(tokenStr, sqlxDB)
		expected := User{}

		if err != sql.ErrNoRows {
			t.Errorf("\nExpected error: %v\nGot: %v", sql.ErrNoRows, err)
		}

		if actual != expected {
			t.Errorf("\nExpected: %v\ngot: %v", expected, actual)
		}
	})
}

func TestAuthenticateLogin(t *testing.T) {
	// back to this once the query is tested
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	user := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	password := "password"
	user, _ = SetUserPassword(user, password, bcrypt.MinCost)
	if err != nil {
		t.Fatalf("Error hashing password for test: %v", err)
	}

	t.Run("Test valid username and password returns User", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "created_at", "updated_at", "password"}).
			AddRow(user.ID(), user.Email(), user.FirstName(), user.LastName(), user.CreatedAt(), user.UpdatedAt(), user.Password())

		mock.ExpectQuery(`^SELECT (.+) FROM users WHERE email = \?`).WithArgs(user.Email()).WillReturnRows(rows)

		u, err := AuthenticateLogin(user.Email(), password, sqlxDB)
		if err != nil {
			t.Errorf("Got unexpected error: %v", err)
		}

		if u != user {
			t.Errorf("Expected user: %v\ngot: %v", user, u)
		}
	})

	t.Run("Test valid username and password mistmatch returns error and empty User{}", func(t *testing.T) {
		badPass, _ := bcrypt.GenerateFromPassword([]byte("a"), bcrypt.MinCost)
		rows := sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "created_at", "updated_at", "password"}).
			AddRow(user.ID(), user.Email(), user.FirstName(), user.LastName(), user.CreatedAt(), user.UpdatedAt(), string(badPass))

		mock.ExpectQuery(`^SELECT (.+) FROM users WHERE email = \?`).WithArgs(user.Email()).WillReturnRows(rows)

		u, err := AuthenticateLogin(user.Email(), password, sqlxDB)
		if err != bcrypt.ErrMismatchedHashAndPassword {
			t.Errorf("Expected error: %v\nGot: %v", UserNotFoundError, err)
		}

		if u != (User{}) {
			t.Errorf("Expected empty user: %v\ngot: %v", User{}, u)
		}
	})

	// TODO: test mismatched email
	t.Run("Test invalid username/email and returns error and User{}", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "created_at", "updated_at", "password"})

		mock.ExpectQuery(`^SELECT (.+) FROM users WHERE email = \?`).WithArgs("nomatch@example.com").WillReturnRows(rows)

		u, err := AuthenticateLogin("nomatch@example.com", password, sqlxDB)
		if err != UserNotFoundError {
			t.Errorf("Expected: %v\nGot : %v", UserNotFoundError, err)
		}

		if u != (User{}) {
			t.Errorf("Expected empty user: %v\ngot: %v", user, u)
		}
	})

}
