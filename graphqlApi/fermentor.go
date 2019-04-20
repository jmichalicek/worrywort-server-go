package graphqlApi

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"log"
)

// Resolve a worrywort.Fermentor
type fermentorResolver struct {
	f *worrywort.Fermentor
}

func (r *fermentorResolver) ID() graphql.ID {
	if r.f != nil {
		return graphql.ID(r.f.UUID)
	} else {
		log.Printf("Nil Id on fermentor: %v", spew.Sdump(r))
		return graphql.ID("")
	}
}
func (r *fermentorResolver) CreatedAt() string { return dateString(r.f.CreatedAt) }
func (r *fermentorResolver) UpdatedAt() string { return dateString(r.f.UpdatedAt) }
func (r *fermentorResolver) CreatedBy(ctx context.Context) *userResolver {
	var resolved *userResolver
	// Not sure these parens are necessary, but vs code complains without them
	// because it seems to think I am referring to this function
	if (r.f.CreatedBy) != nil {
		resolved = &userResolver{u: r.f.CreatedBy}
	} else if r.f.UserId != nil {
		db, ok := ctx.Value("db").(*sqlx.DB)
		if !ok {
			log.Printf("No database in context")
			return nil
		}
		user, err := worrywort.FindUser(map[string]interface{}{"id": *r.f.UserId}, db)

		if err != nil {
			log.Printf("%v", err)
			return nil
		}
		resolved = &userResolver{u: user}
	}
	return resolved
}
