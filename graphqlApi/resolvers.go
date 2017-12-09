package graphqlApi

import (
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	graphql "github.com/neelance/graphql-go"
	"time"
	"fmt"
	"strconv"
)

// Takes a time.Time and returns nil if the time is zero or pointer to the time string formatted as RFC3339
func dateString(dt time.Time) *string {
	if dt.IsZero() {
		return nil
	}
	dtString := dt.Format(time.RFC3339)
	return &dtString
}

type Resolver struct{}

func (r *Resolver) CurrentUser() *userResolver {
	u := worrywort.NewUser(1, "jmichalicek@gmail.com", "Justin", "Michalicek", time.Now(), time.Now())
	ur := userResolver{u: u}
	fmt.Println(ur)
	return &ur
}

func (r *Resolver) Batch(args struct{ ID graphql.ID }) *batchResolver {
	brewedDate := time.Now()
	bottledDate := time.Time{} // zero time
	createdAt := time.Now()
	updatedAt := time.Now()
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	batch := worrywort.NewBatch(1, "Testing", brewedDate, bottledDate, 5, 4.5, worrywort.GALLON, 1.060, 1.020, u, createdAt, updatedAt,
		"Brew notes", "Taste notes", "http://example.org/beer")
	return &batchResolver{b: batch}
}

func (r *Resolver) Fermenter(args struct{ ID graphql.ID }) *fermenterResolver {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	f := worrywort.NewFermenter(1, "Ferm", "A Fermenter", 5.0, worrywort.GALLON, worrywort.BUCKET, true, true, u, createdAt, updatedAt)

	return &fermenterResolver{f: f}
}

func (r *Resolver) Thermometer(args struct{ ID graphql.ID }) *thermometerResolver {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	therm := worrywort.NewThermometer(1, "Therm1", u, createdAt, updatedAt)
	return &thermometerResolver{t: therm}
}

func (r *Resolver) TemperatureMeasurement(args struct{ ID graphql.ID }) *temperatureMeasurementResolver {
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	b := worrywort.NewBatch(1, "Testing", time.Now(), time.Now(), 5, 4.5, worrywort.GALLON, 1.060, 1.020, u, time.Now(), time.Now(),
		"Brew notes", "Taste notes", "http://example.org/beer")
	f := worrywort.NewFermenter(1, "Ferm", "A Fermenter", 5.0, worrywort.GALLON, worrywort.BUCKET, true, true, u, time.Now(), time.Now())
	therm := worrywort.NewThermometer(1, "Therm1", u, time.Now(), time.Now())
	createdAt := time.Now()
	updatedAt := time.Now()
	timeRecorded := time.Now()
	m := worrywort.NewTemperatureMeasurement(
		"shouldbeauuid", 64.26, worrywort.FAHRENHEIT, b, therm, f, timeRecorded, createdAt, updatedAt, u)
	return &temperatureMeasurementResolver{m: m}
}

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

func (r *userResolver) ID() graphql.ID { return graphql.ID(strconv.FormatInt(r.u.ID(), 10)) }
func (r *userResolver) FirstName() string { return r.u.FirstName() }
func (r *userResolver) LastName() string  { return r.u.LastName() }
func (r *userResolver) Email() string     { return r.u.Email() }

// TODO: I should make an actual DateTime type which can be null or a valid datetime string
func (r *userResolver) CreatedAt() *string { return dateString(r.u.CreatedAt()) }
func (r *userResolver) UpdatedAt() *string { return dateString(r.u.UpdatedAt()) }

type batchResolver struct {
	b worrywort.Batch
}

func (r *batchResolver) ID() graphql.ID { return graphql.ID(strconv.FormatInt(r.b.ID(), 10)) }
func (r *batchResolver) Name() string         { return r.b.Name() }
func (r *batchResolver) BrewNotes() string    { return r.b.BrewNotes() }
func (r *batchResolver) TastingNotes() string { return r.b.TastingNotes() }
func (r *batchResolver) BrewedDate() *string  { return dateString(r.b.BrewedDate()) }
func (r *batchResolver) BottledDate() *string { return dateString(r.b.BottledDate()) }
func (r *batchResolver) VolumeBoiled() *float64 {
	// If the value is optional/nullable in the GraphQL schema then we must return a pointer
	// to it.
	vol := r.b.VolumeBoiled()
	if vol == 0 {
		return nil
	}
	return &vol
}
func (r *batchResolver) VolumeInFermenter() *float64 {
	vol := r.b.VolumeInFermenter()
	if vol == 0 {
		return nil
	}
	return &vol
}
func (r *batchResolver) VolumeUnits() worrywort.VolumeUnitType { return r.b.VolumeUnits() }
func (r *batchResolver) OriginalGravity() *float64 {
	// TODO: not sure I like this... maybe it really was 0.  Not likely with OG, of course.
	og := r.b.OriginalGravity()
	if og == 0 {
		return nil
	}
	return &og
}
func (r *batchResolver) FinalGravity() *float64 {
	fg := r.b.FinalGravity()
	if fg == 0 {
		return nil
	}
	return &fg
}
func (r *batchResolver) RecipeURL() string  { return r.b.RecipeURL() } // this could even return a parsed URL object...
func (r *batchResolver) CreatedAt() *string { return dateString(r.b.CreatedAt()) }
func (r *batchResolver) UpdatedAt() *string { return dateString(r.b.UpdatedAt()) }

// TODO: Make this return an actual nil if there is no createdBy, such as for a deleted user?
func (r *batchResolver) CreatedBy() *userResolver { return &userResolver{u: r.b.CreatedBy()} }

type fermenterResolver struct {
	f worrywort.Fermenter
}

type thermometerResolver struct {
	t worrywort.Thermometer
}

type temperatureMeasurementResolver struct {
	m worrywort.TemperatureMeasurement
}
