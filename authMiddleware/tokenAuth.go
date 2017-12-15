// Middleware for basic token authentication where a token is directly
// tied to a user account.  This is intended for situations such as
// server to server API requests, etc. where a login -> get jwt -> request with jwt
// would be annoying
package authMiddleware

import (
	"context"
	"errors"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"net/http"
	"strings"
)

// TODO:  tests for this middleware like at https://medium.com/@PurdonKyle/unit-testing-golang-http-middleware-c7727ca896ea
// Eventually this will be configurable.
const DefaultUserKey string = "user"

// Type safe function to get user from context
func UserFromContext(ctx context.Context) (worrywort.User, error) {
	u, ok := ctx.Value(DefaultUserKey).(worrywort.User)
	if !ok {
		// can this differentiate between missing key and invalid value?
		return worrywort.User{}, errors.New("Could not get worrywort.User from context")
	}
	return u, nil
}

func newContextWithUser(ctx context.Context, req *http.Request, lookupFn func(string) worrywort.User) context.Context {
	authHeader := req.Header.Get("Authorization")
	headerParts := strings.Fields(authHeader)
	if len(headerParts) > 1 {
		if strings.ToLower(headerParts[0]) == "token" {
			return context.WithValue(ctx, DefaultUserKey, lookupFn(headerParts[1]))
		}
	}
	return ctx
}

func NewTokenAuthHandler(lookupFn func(string) worrywort.User) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			u, _ := UserFromContext(ctx)
			// only update context if user is not already populated.
			// what if it was ok, but user is an empty User{}?
			if (worrywort.User{}) == u {
				ctx = newContextWithUser(ctx, req, lookupFn)
			}
			next.ServeHTTP(rw, req.WithContext(ctx))
		})}
}
