package graphql_api

import (
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
)

// An auth token returned after logging in to use in Authentication headers
type authTokenResolver struct {
	t worrywort.AuthToken
	// return a status such as ok or error?
}

func (a *authTokenResolver) ID() graphql.ID { return graphql.ID(a.t.ForAuthenticationHeader()) }
func (a *authTokenResolver) Token() string  { return a.t.ForAuthenticationHeader() }
