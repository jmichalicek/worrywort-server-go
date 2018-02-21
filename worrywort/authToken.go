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

type AuthTokenScopeType int32

const (
	ALL AuthTokenScopeType = iota
	READ_ALL
	WRITE_TEMPS
	READ_TEMPS
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

func (t authToken) ID() string { return t.TokenId }
func (t authToken) Save() error {
	// TODO: Save the token to the db
	return nil
}

func NewToken(tokenId, token string, user User, scope AuthTokenScopeType, hashCost int) (authToken, error) {
	// use https://github.com/google/uuid
	// to make uuid NewRandom() function
	passwdBytes, err := bcrypt.GenerateFromPassword([]byte(token), hashCost)
	if err != nil {
		return authToken{}, err
	}

	return authToken{TokenId: tokenId, Token: string(passwdBytes), User: user, Scope: scope}, nil
}

// Generate a random auth token for a user with the given scope
func GenerateTokenForUser(user User, scope AuthTokenScopeType) (authToken, error) {
	tokenId, err := uuid.NewRandom()
	if err != nil {
		return authToken{}, err
	}

	token, err := uuid.NewRandom()
	if err != nil {
		return authToken{}, err
	}

	return NewToken(tokenId.String(), token.String(), user, scope, DefaultTokenHashCost)
}