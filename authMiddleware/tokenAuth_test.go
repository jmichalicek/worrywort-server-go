package authMiddleware

import (
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTokenMiddleware(t *testing.T) {
	// Handler
	uid := int64(1)
	expectedUser := worrywort.User{Id: &uid, Email: "jmichalicek@gmail.com", FirstName: "Justin", LastName: "Michalicek",
		CreatedAt: time.Now(), UpdatedAt: time.Now()}
	getUser := func(token string) (worrywort.User, error) { return expectedUser, nil }
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
		uid := int64(2)
		u := worrywort.User{Id: &uid, Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}

		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "token 12345")
		ctx := req.Context()
		ctx = context.WithValue(ctx, DefaultUserKey, &u)
		req = req.WithContext(ctx)

		// Handler which errors is wrong user is in context
		handler := tokenAuthHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			ctxUser, ok := ctx.Value("user").(*worrywort.User)
			if !ok {
				t.Errorf("Error getting user %v as worrywort.User", ctxUser)
			}
			if !cmp.Equal(&u, ctxUser) {
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
