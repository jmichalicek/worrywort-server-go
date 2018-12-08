package graphqlApi

import (
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"strconv"
)

type userResolver struct {
	u *worrywort.User
}

func (r *userResolver) ID() graphql.ID    { return graphql.ID(strconv.Itoa(r.u.Id)) }
func (r *userResolver) FirstName() string { return r.u.FirstName }
func (r *userResolver) LastName() string  { return r.u.LastName }
func (r *userResolver) Email() string     { return r.u.Email }
func (r *userResolver) CreatedAt() string { return dateString(r.u.CreatedAt) }
func (r *userResolver) UpdatedAt() string { return dateString(r.u.UpdatedAt) }