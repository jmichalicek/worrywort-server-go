package authMiddleware

import (
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"net/http"
)

// A simple middleware to return a 403 if no authenticated user
func NewLoginRequiredHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			u, _ := UserFromContext(ctx)
			// only update context if user is not already populated.
			// what if it was ok, but user is an empty User{}?
			if (worrywort.User{}) == u {
				// http.StatusUnauthorized == 403
				http.Error(rw, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(rw, req.WithContext(ctx))
		})
	}
}
