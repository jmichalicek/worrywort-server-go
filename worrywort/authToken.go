package worrywort

import (
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"time"
	// "log"
)

var InvalidTokenError error = errors.New("Invalid token. Not found.")

var TokenFormatError = errors.New("Token should be formatted as `tokenId:secret` but was not")

// TODO: Possibly move authToken stuff to its own package so that scope stuff will be
// authToken.READ_ALL, etc.
type AuthTokenScopeType int64

const (
	TOKEN_SCOPE_ALL AuthTokenScopeType = iota
	TOKEN_SCOPE_READ_ALL
	TOKEN_SCOPE_WRITE_TEMPS
	TOKEN_SCOPE_READ_TEMPS
)

// Simplified auth tokens.  May eventually be replaced with proper OAuth 2.
type AuthToken struct {
	// really could use email as the pk for the db, but fudging it because I've been trained by ORMs
	Id         string             `db:"id"`  // Can this just be a uuid type?
	Token      string             `db:"token"`
	User       User               `db:"user,prefix=u."`
	ExpiresAt  pq.NullTime        `db:"expires_at"`
	CreatedAt  time.Time          `db:"created_at"`
	UpdatedAt  time.Time          `db:"updated_at"`
	Scope      AuthTokenScopeType `db:"scope"`
	fromString string             // usually empty, the string this token was generated from
}

func (t AuthToken) ForAuthenticationHeader() string {
	// TODO: Base64 encode this?
	// "encoding/base64"
	return t.Id + ":" + t.fromString
}
func (t *AuthToken) Save(db *sqlx.DB) error {
	// TODO: May change the name of this table as it suggests a joining table.
	tokenId := new(string)
	createdAt := new(time.Time)
	updatedAt := new(time.Time)
	if t.Id != "" {
		return nil
	}
	query := db.Rebind(`INSERT INTO user_authtokens (token, expires_at, updated_at, scope, user_id)
		VALUES (?, ?, NOW(), ?, ?) RETURNING id, created_at, updated_at`)
	err := db.QueryRow(
		query, t.Token, t.ExpiresAt, t.Scope, t.User.Id).Scan(tokenId, createdAt, updatedAt)
	if err == nil {
		t.Id = *tokenId
		t.CreatedAt = *createdAt
		t.UpdatedAt = *updatedAt
	}
	return err
}

func (t AuthToken) Compare(token string) bool {
	tokenHash := MakeTokenHash(token)
	return tokenHash == t.Token
}

// Make a hashed token from string.  May rename
func MakeTokenHash(tokenStr string) string {
	tokenBytes := sha512.Sum512([]byte(tokenStr))
	// tokenBytes is a byte array, which cannot be directly cast to a string.  Instead make it a
	// byte slice for EncodeToString, which does cast properly.
	token := base64.URLEncoding.EncodeToString(tokenBytes[:])
	return token
}

// Returns an AuthToken with a hashed token for a given tokenId and token string
func NewToken(token string, user User, scope AuthTokenScopeType) AuthToken {
	// TODO: instead of taking hashCost, take a function which hashes the passwd
	// for now use SHA512.  The token as a uuidv4 already has significant entropy, so salt is
	// less necessary.  Users must already login with a slow hash before generating a token here,
	// and for API usage on every request this token needs to be fast to calculate.
	tokenString := MakeTokenHash(token)
	return AuthToken{Token: tokenString, User: user, Scope: scope, fromString: token}
}

// Generate a random auth token for a user with the given scope
func GenerateTokenForUser(user User, scope AuthTokenScopeType) (AuthToken, error) {
	// TODO: instead of taking hashCost, take a function which hashes the passwd - this could then do bcrypt at any cost,
	// pbkdf2, or for testing situations a simple md5 or just leave alone.
	token, err := uuid.NewRandom()
	if err != nil {
		return AuthToken{}, err
	}

	// not sure there's much point to this, but it makes it nicer looking
	tokenb64 := base64.URLEncoding.EncodeToString([]byte(token.String()))
	return NewToken(tokenb64, user, scope), nil
}
