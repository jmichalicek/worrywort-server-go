package graphqlApi

import (
	"context"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"log"
	// "os"
	"database/sql"
	"strconv"
	"time"
)

// Takes a time.Time and returns nil if the time is zero or pointer to the time string formatted as RFC3339
func nullableDateString(dt time.Time) *string {
	if dt.IsZero() {
		return nil
	}
	dtString := dt.Format(time.RFC3339)
	return &dtString
}

func dateString(dt time.Time) string {
	return dt.Format(time.RFC3339)
}

type Resolver struct {
	// todo: should be Db?
	db *sqlx.DB
}

func NewResolver(db *sqlx.DB) *Resolver {
	return &Resolver{db: db}
}

func (r *Resolver) CurrentUser(ctx context.Context) *userResolver {
	// This ensures we have the right type from the context
	// may change to just "authMiddleware" or something though so that
	// a single function can exist to get user from any of the auth methods
	// or just write a separate function for that here instead of using it from authMiddleware.
	// TODO: should check errors
	u, _ := authMiddleware.UserFromContext(ctx)
	ur := userResolver{u: u}
	return &ur
}

// handle errors by returning error with 403?
// func sig: func (r *Resolver) Batch(ctx context.Context, args struct{ ID graphql.ID }) (*batchResolver, error) {
func (r *Resolver) Batch(ctx context.Context, args struct{ ID graphql.ID }) (*batchResolver, error) {
	// TODO: panic on error, no user, etc.
	u, _ := authMiddleware.UserFromContext(ctx)
	var err error
	batchArgs := make(map[string]interface{})
	// TODO: Or if batch is publicly readable by anyone?
	batchArgs["created_by_user_id"] = u.Id
	batchArgs["id"], err = strconv.ParseInt(string(args.ID), 10, 0)

	if err != nil {
		log.Printf("%v", err)
		return nil, nil
	}

	batchPtr, err := worrywort.FindBatch(batchArgs, r.db)
	if err != nil {
		// do not expose sql errors
		if err != sql.ErrNoRows {
			log.Printf("%v", err)
		}
		return nil, nil
	}
	return &batchResolver{b: *batchPtr}, nil
}

func (r *Resolver) Batches(ctx context.Context) (*[]*batchResolver, error) {
	u, _ := authMiddleware.UserFromContext(ctx)
	var resolvedBatches []*batchResolver
	batchesPtr, err := worrywort.BatchesForUser(r.db, u, nil, nil)
	if err != nil {
		return nil, err
	}

	for _, batch := range *batchesPtr {
		resolvedBatches = append(resolvedBatches, &batchResolver{b: batch})
	}
	return &resolvedBatches, err
}

func (r *Resolver) Fermenter(ctx context.Context, args struct{ ID graphql.ID }) (*fermenterResolver, error) {
	// authUser, _ := authMiddleware.UserFromContext(ctx)
	// TODO: panic on error, no user, etc.

	createdAt := time.Now()
	updatedAt := time.Now()
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	f := worrywort.NewFermenter(1, "Ferm", "A Fermenter", 5.0, worrywort.GALLON, worrywort.BUCKET, true, true, u, createdAt, updatedAt)

	return &fermenterResolver{f: f}, nil
}

func (r *Resolver) TemperatureSensor(ctx context.Context, args struct{ ID graphql.ID }) (*temperatureSensorResolver, error) {
	// authUser, _ := authMiddleware.UserFromContext(ctx)
	// TODO: panic on error, no user, etc.

	createdAt := time.Now()
	updatedAt := time.Now()
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	therm := worrywort.NewTemperatureSensor(1, "Therm1", u, createdAt, updatedAt)
	return &temperatureSensorResolver{t: therm}, nil
}

func (r *Resolver) TemperatureMeasurement(ctx context.Context, args struct{ ID graphql.ID }) (*temperatureMeasurementResolver, error) {
	// authUser, _ := authMiddleware.UserFromContext(ctx)
	// TODO: panic on error, no user, etc.

	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	b := worrywort.NewBatch(1, "Testing", time.Now(), time.Now(), 5, 4.5, worrywort.GALLON, 1.060, 1.020, u, time.Now(), time.Now(),
		"Brew notes", "Taste notes", "http://example.org/beer")
	f := worrywort.NewFermenter(1, "Ferm", "A Fermenter", 5.0, worrywort.GALLON, worrywort.BUCKET, true, true, u, time.Now(), time.Now())
	therm := worrywort.NewTemperatureSensor(1, "Therm1", u, time.Now(), time.Now())
	createdAt := time.Now()
	updatedAt := time.Now()
	timeRecorded := time.Now()

	tempId := "REMOVEME"
	// TODO: This needs to save and THAT is whre the uuid should really be generated
	m := worrywort.NewTemperatureMeasurement(
		tempId, 64.26, worrywort.FAHRENHEIT, b, therm, f, timeRecorded, createdAt, updatedAt, u)
	return &temperatureMeasurementResolver{m: m}, nil
}

// TODO: example on repo would have used the user type above, but I don't think I need to.  Pretty sure that was
// because there were no other types to work with already.

// TODO: Do these resolver receivers need to receive a pointer?
type userResolver struct {
	u worrywort.User
}

func (r *userResolver) ID() graphql.ID    { return graphql.ID(strconv.Itoa(r.u.Id)) }
func (r *userResolver) FirstName() string { return r.u.FirstName }
func (r *userResolver) LastName() string  { return r.u.LastName }
func (r *userResolver) Email() string     { return r.u.Email }
func (r *userResolver) CreatedAt() string { return dateString(r.u.CreatedAt) }
func (r *userResolver) UpdatedAt() string { return dateString(r.u.UpdatedAt) }

type batchResolver struct {
	b worrywort.Batch
}

func (r *batchResolver) ID() graphql.ID       { return graphql.ID(strconv.Itoa(r.b.ID)) }
func (r *batchResolver) Name() string         { return r.b.Name }
func (r *batchResolver) BrewNotes() string    { return r.b.BrewNotes }
func (r *batchResolver) TastingNotes() string { return r.b.TastingNotes }

// TODO: I should make an actual DateTime type which can be null or a valid datetime string
func (r *batchResolver) BrewedDate() *string  { return nullableDateString(r.b.BrewedDate) }
func (r *batchResolver) BottledDate() *string { return nullableDateString(r.b.BottledDate) }
func (r *batchResolver) VolumeBoiled() *float64 {
	// If the value is optional/nullable in the GraphQL schema then we must return a pointer
	// to it.
	// TODO: I do not like this.  Maybe switch the data type to https://godoc.org/gopkg.in/guregu/null.v3 nullint
	// on the struct
	vol := r.b.VolumeBoiled
	if vol == 0 {
		return nil
	}
	return &vol
}

func (r *batchResolver) VolumeInFermenter() *float64 {
	vol := r.b.VolumeInFermenter

	// TODO: I do not like this.  Maybe switch the data type to https://godoc.org/gopkg.in/guregu/null.v3 nullint
	// on the struct
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
func (r *batchResolver) CreatedBy() *userResolver { return &userResolver{u: r.b.CreatedBy} }

// Resolve a worrywort.Fermenter
type fermenterResolver struct {
	f worrywort.Fermenter
}

func (r *fermenterResolver) ID() graphql.ID    { return graphql.ID(strconv.Itoa(r.f.ID)) }
func (r *fermenterResolver) CreatedAt() string { return dateString(r.f.CreatedAt) }
func (r *fermenterResolver) UpdatedAt() string { return dateString(r.f.UpdatedAt) }

// TODO: Make this return an actual nil if there is no createdBy, such as for a deleted user?
func (r *fermenterResolver) CreatedBy() *userResolver { return &userResolver{u: r.f.CreatedBy} }

// Resolve a worrywort.TemperatureSensor
type temperatureSensorResolver struct {
	t worrywort.TemperatureSensor
}

func (r *temperatureSensorResolver) ID() graphql.ID    { return graphql.ID(strconv.Itoa(r.t.ID)) }
func (r *temperatureSensorResolver) CreatedAt() string { return dateString(r.t.CreatedAt) }
func (r *temperatureSensorResolver) UpdatedAt() string { return dateString(r.t.UpdatedAt) }

// TODO: Make this return an actual nil if there is no createdBy, such as for a deleted user?
func (r *temperatureSensorResolver) CreatedBy() *userResolver {
	return &userResolver{u: r.t.CreatedBy}
}

// Resolve a worrywort.TemperatureMeasurement
type temperatureMeasurementResolver struct {
	// m for measurement
	m worrywort.TemperatureMeasurement
}

func (r *temperatureMeasurementResolver) ID() graphql.ID    { return graphql.ID(r.m.ID) }
func (r *temperatureMeasurementResolver) CreatedAt() string { return dateString(r.m.CreatedAt) }
func (r *temperatureMeasurementResolver) UpdatedAt() string { return dateString(r.m.UpdatedAt) }

// TODO: Make this return an actual nil if there is no createdBy, such as for a deleted user?
func (r *temperatureMeasurementResolver) CreatedBy() *userResolver {
	return &userResolver{u: r.m.CreatedBy}
}

// An auth token returned after logging in to use in Authentication headers
type authTokenResolver struct {
	t worrywort.AuthToken
	// return a status such as ok or error?
}

func (a *authTokenResolver) ID() graphql.ID { return graphql.ID(a.t.ForAuthenticationHeader()) }
func (a *authTokenResolver) Token() string  { return a.t.ForAuthenticationHeader() }

// Mutations

// TODO: Something here is not working.  It builds, but blows up.  Cannot tell for sure if it is
// due to returning an error or maybe something in middleware
func (r *Resolver) Login(args *struct {
	Username string
	Password string
}) (*authTokenResolver, error) {
	user, err := worrywort.AuthenticateLogin(args.Username, args.Password, r.db)
	// TODO: Check for errors which should not be exposed?  Or for known good errors to expose
	// and return something more generic + log if unexpected?
	if err != nil {
		return nil, err
	}

	token, err := worrywort.GenerateTokenForUser(user, worrywort.TOKEN_SCOPE_ALL)
	if err != nil {
		return nil, err
	}

	// TODO: not yet implemented, will need db
	err = token.Save(r.db)
	if err != nil {
		return nil, err
	}
	atr := authTokenResolver{t: token}
	return &atr, nil
}
