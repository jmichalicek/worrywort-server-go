package worrywort

import (
	"database/sql"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"testing"
	"time"
)

func TestNewUser(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := NewUser(nil, "user@example.com", "Justin", "Michalicek", createdAt, updatedAt)
	expectedUser := User{Id: u.Id, Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek",
		CreatedAt: createdAt, UpdatedAt: updatedAt}
	if u != expectedUser {
		t.Errorf("Expected:\n\n%v\n\nGot:\n\n%v", expectedUser, u)
	}
	if u.Id != nil {
		t.Errorf("NewUser returned with unexpected user id %v", u.Id)
	}
}

func TestUserStruct(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now().Add(time.Hour * time.Duration(1))
	u := NewUser(nil, "user@example.com", "Justin", "Michalicek", createdAt, updatedAt)

	t.Run("SetUserPassword()", func(t *testing.T) {
		password := "password"
		// Not really part of User, but whatever for now.
		// I believe the password hashing makes this test slow.  Should do like Django
		// and use faster hashing for tests, perhaps, or reduce bcrypt cost at least
		updatedUser, err := SetUserPassword(u, "password", bcrypt.MinCost)
		if err != nil {
			t.Errorf("Unexpected error hashing password: %v", err)
		}

		if bcrypt.CompareHashAndPassword([]byte(updatedUser.Password), []byte(password)) != nil {
			t.Errorf("SetUserPassword() did not hash and set the password as expected")
		}
	})
}

func TestUserDatabaseFunctionality(t *testing.T) {
	// Subtests which use the database and a user so that user is only saved once, password only saved once, etc.
	// If modifications to user start happening here then need to see if txdb is wrapping each t.Run() or if
	// I need to find a way to do that manually.
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()
	user := NewUser(nil, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	password := "password"
	user, err = SetUserPassword(user, password, bcrypt.MinCost)
	if err != nil {
		t.Fatalf("Error hashing password for test: %v", err)
	}
	user, err = SaveUser(db, user)
	if err != nil {
		t.Fatalf("Error setting up test user: %v", err)
	}

	//func TestLookupUser(t *testing.T) {
	t.Run("TestLookupUser", func(t *testing.T) {
		t.Run("Test valid user id returns user", func(t *testing.T) {
			actual, err := LookupUser(*user.Id, db)

			if err != nil {
				t.Errorf("LookupUser() returned error %v", err)
			}

			if !cmp.Equal(user, *actual) {
				t.Errorf("Expected: -  Got: +\n%s", cmp.Diff(&user, actual))
			}
		})

		t.Run("Test invalid user id returns empty user", func(t *testing.T) {
			actual, err := LookupUser(0, db)

			if err != sql.ErrNoRows {
				t.Errorf("Expected error: %v\ngot: %v\n", sql.ErrNoRows, err)
			}

			if actual != nil {
				t.Errorf("Expected: %v\ngot: %v\n", nil, actual)
			}
		})
	})

	t.Run("TestLookupUserByToken", func(t *testing.T) {
		tokenKey := "secret"
		token := NewToken(tokenKey, user, TOKEN_SCOPE_ALL)
		token.Save(db)
		tokenId := token.Id

		t.Run("Test valid token returns user", func(t *testing.T) {
			tokenStr := tokenId + ":" + tokenKey
			actual, err := LookupUserByToken(tokenStr, db)

			if err != nil {
				t.Errorf("LookupUserByToken() returned error %v", err)
			}

			if !cmp.Equal(user, actual) {
				// this or cmp.Diff()?  This is easier to tell which was expected
				t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(user, actual))
			}
		})

		t.Run("Test invalid token with valid token id", func(t *testing.T) {
			tokenStr := token.Id + ":tokenstr"
			actual, err := LookupUserByToken(tokenStr, db)
			expected := User{}

			if err != InvalidTokenError {
				t.Errorf("\nExpected error: %v\nGot: %v", InvalidTokenError, err)
			}

			if actual != expected {
				t.Errorf("\nExpected: %v\ngot: %v", expected, actual)
			}
		})

		t.Run("Test invalid token id", func(t *testing.T) {
			badTokenId, err := uuid.NewRandom()
			tokenStr := badTokenId.String() + ":tokenstr"
			actual, err := LookupUserByToken(tokenStr, db)
			expected := User{}

			if err != InvalidTokenError {
				t.Errorf("\nExpected error: %v\nGot: %v", InvalidTokenError, err)
			}

			if actual != expected {
				t.Errorf("\nExpected: %v\ngot: %v", expected, actual)
			}
		})
	})

	t.Run("TestAuthenticateLogin", func(t *testing.T) {
		t.Run("Test valid username and password returns User", func(t *testing.T) {
			u, err := AuthenticateLogin(user.Email, password, db)
			if err != nil {
				t.Errorf("Got unexpected error: %v", err)
			}

			if !cmp.Equal(u, &user) {
				t.Errorf("User did not match: %v", cmp.Diff(u, &user))
			}
		})

		t.Run("Test valid username and password mistmatch returns error and empty User{}", func(t *testing.T) {
			u, err := AuthenticateLogin(user.Email, "a", db)
			if err != bcrypt.ErrMismatchedHashAndPassword {
				t.Errorf("Expected error: %v\nGot: %v", bcrypt.ErrMismatchedHashAndPassword, err)
			}

			if u != nil {
				t.Errorf("Expected: %v\ngot: %v", nil, u)
			}
		})

		// TODO: test mismatched email
		t.Run("Test invalid username/email and returns error and User{}", func(t *testing.T) {
			u, err := AuthenticateLogin("nomatch@example.com", password, db)
			if err != UserNotFoundError {
				t.Errorf("Expected: %v\nGot: %v", UserNotFoundError, err)
			}

			if u != nil {
				t.Errorf("Expected: %v\ngot: %v", nil, u)
			}
		})
	})
}
