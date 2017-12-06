package graphqlApi

import (
	graphql "github.com/neelance/graphql-go"
	"worrywort"
)

type Resolver struct{}


// TODO: example on repo would have used the user type above, but I don't think I need to.  Pretty sure that was
// because there were no other types to work with already.
type userResolver struct{
	u *worrywort.User
// 	ID() graphql.ID
// Name() string
// Friends() *[]*characterResolver
// FriendsConnection(friendsConnectionArgs) (*friendsConnectionResolver, error)
// AppearsIn() []string
}

func (r *userResolver) ID() graphql.ID {
	return graphql.ID(r.u.Id())
}

func (r *userResolver) FirstName() string {
	return r.u.FirstName()
}

func (r *userResolver) LastName() string {
	return r.u.LastName()
}

func (r *userResolver) Email() string {
	return r.u.Email()
}

// Does this need to return pointer to list of pointers?
// For now I am just rolling with the example at https://github.com/neelance/graphql-go/blob/master/example/starwars/starwars.go#L375
// and https://github.com/neelance/graphql-go/blob/master/example/starwars/starwars.go#L418
// func (r *userResolver) Batches() *[]*batchResolver {
// 	return r.u.Email()
// }

// TODO: 99% sure I can get rid of these types and just use the types from the worrywort package.
type user struct {
	ID        graphql.ID
	FirstName string
	LastName  string
	Email     string

	// I guess, based on the example on the github repo
	// but it seems like this should specify the actual type?
	Batches   []graphql.ID
}

type batch struct {
	ID        graphql.ID
	Name         string
	BrewNotes    string
	TastingNotes string

	// TODO: not sure if this will play nicely with graphql.  Probably need to see how to handle it correctly.
	BrewedDate   time.Time
	BottledDate  time.Time

	VolumeBoiled      float32
	VolumeInFermenter float32
	VolumeUnits       worrywort.VolumeUnitType

	OriginalGravity float32
	FinalGravity    float32

	// handle this as a string.  It makes nearly everything easier and can easily be run through
	// url.Parse if needed
	RecipeURL string

	CreatedAt time.Time
	UpdatedAt time.Time

	// I guess, based on the example on the github repo
	// but it seems like this should specify the actual type?
	CreatedBy graphql.ID
}

type thermometer struct {
	ID        graphql.ID
	name      string
	createdBy User

	createdAt time.Time
	updatedAt time.Time
}
