package worrywort

// "github.com/jmoiron/sqlx"
import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"time"
)

const InvalidTokenError = "Invalid token.  Not found."
const TokenFormatError = "Token should be formatted as `tokenId:secret` but was not"
const DefaultTokenHashCost int = 10  // to be faster than password hash cost because this will be calculated frequently

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
	TokenId   string             `db:"token_id"`
	Token     string             `db:"token"`
	User      User               `db:",prefix=u."`
	ExpiresAt time.Time          `db:"expires_at"`
	CreatedAt time.Time          `db:"created_at"`
	UpdatedAt time.Time          `db:"updated_at"`
	Scope     AuthTokenScopeType `db:"scope"`
}

type AuthToken struct {
	authToken
}
func (t AuthToken) ID() string { return t.authToken.TokenId }
func (t AuthToken) Token() string { return t.authToken.Token }
func (t AuthToken) ExpiresAt() time.Time { return t.authToken.ExpiresAt }
func (t AuthToken) CreatedAt() time.Time { return t.authToken.CreatedAt }
func (t AuthToken) UpdatedAt() time.Time { return t.authToken.UpdatedAt }
func (t AuthToken) Scope() AuthTokenScopeType { return t.authToken.Scope }
func (t AuthToken) User() User { return t.authToken.User }
func (t AuthToken) ForAuthenticationHeader() string {
	// TODO: Base64 encode this
	// "encoding/base64"
	return t.ID() + ":" + t.Token()
}
func (t AuthToken) Save() error {
	// TODO: Save the token to the db
	return nil
}

func NewToken(tokenId, token string, user User, scope AuthTokenScopeType, hashCost int) (AuthToken, error) {
	// use https://github.com/google/uuid
	// to make uuid NewRandom() function
	passwdBytes, err := bcrypt.GenerateFromPassword([]byte(token), hashCost)
	if err != nil {
		return AuthToken{}, err
	}

	return AuthToken{authToken{TokenId: tokenId, Token: string(passwdBytes), User: user, Scope: scope}}, nil
}

// Generate a random auth token for a user with the given scope
func GenerateTokenForUser(user User, scope AuthTokenScopeType) (AuthToken, error) {
	tokenId, err := uuid.NewRandom()
	if err != nil {
		return AuthToken{}, err
	}

	token, err := uuid.NewRandom()
	if err != nil {
		return AuthToken{}, err
	}

	return NewToken(tokenId.String(), token.String(), user, scope, DefaultTokenHashCost)
}
