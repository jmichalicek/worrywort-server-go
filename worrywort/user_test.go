package worrywort

import (
	"database/sql"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"testing"
	"time"
)

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
		var actual int64 = u.ID()
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
		rows := sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "created_at", "updated_at"}).
			AddRow(user.ID(), user.Email(), user.FirstName(), user.LastName(), user.CreatedAt(), user.UpdatedAt())
		mock.ExpectQuery(`^SELECT (.+) FROM users WHERE id=?`).WithArgs(user.ID()).WillReturnRows(rows)
		actual, err := LookupUser(1, sqlxDB)

		if err != nil {
			t.Errorf("LookupUser() returned error %v", err)
		}

		if user != actual {
			t.Errorf("Expected: %v, got: %v", user, actual)
		}
	})

	t.Run("Test invalid user id returns empty user", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "created_at", "updated_at"})
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
	// using postgres specific functionality, etc.
	// final return is err... might want to deal with that?
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	user := NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())

	t.Run("Test valid token returns user", func(t *testing.T) {
		token := "asdf1234"
		rows := sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "created_at", "updated_at"}).
			AddRow(user.ID(), user.Email(), user.FirstName(), user.LastName(), user.CreatedAt(), user.UpdatedAt())
		mock.ExpectQuery(`^SELECT (.+) FROM user_authtokens t LEFT JOIN users u ON t.user_id = u.id WHERE t.token=?`).
			WithArgs(token).WillReturnRows(rows)
		actual, err := LookupUserByToken(token, sqlxDB)

		if err != nil {
			t.Errorf("TestLookupUserByToken() returned error %v", err)
		}

		if user != actual {
			t.Errorf("Expected: %v, got: %v", user, actual)
		}
	})

	t.Run("Test invalid token returns empty user", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "email", "first_name", "last_name", "created_at", "updated_at"})
		mock.ExpectQuery(`^SELECT (.+) FROM user_authtokens t LEFT JOIN users u ON t.user_id = u.id WHERE t.token=?`).
			WithArgs("").WillReturnRows(rows)
		actual, err := LookupUserByToken("", sqlxDB)
		expected := User{}

		if err != sql.ErrNoRows {
			t.Errorf("TestLookupUserByToken() expected error: %v, but returned %v", sql.ErrNoRows, err)
		}

		if actual != expected {
			t.Errorf("Expected: %v, got: %v", expected, actual)
		}
	})
}
