package authMiddleware

import (
	"context"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTokenMiddleware(t *testing.T) {
	// Handler
	expectedUser := worrywort.NewUser(1, "jmichalicek@gmail.com", "Justin", "Michalicek", time.Now(), time.Now())
	getUser := func(token string) worrywort.User { return expectedUser }
	tokenAuthHandler := NewTokenAuthHandler(getUser)

	t.Run("Valid token header with no user should set user in context", func(t *testing.T) {
		// This should add the user to the request context.

		// Handler which errors if wrong user is in context
		handler := tokenAuthHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			ctxUser, ok := ctx.Value("user").(worrywort.User)
			if !ok {
				t.Errorf("Error getting user %v as worrywort.User", ctxUser)
			}
			if expectedUser != ctxUser {
				t.Errorf("Got %v but expected %v", ctxUser, expectedUser)
			}
			rw.WriteHeader(http.StatusOK)
		}))

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "token 12345")

		rr := httptest.NewRecorder()
		// make the request and test it
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
	})

	t.Run("Valid token header with already set user should leave existing user in context", func(t *testing.T) {
		// If a user is already set, such as by a different auth middleware such as http basic or jwt
		// then this middleware should not change that.

		u := worrywort.NewUser(2, "jmichalicek@gmail.com", "Justin", "Michalicek", time.Now(), time.Now())
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "token 12345")
		ctx := req.Context()
		ctx = context.WithValue(ctx, DefaultUserKey, u)
		req = req.WithContext(ctx)

		// Handler which errors is wrong user is in context
		handler := tokenAuthHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			ctxUser, ok := ctx.Value("user").(worrywort.User)
			if !ok {
				t.Errorf("Error getting user %v as worrywort.User", ctxUser)
			}
			if u != ctxUser {
				t.Errorf("Got %v but expected %v", ctxUser, u)
			}
			rw.WriteHeader(http.StatusOK)
		}))

		rr := httptest.NewRecorder()
		// make the request and test it
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
	})

	// TODO: test no matching token
}