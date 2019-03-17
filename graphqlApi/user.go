package graphqlApi

import (
	"github.com/davecgh/go-spew/spew"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"log"
)

type userResolver struct {
	u *worrywort.User
}

func (r *userResolver) ID() graphql.ID {
	if r.u == nil {
		log.Printf("user resolver with nil user: %s", spew.Sdump(r))
		return graphql.ID("")
	} else {
		return graphql.ID(r.u.Uuid)
	}
}
func (r *userResolver) FirstName() string { return r.u.FirstName }
func (r *userResolver) LastName() string  { return r.u.LastName }
func (r *userResolver) Email() string     { return r.u.Email }
func (r *userResolver) CreatedAt() string { return dateString(r.u.CreatedAt) }
func (r *userResolver) UpdatedAt() string { return dateString(r.u.UpdatedAt) }
