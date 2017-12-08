package graphqlApi

import (
	graphql "github.com/neelance/graphql-go"
	"time"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
)

type Resolver struct{}

// TODO: example on repo would have used the user type above, but I don't think I need to.  Pretty sure that was
// because there were no other types to work with already.

// TODO: Do these resolver receivers need to receive a pointer?
type userResolver struct {
	u worrywort.User
	// 	ID() graphql.ID
	// Name() string
	// Friends() *[]*characterResolver
	// FriendsConnection(friendsConnectionArgs) (*friendsConnectionResolver, error)
	// AppearsIn() []string
}

func (r *userResolver) ID() graphql.ID    { return graphql.ID(r.u.ID()) }
func (r *userResolver) FirstName() string { return r.u.FirstName() }
func (r *userResolver) LastName() string  { return r.u.LastName() }
func (r *userResolver) Email() string     { return r.u.Email() }
func (r *userResolver) CreatedAt() string { return r.u.CreatedAt().Format(time.RFC3339) }
func (r *userResolver) UpdatedAt() string { return r.u.UpdatedAt().Format(time.RFC3339) }

type batchResolver struct {
	b worrywort.Batch
}

func (r *batchResolver) ID() graphql.ID              { return graphql.ID(r.b.Id()) }
func (r *batchResolver) Name() string                { return r.b.Name() }
func (r *batchResolver) BrewNotes() string           { return r.b.BrewNotes() }
func (r *batchResolver) TastingNotes() string        { return r.b.TastingNotes() }
func (r *batchResolver) BrewedDate() string          { return r.b.BrewedDate().Format(time.RFC3339) }
func (r *batchResolver) BottledDate() string         { return r.b.BottledDate().Format(time.RFC3339) }
func (r *batchResolver) VolumeBoiled() float64       { return r.b.VolumeBoiled() }
func (r *batchResolver) VolumeInFermenter() float64  { return r.b.VolumeInFermenter() }
func (r *batchResolver) VolumeUnits() VolumeUnitType { return r.b.VolumeUnits() }
func (r *batchResolver) OriginalGravity() float64    { return r.b.OriginalGravity() }
func (r *batchResolver) FinalGravity() float64       { return r.b.FinalGravity() }
func (r *batchResolver) RecipeURL() string           { return r.b.RecipeURL() } // this could even return a parsed URL object...
func (r *batchResolver) CreatedAt() string           { return r.b.CreatedAt().Format(time.RFC3339) }
func (r *batchResolver) UpdatedAt() string           { return r.b.UpdatedAt().Format(time.RFC3339) }
func (r *batchResolver) CreatedBy() *UserResolver    { return userResolver{u: r.b.CreatedBy()} }
