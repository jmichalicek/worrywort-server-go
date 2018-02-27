package worrywort

import (
	"crypto/sha512"
	"encoding/base64"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"time"
)

const InvalidTokenError = "Invalid token.  Not found."
const TokenFormatError = "Token should be formatted as `tokenId:secret` but was not"
const DefaultTokenHashCost int = 10 // to be faster than password hash cost because this will be calculated frequently

// TODO: Possibly move authToken stuff to its own package so that scope stuff will be
// authToken.READ_ALL, etc.
type AuthTokenScopeType int32

const (
	TOKEN_SCOPE_ALL AuthTokenScopeType = iota
	TOKEN_SCOPE_READ_ALL
	TOKEN_SCOPE_WRITE_TEMPS
	TOKEN_SCOPE_READ_TEMPS
)

// Simplified auth tokens.  May eventually be replaced with proper OAuth 2.
type authToken struct {
	// really could use email as the pk for the db, but fudging it because I've been trained by ORMs
	Id        string             `db:"token_id"`
	Token     string             `db:"token"`
	User      User               `db:",prefix=u."`
	ExpiresAt pq.NullTime        `db:"expires_at"`
	CreatedAt time.Time          `db:"created_at"`
	UpdatedAt time.Time          `db:"updated_at"`
	Scope     AuthTokenScopeType `db:"scope"`
	fromString	string  // usually empty, the string this token was generated from
}

type AuthToken struct {
	authToken
}

func (t AuthToken) ID() string                { return t.authToken.Id }
func (t AuthToken) Token() string             { return t.authToken.Token }
func (t AuthToken) ExpiresAt() pq.NullTime    { return t.authToken.ExpiresAt }
func (t AuthToken) CreatedAt() time.Time      { return t.authToken.CreatedAt }
func (t AuthToken) UpdatedAt() time.Time      { return t.authToken.UpdatedAt }
func (t AuthToken) Scope() AuthTokenScopeType { return t.authToken.Scope }
func (t AuthToken) User() User                { return t.authToken.User }
func (t AuthToken) ForAuthenticationHeader() string {
	// TODO: Base64 encode this?
	// "encoding/base64"
	return t.ID() + ":" + t.fromString
}
func (t AuthToken) Save(db *sqlx.DB) error {
	// TODO: Save the token to the db
	// TODO: May change the name of this table as it suggests a joining table.
	if t.CreatedAt().IsZero() {
		query := "INSERT INTO user_authtokens (token_id, token, expires_at, updated_at, scope, user_id) VALUES (?, ?, ?, ?, ?, ?, ?)"
		_, err := db.Exec(query, t.ID(), t.Token(), t.ExpiresAt(), time.Now(), t.Scope(), t.User().ID())
		if err != nil {
			return err
		}
	}
	// No update allowed for now.
	//TODO: decide if this will update and what can be updated.  Perhaps can update scope and expiration?  or maybe nothing.
	return nil
}

func (t AuthToken) Compare(token string) bool {
	tokenHash := MakeTokenHash(token)
	return tokenHash == t.Token()
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
func NewToken(tokenId, token string, user User, scope AuthTokenScopeType) AuthToken {
	// TODO: instead of taking hashCost, take a function which hashes the passwd
	// for now use SHA512.  The token as a uuidv4 already has significant entropy, so salt is
	// less necessary.  Users must already login with a slow hash before generating a token here,
	// and for API usage on every request this token needs to be fast to calculate.
	tokenString := MakeTokenHash(token)
	return AuthToken{authToken{Id: tokenId, Token: tokenString, User: user, Scope: scope, fromString: token}}
}

// Generate a random auth token for a user with the given scope
func GenerateTokenForUser(user User, scope AuthTokenScopeType) (AuthToken, error) {
	// TODO: instead of taking hashCost, take a function which hashes the passwd - this could then do bcrypt at any cost,
	// pbkdf2, or for testing situations a simple md5 or just leave alone.
	tokenId, err := uuid.NewRandom()
	if err != nil {
		return AuthToken{}, err
	}

	token, err := uuid.NewRandom()
	if err != nil {
		return AuthToken{}, err
	}

	// not sure there's much point to this, but it makes it nicer looking
	tokenb64 := base64.URLEncoding.EncodeToString([]byte(token.String()))

	return NewToken(tokenId.String(), tokenb64, user, scope), nil
}
