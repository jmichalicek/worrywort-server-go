package graphqlApi

import (
	"context"
	// "fmt"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"log"
	// "os"
	"database/sql"
	"errors"
	"strconv"
	"time"
)

var SERVER_ERROR = errors.New("Unexpected server error.")

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
	// do not really need this now that it is coming in on context so code is inconsistent.
	// but on context is considered "not good"... I could pass this around instead, but would then
	// need to either attach a Resolver or db to every single data type, which also kind of sucks
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
	ur := userResolver{u: &u}
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
	batchArgs["user_id"] = u.Id
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
	return &batchResolver{b: batchPtr}, nil
}

func (r *Resolver) Batches(ctx context.Context) (*[]*batchResolver, error) {
	u, _ := authMiddleware.UserFromContext(ctx)
	var resolvedBatches []*batchResolver = []*batchResolver{}
	batchesPtr, err := worrywort.BatchesForUser(r.db, u, nil, nil)
	if err != nil {
		return nil, err
	}

	for idx, _ := range *batchesPtr {
		// makes sense, but not obvious to me at first:
		// the 2nd value in the range is a variable - but that variable gets reassigned a copy of
		// whatever is being iterated over, not the actual instance itself.  So rather than
		// using it, we use the index to get the real thing we are looking for
		br := batchResolver{b: &(*batchesPtr)[idx]}
		resolvedBatches = append(resolvedBatches, &br)
	}
	return &(resolvedBatches), err
}

func (r *Resolver) Fermenter(ctx context.Context, args struct{ ID graphql.ID }) (*fermenterResolver, error) {
	// authUser, _ := authMiddleware.UserFromContext(ctx)
	// TODO: Implement correctly!  Look up the Fermentor with FindFermenter
	return nil, errors.New("Not Implemented") // so that it is obvious this is no implemented
}

func (r *Resolver) TemperatureSensor(ctx context.Context, args struct{ ID graphql.ID }) (*temperatureSensorResolver, error) {
	// authUser, _ := authMiddleware.UserFromContext(ctx)
	// TODO: Implement me
	return nil, errors.New("Not Implemented") // so that it is obvious this is no implemented
}

func (r *Resolver) TemperatureMeasurement(ctx context.Context, args struct{ ID graphql.ID }) (*temperatureMeasurementResolver, error) {
	// authUser, _ := authMiddleware.UserFromContext(ctx)
	// TODO: panic on error, no user, etc.
	// TODO: REALLY IMPLEMENT THIS!
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	b := worrywort.Batch{Name: "Testing", BrewedDate: time.Now(), BottledDate: time.Now(), VolumeBoiled: 5,
		VolumeInFermenter: 4.5, VolumeUnits: worrywort.GALLON, OriginalGravity: 1.060, FinalGravity: 1.020,
		UserId: sql.NullInt64{Int64: int64(u.Id), Valid: true}, BrewNotes: "Brew notes",
		TastingNotes: "Taste notes", RecipeURL: "http://example.org/beer", CreatedBy: &u}
	f := worrywort.NewFermenter(1, "Ferm", "A Fermenter", 5.0, worrywort.GALLON, worrywort.BUCKET, true, true, u, time.Now(), time.Now())
	therm := worrywort.NewTemperatureSensor(1, "Therm1", &u, time.Now(), time.Now())
	createdAt := time.Now()
	updatedAt := time.Now()
	timeRecorded := time.Now()

	tempId := "REMOVEME"
	// TODO: This needs to save and THAT is whre the uuid should really be generated
	m := worrywort.TemperatureMeasurement{Id: tempId, Temperature: 64.26, Units: worrywort.FAHRENHEIT, RecordedAt: timeRecorded,
		Batch: &b, TemperatureSensor: &therm, Fermenter: &f, CreatedBy: &u, CreatedAt: createdAt, UpdatedAt: updatedAt}
	return &temperatureMeasurementResolver{m: &m}, nil
}

// Input types
// Create a temperatureMeasurement... review docs on how to really implement this
type createTemperatureMeasurementInput struct {
	BatchId             *graphql.ID
	RecordedAt          string //time.Time
	Temperature         float64
	TemperatureSensorId graphql.ID
	Units               string // it seems this graphql server cannot handle mapping enum to struct inputs
}

// Mutation Payloads
type createTemperatureMeasurementPayload struct {
	t *temperatureMeasurementResolver
}

func (c createTemperatureMeasurementPayload) TemperatureMeasurement() *temperatureMeasurementResolver {
	return c.t
}

// Mutations

// Create a temperature measurementId
func (r *Resolver) CreateTemperatureMeasurement(ctx context.Context, args *struct {
	Input *createTemperatureMeasurementInput
}) (*createTemperatureMeasurementPayload, error) {
	u, _ := authMiddleware.UserFromContext(ctx)
	userId := sql.NullInt64{Valid: true, Int64: int64(u.Id)}

	var inputPtr *createTemperatureMeasurementInput = args.Input
	var input createTemperatureMeasurementInput = *inputPtr
	var unitType worrywort.TemperatureUnitType

	// bleh.  Too bad this lib doesn't map the input types with enums/iota correctly
	if input.Units == "FAHRENHEIT" {
		unitType = worrywort.FAHRENHEIT
	} else {
		unitType = worrywort.CELSIUS
	}

	sensorId := ToNullInt64(string(input.TemperatureSensorId))
	tempSensorId, err := strconv.ParseInt(string(input.TemperatureSensorId), 10, 0)
	sensorPtr, err := worrywort.FindTemperatureSensor(map[string]interface{}{"id": tempSensorId, "user_id": u.Id}, r.db)
	if err != nil {
		// TODO: Probably need a friendlier error here or for our payload to have a shopify style userErrors
		// and then not ever return nil from this either way...maybe
		if err != sql.ErrNoRows {
			log.Printf("%v", err)
		}
		// TODO: only return THIS error if it really does not exist.  May need other errors
		// for other stuff
		return nil, errors.New("Specified TemperatureSensor does not exist.")
	}

	var batchPtr *worrywort.Batch = nil
	var batchId sql.NullInt64
	if input.BatchId != nil {
		batchId = ToNullInt64(string(*input.BatchId))
		batchPtr, err = worrywort.FindBatch(map[string]interface{}{"user_id": u.Id, "id": batchId}, r.db)
		if err != nil {
			if err != sql.ErrNoRows {
				log.Printf("%v", err)
			}
			// return nil, errors.New("Batch not found") ?  Need a TemperatureMeasurementCreate type for that
			// as TemperatureMeasurementCreate {userErrors: [UserError] temperatureMeasurement: TemperatureMeasurement}
			return nil, errors.New("Specified Batch does not exist.")
		}
	}
	// err becomes nil here if it was set within `if input.BatchId` stuff so we have to catch ALL of the errors in there
	// golang variable scoping I need to learn about?

	// for actual iso 8601, use "2006-01-02T15:04:05-0700"
	// TODO: test parsing both
	recordedAt, err := time.Parse(time.RFC3339, input.RecordedAt)
	if err != nil {
		// TODO: See what the actual error types are and try to return friendlier errors which are not golang specific messaging
		return nil, err
	}

	t := worrywort.TemperatureMeasurement{TemperatureSensor: sensorPtr, TemperatureSensorId: sensorId,
		Temperature: input.Temperature, Units: unitType, RecordedAt: recordedAt, CreatedBy: &u, UserId: userId,
		Batch: batchPtr, BatchId: batchId}
	t, err = worrywort.SaveTemperatureMeasurement(r.db, t)
	if err != nil {
		log.Printf("Failed to save TemperatureMeasurement: %v\n", err)
		return nil, err
	}
	tr := temperatureMeasurementResolver{m: &t}
	result := createTemperatureMeasurementPayload{t: &tr}
	return &result, nil
}

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

	err = token.Save(r.db)
	if err != nil {
		return nil, err
	}
	atr := authTokenResolver{t: token}
	return &atr, nil
}

// HELPERS - move to a different file for organization?
// ToNullInt64 validates a sql.NullInt64 if incoming string evaluates to an integer, invalidates if it does not
// Very useful for taking-y string graphql.ID values and getting a Nullint64
func ToNullInt64(s string) sql.NullInt64 {
	// Should ToNullInt64 just take a graphql.ID ?
	i, err := strconv.Atoi(s)
	return sql.NullInt64{Int64: int64(i), Valid: err == nil}
}
