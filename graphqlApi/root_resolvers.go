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

// This also could be handled in middleware, but then I would need two separate
// schemas and routes - one for authenticated stuff, one for
var NOT_AUTHTENTICATED_ERROR = errors.New("User must be authenticated")

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

// move these somewhere central
type pageInfo struct {
	HasNextPage     bool
	HasPreviousPage bool
}

func (r pageInfo) HASNEXTPAGE() bool     { return r.HasNextPage }
func (r pageInfo) HASPREVIOUSPAGE() bool { return r.HasPreviousPage }

type Resolver struct {
	// todo: should be Db?
	// do not really need this now that it is coming in on context so code is inconsistent.
	// but on context is considered "not good"... I could pass this around instead, but would then
	// need to either attach a Resolver or db to every single data type, which also kind of sucks
	db *sqlx.DB
}

/* This is the root resolver */
func NewResolver(db *sqlx.DB) *Resolver {
	return &Resolver{db: db}
}

func (r *Resolver) CurrentUser(ctx context.Context) (*userResolver, error) {
	// This ensures we have the right type from the context
	// may change to just "authMiddleware" or something though so that
	// a single function can exist to get user from any of the auth methods
	// or just write a separate function for that here instead of using it from authMiddleware.
	// TODO: should check errors
	u, _ := authMiddleware.UserFromContext(ctx)
	ur := userResolver{u: &u}
	return &ur, nil
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
	if batchPtr == nil {
		return nil, nil
	}
	return &batchResolver{b: batchPtr}, nil
}

func (r *Resolver) Batches(ctx context.Context, args struct {
	First *int
	After *string
}) (*batchConnection, error) {
	u, _ := authMiddleware.UserFromContext(ctx)
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, SERVER_ERROR
	}

	userIdNullInt := sql.NullInt64{Int64: int64(u.Id), Valid: true}
	batches, err := worrywort.FindBatches(map[string]interface{}{"user_id": userIdNullInt}, db)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("%v", err)
		return nil, err
	}
	edges := []*batchEdge{}
	for index, _ := range batches {
		resolvedBatch := batchResolver{b: batches[index]}
		// should base64 encode this cursor, but whatever for now
		edge := &batchEdge{Node: &resolvedBatch, Cursor: string(resolvedBatch.ID())}
		edges = append(edges, edge)
	}
	hasNextPage := false
	hasPreviousPage := false
	return &batchConnection{
		PageInfo: &pageInfo{HasNextPage: hasNextPage, HasPreviousPage: hasPreviousPage},
		Edges:    &edges}, nil
}

func (r *Resolver) Fermentor(ctx context.Context, args struct{ ID graphql.ID }) (*fermentorResolver, error) {
	// authUser, _ := authMiddleware.UserFromContext(ctx)
	// TODO: Implement correctly!  Look up the Fermentor with FindFermentor
	return nil, errors.New("Not Implemented") // so that it is obvious this is no implemented
}

func (r *Resolver) Sensor(ctx context.Context, args struct{ ID graphql.ID }) (*sensorResolver, error) {
	authUser, _ := authMiddleware.UserFromContext(ctx)
	var resolved *sensorResolver
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, SERVER_ERROR
	}

	sensorId, err := strconv.Atoi(string(args.ID))
	if err != nil {
		// not sure what could go wrong here - maybe a generic error and log the real error.
		log.Printf("%v", err)
		return nil, err
	}

	userId := sql.NullInt64{Valid: true, Int64: int64(authUser.Id)}

	sensor, err := worrywort.FindSensor(map[string]interface{}{"id": sensorId, "user_id": userId}, db)
	if err != nil {
		log.Printf("%v", err)
	} else if sensor != nil {
		resolved = &sensorResolver{s: sensor}
	}
	return resolved, err
}

func (r *Resolver) Sensors(ctx context.Context, args struct {
	First *int
	After *string
}) (*sensorConnection, error) {
	authUser, _ := authMiddleware.UserFromContext(ctx)
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, SERVER_ERROR
	}
	userId := sql.NullInt64{Valid: true, Int64: int64(authUser.Id)}
	// Now get the temperature sensors, build out the info
	sensors, err := worrywort.FindSensors(map[string]interface{}{"user_id": userId}, db)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("%v", err)
		return nil, err
	}
	edges := []*sensorEdge{}
	for index, _ := range sensors {
		sensorResolver := sensorResolver{s: sensors[index]}
		// should base64 encode this cursor, but whatever for now
		edge := &sensorEdge{Node: &sensorResolver, Cursor: string(sensorResolver.ID())}
		edges = append(edges, edge)
	}
	hasNextPage := false
	hasPreviousPage := false
	return &sensorConnection{
		PageInfo: &pageInfo{HasNextPage: hasNextPage, HasPreviousPage: hasPreviousPage},
		Edges:    &edges}, nil
}

// Returns a single resolved TemperatureMeasurement by ID, owned by the authenticated user
func (r *Resolver) TemperatureMeasurement(ctx context.Context, args struct{ ID graphql.ID }) (*temperatureMeasurementResolver, error) {
	authUser, _ := authMiddleware.UserFromContext(ctx)
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, SERVER_ERROR
	}
	var resolved *temperatureMeasurementResolver
	userId := sql.NullInt64{Valid: true, Int64: int64(authUser.Id)}
	measurementId := string(args.ID)
	measurement, err := worrywort.FindTemperatureMeasurement(
		map[string]interface{}{"id": measurementId, "user_id": userId}, db)
	if err != nil {
		log.Printf("%v", err)
	} else if measurement != nil {
		resolved = &temperatureMeasurementResolver{m: measurement}
	}
	return resolved, err
}

func (r *Resolver) TemperatureMeasurements(ctx context.Context, args struct {
	First       *int
	After       *string
	SensorId    *string
	BatchId     *string
	FermentorId *string
}) (*temperatureMeasurementConnection, error) {
	authUser, _ := authMiddleware.UserFromContext(ctx)
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, SERVER_ERROR
	}
	userId := sql.NullInt64{Valid: true, Int64: int64(authUser.Id)}

	// TODO: pagination, the rest of the optional filter params
	measurements, err := worrywort.FindTemperatureMeasurements(map[string]interface{}{"user_id": userId}, db)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("%v", err)
		return nil, err
	}
	edges := []*temperatureMeasurementEdge{}
	for index, _ := range measurements {
		measurementResolver := temperatureMeasurementResolver{m: measurements[index]}
		// should base64 encode this cursor, but whatever for now
		edge := &temperatureMeasurementEdge{Node: &measurementResolver, Cursor: string(measurementResolver.ID())}
		edges = append(edges, edge)
	}
	hasNextPage := false
	hasPreviousPage := false
	return &temperatureMeasurementConnection{
		PageInfo: &pageInfo{HasNextPage: hasNextPage, HasPreviousPage: hasPreviousPage},
		Edges:    &edges}, nil
}

// Input types
// Create a temperatureMeasurement... review docs on how to really implement this
type createTemperatureMeasurementInput struct {
	BatchId     *graphql.ID
	RecordedAt  string //time.Time
	Temperature float64
	SensorId    graphql.ID
	Units       string // it seems this graphql server cannot handle mapping enum to struct inputs
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
// TODO: move me to temperature_measurement.go ??
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

	sensorId := ToNullInt64(string(input.SensorId))
	tempSensorId, err := strconv.ParseInt(string(input.SensorId), 10, 0)
	sensorPtr, err := worrywort.FindSensor(map[string]interface{}{"id": tempSensorId, "user_id": u.Id}, r.db)
	if err != nil {
		// TODO: Probably need a friendlier error here or for our payload to have a shopify style userErrors
		// and then not ever return nil from this either way...maybe
		if err != sql.ErrNoRows {
			log.Printf("%v", err)
		}
		// TODO: only return THIS error if it really does not exist.  May need other errors
		// for other stuff
		return nil, errors.New("Specified Sensor does not exist.")
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

	t := worrywort.TemperatureMeasurement{Sensor: sensorPtr, SensorId: sensorId,
		Temperature: input.Temperature, Units: unitType, RecordedAt: recordedAt, CreatedBy: &u, UserId: userId}
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
