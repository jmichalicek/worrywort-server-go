// Middleware for basic token authentication where a token is directly
// tied to a user account.  This is intended for situations such as
// server to server API requests, etc. where a login -> get jwt -> request with jwt
// would be annoying
package middleware

import (
	"context"
	"errors"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"log"
	"net/http"
	"strings"
	// "github.com/davecgh/go-spew/spew"
)

// TODO:  tests for this middleware like at https://medium.com/@PurdonKyle/unit-testing-golang-http-middleware-c7727ca896ea
// Eventually this will be configurable.
// TODO: Now that this package is just `middleware` and not auth specific, this const feels either poorly named or misplaced.
const DefaultUserKey string = "user"

var ErrUserNotInContext = errors.New("Could not get worrywort.User from context")

// Type safe function to get user from context
func UserFromContext(ctx context.Context) (*worrywort.User, error) {
	// May return *worrywort.User so that I can return nil
	u, ok := ctx.Value(DefaultUserKey).(*worrywort.User)
	if !ok {
		// can this differentiate between missing key and invalid value?
		return nil, ErrUserNotInContext
	}
	return u, nil
}

func newContextWithUser(ctx context.Context, req *http.Request, lookupFn func(string) (*worrywort.User, error)) context.Context {
	authHeader := req.Header.Get("Authorization")
	headerParts := strings.Fields(authHeader)
	if len(headerParts) > 1 {
		if strings.ToLower(headerParts[0]) == "token" {
			// TODO: Handle error here.  If it's no rows returned, then no big deal
			// but anything else may need handled or logged
			user, err := lookupFn(headerParts[1])
			if err != nil {
				if err != worrywort.ErrInvalidToken {
					log.Printf("%v", err)
				}
				return ctx
			} else {
				return context.WithValue(ctx, DefaultUserKey, user)
			}
		}
	}
	return ctx
}

// a middleware to handle token auth
// This is really overkill - the injected function could just live here since this is not really intended
// to be a generic, reusable thing.  This does make testing easier, though, since I can inject a function which
// just returns what I need and not mock out a db connection.
func NewTokenAuthHandler(lookupFn func(string) (*worrywort.User, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			u, _ := UserFromContext(ctx)
			// only update context if user is not already populated.
			if u == nil {
				ctx = newContextWithUser(ctx, req, lookupFn)
			}
			next.ServeHTTP(rw, req.WithContext(ctx))
		})
	}
}
