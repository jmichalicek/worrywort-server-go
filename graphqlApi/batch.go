package graphqlApi

import (
	"context"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"log"
	"strconv"
)

// Batch resolver related code
type batchResolver struct {
	b *worrywort.Batch
}

func (r *batchResolver) ID() graphql.ID {
	return graphql.ID(strconv.Itoa(r.b.Id))
}
func (r *batchResolver) Name() string         { return r.b.Name }
func (r *batchResolver) BrewNotes() string    { return r.b.BrewNotes }
func (r *batchResolver) TastingNotes() string { return r.b.TastingNotes }

// TODO: I should make an actual DateTime type which can be null or a valid datetime string
func (r *batchResolver) BrewedDate() *string {
	return nullableDateString(r.b.BrewedDate)
}
func (r *batchResolver) BottledDate() *string { return nullableDateString(r.b.BottledDate) }
func (r *batchResolver) VolumeBoiled() *float64 {
	// TODO: I do not like this.  Maybe switch the data type to sql.NullFloat64?
	vol := r.b.VolumeBoiled
	if vol == 0 {
		return nil
	}
	return &vol
}

func (r *batchResolver) VolumeInFermentor() *float64 {
	vol := r.b.VolumeInFermentor
	// TODO: I do not like this.  Maybe switch the data type to sql.NullFloat64?
	if vol == 0 {
		return nil
	}
	return &vol
}

func (r *batchResolver) VolumeUnits() worrywort.VolumeUnitType { return r.b.VolumeUnits }
func (r *batchResolver) OriginalGravity() *float64 {
	// TODO: I do not like this.  Maybe switch the data type to https://godoc.org/gopkg.in/guregu/null.v3 nullint
	// on the struct
	og := r.b.OriginalGravity
	if og == 0 {
		return nil
	}
	return &og
}

func (r *batchResolver) FinalGravity() *float64 {
	fg := r.b.FinalGravity

	// TODO: I do not like this.  Maybe switch the data type to https://godoc.org/gopkg.in/guregu/null.v3 nullint
	// on the struct
	if fg == 0 {
		return nil
	}
	return &fg
}

func (r *batchResolver) RecipeURL() string { return r.b.RecipeURL } // this could even return a parsed URL object...
func (r *batchResolver) CreatedAt() string { return dateString(r.b.CreatedAt) }
func (r *batchResolver) UpdatedAt() string { return dateString(r.b.UpdatedAt) }

// TODO: Make this return an actual nil if there is no createdBy, such as for a deleted user?
func (r *batchResolver) CreatedBy(ctx context.Context) (*userResolver, error) {
	// IMPLEMENT DATALOADER
	// TODO: yeah, maybe make Batch.CreatedBy and others a pointer... or a function with a private pointer to cache
	if r.b.CreatedBy != nil && r.b.CreatedBy.Id != 0 {
		// TODO: this will probably go to taking a pointer to the User
		return &userResolver{u: r.b.CreatedBy}, nil
	}

	// Looking at https://github.com/OscarYuen/go-graphql-starter/blob/f8ff416af2213ef93ef5f459904d6a403ab25843/service/user_service.go#L23
	// and https://github.com/OscarYuen/go-graphql-starter/blob/f8ff416af2213ef93ef5f459904d6a403ab25843/server.go#L20
	// I will just want to put the db in my context even though it seems like many things say do not do that.
	// Not sure I like this at all, but I also do not want to have to attach the db from resolver to every other
	// resolver/type struct I create.
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		log.Printf("No database in context")
		return nil, SERVER_ERROR
	}
	user, err := worrywort.LookupUser(int(r.b.UserId.Int64), db)
	return &userResolver{u: user}, err
}

type batchEdge struct {
	Cursor string
	Node   *batchResolver
}

func (r *batchEdge) CURSOR() string       { return r.Cursor }
func (r *batchEdge) NODE() *batchResolver { return r.Node }

// Going full relay, I suppose
// the graphql lib needs case-insensitive match of names on the methods
// so the resolver functions are just named all caps... alternately the
// struct members could be named as such to avoid a collision
// idea from https://github.com/deltaskelta/graphql-go-pets-example/blob/ab169fb644b1a00998208e7feede5975214d60da/users.go#L156
type batchConnection struct {
	// if dataloader is implemented, this could just store the ids (and do a lighter query for those ids) and use dataloader
	// to get each individual edge or sensor and build the edge in the resolver function
	Edges    *[]*batchEdge
	PageInfo *pageInfo
}

func (r *batchConnection) PAGEINFO() pageInfo   { return *r.PageInfo }
func (r *batchConnection) EDGES() *[]*batchEdge { return r.Edges }