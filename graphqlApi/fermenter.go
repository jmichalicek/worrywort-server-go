package graphqlApi

import (
	"context"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"log"
	"strconv"
)

// Resolve a worrywort.Fermentor
type fermentorResolver struct {
	f *worrywort.Fermentor
}

func (r *fermentorResolver) ID() graphql.ID    { return graphql.ID(strconv.Itoa(r.f.Id)) }
func (r *fermentorResolver) CreatedAt() string { return dateString(r.f.CreatedAt) }
func (r *fermentorResolver) UpdatedAt() string { return dateString(r.f.UpdatedAt) }

// TODO: Make this return an actual nil if there is no createdBy, such as for a deleted user?
func (r *fermentorResolver) CreatedBy(ctx context.Context) *userResolver {
	// if UserResolver.u is a pointer, we can make this a simple one liner, possibly.
	// Need to see if there is difference in null resolver and resolver with null data.
	// I suspect there is
	var resolved *userResolver
	// Not sure these parens are necessary, but vs code complains without them
	// because it seems to think I am referring to this function
	if (r.f.CreatedBy) != nil {
		resolved = &userResolver{u: r.f.CreatedBy}
	} else if r.f.UserId.Valid {
		db, ok := ctx.Value("db").(*sqlx.DB)
		if !ok {
			log.Printf("No database in context")
			return nil
		}
		user, err := worrywort.LookupUser(int(r.f.UserId.Int64), db)
		if err != nil {
			log.Printf("%v", err)
			return nil
		}
		resolved = &userResolver{u: user}
	}
	return resolved
}
