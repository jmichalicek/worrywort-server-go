package graphqlApi

import (
	"context"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"strconv"
)

// Resolve a worrywort.Fermenter
type fermenterResolver struct {
	f *worrywort.Fermenter
}

func (r *fermenterResolver) ID() graphql.ID    { return graphql.ID(strconv.Itoa(r.f.Id)) }
func (r *fermenterResolver) CreatedAt() string { return dateString(r.f.CreatedAt) }
func (r *fermenterResolver) UpdatedAt() string { return dateString(r.f.UpdatedAt) }

// TODO: Make this return an actual nil if there is no createdBy, such as for a deleted user?
func (r *fermenterResolver) CreatedBy(ctx context.Context) *userResolver {
	// if UserResolver.u is a pointer, we can make this a simple one liner, possibly.
	// Need to see if there is difference in null resolver and resolver with null data.
	// I suspect there is
	if r.f.CreatedBy == nil {
		return nil
	}
	return &userResolver{u: r.f.CreatedBy}
}
