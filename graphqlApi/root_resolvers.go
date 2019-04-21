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
	"github.com/davecgh/go-spew/spew"
	"strconv"
	"time"
)

// log.SetFlags(log.LstdFlags | log.Lshortfile)
var ErrServerError = errors.New("Unexpected server error.")

// This also could be handled in middleware, but then I would need two separate
// schemas and routes - one for authenticated stuff, one for
var ErrUserNotAuthenticated = errors.New("User must be authenticated")

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
	// Lshortfile tells me too little - filename, but not which package it is in, etc.
	// Llongfile tells me too much - the full path at build from the go root. I really just need from the project root dir.
	log.SetFlags(log.LstdFlags | log.Llongfile)
	return &Resolver{db: db}
}

func (r *Resolver) CurrentUser(ctx context.Context) (*userResolver, error) {
	// This ensures we have the right type from the context
	// may change to just "authMiddleware" or something though so that
	// a single function can exist to get user from any of the auth methods
	// or just write a separate function for that here instead of using it from authMiddleware.
	// TODO: should check errors
	u, _ := authMiddleware.UserFromContext(ctx)
	if u == nil {
		return nil, ErrUserNotAuthenticated
	}
	// log.Printf("User is: %s", spew.Sdump(u))
	ur := userResolver{u: u}
	return &ur, nil
}

// handle errors by returning error with 403?
// func sig: func (r *Resolver) Batch(ctx context.Context, args struct{ ID graphql.ID }) (*batchResolver, error) {
func (r *Resolver) Batch(ctx context.Context, args struct{ ID graphql.ID }) (*batchResolver, error) {
	// TODO: panic on error, no user, etc.
	u, _ := authMiddleware.UserFromContext(ctx)
	if u == nil {
		return nil, ErrUserNotAuthenticated
	}
	var err error
	batchArgs := make(map[string]interface{})
	// TODO: Or if batch is publicly readable by anyone?
	batchArgs["user_id"] = u.Id
	batchArgs["uuid"] = args.ID
	// batchArgs["id"], err = strconv.ParseInt(string(args.ID), 10, 0)

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
	if u == nil {
		return nil, ErrUserNotAuthenticated
	}
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, ErrServerError
	}

	queryparams := map[string]interface{}{"user_id": u.Id}
	offset := 0

	if args.After != nil && *args.After != "" {
		if cursorData, err := DecodeCursor(*args.After); err == nil && cursorData.Offset != nil {
			offset = *cursorData.Offset
			queryparams["offset"] = *cursorData.Offset
		}
	}

	if args.First != nil {
		queryparams["limit"] = *args.First + 1 // +1 to easily see if there are more
	}

	batches, err := worrywort.FindBatches(queryparams, db)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("%v", err)
		return nil, err
	}

	hasNextPage := false
	hasPreviousPage := false
	edges := []*batchEdge{}
	for i, b := range batches {
		if args.First == nil || i < *args.First {
			resolvedBatch := batchResolver{b: b}
			// TODO: maybe move this bit of addition into MakeOffsetCursor?
			cursorval := offset + i + 1
			c, err := MakeOffsetCursor(cursorval)
			if err != nil {
				log.Printf("%s", err)
				return nil, ErrServerError
			}
			edge := &batchEdge{Node: &resolvedBatch, Cursor: c}
			edges = append(edges, edge)
		} else {
			hasNextPage = true
		}
	}
	return &batchConnection{
		PageInfo: &pageInfo{HasNextPage: hasNextPage, HasPreviousPage: hasPreviousPage},
		Edges:    &edges}, nil
}

func (r *Resolver) BatchSensorAssociations(ctx context.Context, args struct {
	First    *int
	After    *string
	BatchId  *string
	SensorId *string
}) (*batchSensorAssociationConnection, error) {
	u, _ := authMiddleware.UserFromContext(ctx)
	if u == nil {
		return nil, ErrUserNotAuthenticated
	}
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, ErrServerError
	}
	var offset int
	queryparams := map[string]interface{}{"user_id": u.Id}

	if args.After != nil && *args.After != "" {
		if cursorData, err := DecodeCursor(*args.After); err == nil && cursorData.Offset != nil {
			offset = *cursorData.Offset
			queryparams["offset"] = *cursorData.Offset
		}
	}

	if args.First != nil {
		queryparams["limit"] = *args.First + 1 // +1 to easily see if there are more
	}

	if args.BatchId != nil {
		queryparams["batch_uuid"] = *args.BatchId
	}

	if args.SensorId != nil {
		queryparams["sensor_uuid"] = args.SensorId
	}

	associations, err := worrywort.FindBatchSensorAssociations(queryparams, db)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("%v", err)
		return nil, err
	}

	hasNextPage := false
	hasPreviousPage := false
	edges := []*batchSensorAssociationEdge{}
	for i, assoc := range associations {
		if args.First == nil || i < *args.First {
			resolved := batchSensorAssociationResolver{assoc: assoc}
			// TODO: Not 100% sure about this. We have current offset + current index + 1 where the extra 1
			// is added so that the offset value in the cursor will be to start at the NEXT item, which feels odd
			// since the param used is "After". This could optionally add the 1 to the incoming data
			// which might feel more natural
			cursorval := offset + i + 1
			c, err := MakeOffsetCursor(cursorval)
			if err != nil {
				log.Printf("%s", err)
				return nil, ErrServerError
			}
			edge := &batchSensorAssociationEdge{Node: &resolved, Cursor: c}
			edges = append(edges, edge)
		} else {
			// had one more than was actually requested, there is a next page
			hasNextPage = true
		}
	}
	return &batchSensorAssociationConnection{
		PageInfo: &pageInfo{HasNextPage: hasNextPage, HasPreviousPage: hasPreviousPage},
		Edges:    &edges}, nil
}

func (r *Resolver) Fermentor(ctx context.Context, args struct{ ID graphql.ID }) (*fermentorResolver, error) {
	// authUser, _ := authMiddleware.UserFromContext(ctx)
	// TODO: Implement correctly!  Look up the Fermentor with FindFermentor
	return nil, errors.New("Not Implemented") // so that it is obvious this is no implemented
}

func (r *Resolver) Sensor(ctx context.Context, args struct{ ID graphql.ID }) (*sensorResolver, error) {
	user, _ := authMiddleware.UserFromContext(ctx)
	if user == nil {
		return nil, ErrUserNotAuthenticated
	}
	var resolved *sensorResolver
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, ErrServerError
	}

	sensor, err := worrywort.FindSensor(map[string]interface{}{"uuid": string(args.ID), "user_id": *user.Id}, db)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("%v", err)
		}
		return nil, nil // maybe error should be returned
	} else if sensor != nil && sensor.UUID != "" {
		// TODO: check for UUID is a hack because I need to rework FindSensor to return nil
		// if it did not find a sensor
		resolved = &sensorResolver{s: sensor}
	} else {
		resolved = nil
	}
	return resolved, err
}

func (r *Resolver) Sensors(ctx context.Context, args struct {
	First *int
	After *string
}) (*sensorConnection, error) {
	u, _ := authMiddleware.UserFromContext(ctx)
	if u == nil {
		return nil, ErrUserNotAuthenticated
	}
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context: %s", spew.Sdump(ctx))
		return nil, ErrServerError
	}

	// TODO: Can I de-duplicate some of this and put it in a reusable function?
	var offset int
	queryparams := map[string]interface{}{"user_id": u.Id}

	if args.After != nil && *args.After != "" {
		if cursorData, err := DecodeCursor(*args.After); err == nil && cursorData.Offset != nil {
			offset = *cursorData.Offset
			queryparams["offset"] = *cursorData.Offset
		}
	}

	if args.First != nil {
		queryparams["limit"] = *args.First + 1 // +1 to easily see if there are more
	}

	// Now get the temperature sensors, build out the info
	sensors, err := worrywort.FindSensors(queryparams, db)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("%v", err)
		return nil, err
	}

	hasNextPage := false
	hasPreviousPage := false
	edges := []*sensorEdge{}

	for i, s := range sensors {
		if args.First == nil || i < *args.First {
			cursorval := offset + i + 1
			c, err := MakeOffsetCursor(cursorval)
			if err != nil {
				log.Printf("%s", err)
				return nil, ErrServerError
			}
			sensorResolver := sensorResolver{s: s}
			edge := &sensorEdge{Node: &sensorResolver, Cursor: c}
			edges = append(edges, edge)
		} else {
			// had one more than was actually requested, there is a next page
			hasNextPage = true
		}
	}
	return &sensorConnection{
		PageInfo: &pageInfo{HasNextPage: hasNextPage, HasPreviousPage: hasPreviousPage},
		Edges:    &edges}, nil

	// for index, _ := range sensors {
	// 	sensorResolver := sensorResolver{s: sensors[index]}
	// 	// should base64 encode this cursor, but whatever for now
	// 	edge := &sensorEdge{Node: &sensorResolver, Cursor: string(sensorResolver.ID())}
	// 	edges = append(edges, edge)
	// }
	// hasNextPage := false
	// hasPreviousPage := false
	// return &sensorConnection{
	// 	PageInfo: &pageInfo{HasNextPage: hasNextPage, HasPreviousPage: hasPreviousPage},
	// 	Edges:    &edges}, nil
}

// Returns a single resolved TemperatureMeasurement by ID, owned by the authenticated user
func (r *Resolver) TemperatureMeasurement(ctx context.Context, args struct{ ID graphql.ID }) (*temperatureMeasurementResolver, error) {
	authUser, _ := authMiddleware.UserFromContext(ctx)
	if authUser == nil {
		return nil, ErrUserNotAuthenticated
	}
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, ErrServerError
	}
	var resolved *temperatureMeasurementResolver
	measurementId := string(args.ID)
	measurement, err := worrywort.FindTemperatureMeasurement(
		map[string]interface{}{"id": measurementId, "user_id": *authUser.Id}, db)
	if err != nil {
		log.Printf("%v", err)
	} else if measurement != nil {
		resolved = &temperatureMeasurementResolver{m: measurement}
	}
	return resolved, err
}

func (r *Resolver) TemperatureMeasurements(ctx context.Context, args struct {
	First    *int
	After    *string
	SensorId *string
	BatchId  *string
}) (*temperatureMeasurementConnection, error) {
	authUser, _ := authMiddleware.UserFromContext(ctx)
	if authUser == nil {
		return nil, ErrUserNotAuthenticated
	}
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, ErrServerError
	}

	queryparams := map[string]interface{}{"user_id": *authUser.Id}
	offset := 0 // TODO: implement correct offset in return values.

	if args.After != nil && *args.After != "" {
		if cursorData, err := DecodeCursor(*args.After); err == nil && cursorData.Offset != nil {
			offset = *cursorData.Offset
			queryparams["offset"] = *cursorData.Offset
		}
	}

	if args.First != nil {
		queryparams["limit"] = *args.First + 1 // +1 to easily see if there are more
	}

	if args.BatchId != nil {
		queryparams["batch_uuid"] = *args.BatchId
	}

	if args.SensorId != nil {
		queryparams["sensor_uuid"] = args.SensorId
	}

	// TODO: pagination, the rest of the optional filter params
	measurements, err := worrywort.FindTemperatureMeasurements(queryparams, db)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("%v", err)
		return nil, err
	}
	// edges := []*temperatureMeasurementEdge{}
	// for index, _ := range measurements {
	// 	measurementResolver := temperatureMeasurementResolver{m: measurements[index]}
	// 	// should base64 encode this cursor, but whatever for now
	// 	edge := &temperatureMeasurementEdge{Node: &measurementResolver, Cursor: string(measurementResolver.ID())}
	// 	edges = append(edges, edge)
	// }
	// hasNextPage := false
	// hasPreviousPage := false

	edges := []*temperatureMeasurementEdge{}
	hasNextPage := false
	hasPreviousPage := false
	for i, m := range measurements {
		if args.First == nil || i < *args.First {
			resolved := temperatureMeasurementResolver{m: m}
			// TODO: maybe move this bit of addition into MakeOffsetCursor?
			cursorval := offset + i + 1
			c, err := MakeOffsetCursor(cursorval)
			if err != nil {
				log.Printf("%s", err)
				return nil, ErrServerError
			}
			edge := &temperatureMeasurementEdge{Node: &resolved, Cursor: c}
			edges = append(edges, edge)
		} else {
			hasNextPage = true
		}
	}
	return &temperatureMeasurementConnection{
		PageInfo: &pageInfo{HasNextPage: hasNextPage, HasPreviousPage: hasPreviousPage},
		Edges:    &edges}, nil
}

// Input types
// Create a temperatureMeasurement... review docs on how to really implement this
type createTemperatureMeasurementInput struct {
	RecordedAt  string //time.Time
	Temperature float64
	SensorId    graphql.ID
	Units       string // it seems this graphql server cannot handle mapping enum to struct inputs
}

type createSensorInput struct {
	Name string
}

// Mutation Payloads
type createTemperatureMeasurementPayload struct {
	t *temperatureMeasurementResolver
}

func (c createTemperatureMeasurementPayload) TemperatureMeasurement() *temperatureMeasurementResolver {
	return c.t
}

type createSensorPayload struct {
	s *sensorResolver
}

func (c createSensorPayload) Sensor() *sensorResolver {
	return c.s
}

// Mutations

// Create a temperature measurementId
// TODO: move me to temperature_measurement.go ??
func (r *Resolver) CreateTemperatureMeasurement(ctx context.Context, args *struct {
	Input *createTemperatureMeasurementInput
}) (*createTemperatureMeasurementPayload, error) {
	u, _ := authMiddleware.UserFromContext(ctx)
	if u == nil {
		return nil, ErrUserNotAuthenticated
	}

	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, ErrServerError
	}

	var inputPtr *createTemperatureMeasurementInput = args.Input
	// TODO: make sure input was not nil. Technically the schema does this for us
	// but might be safer to handle here, too, or at least have a test case for it.
	var input createTemperatureMeasurementInput = *inputPtr
	var unitType worrywort.TemperatureUnitType

	// bleh.  Too bad this lib doesn't map the input types with enums/iota correctly
	if input.Units == "FAHRENHEIT" {
		unitType = worrywort.FAHRENHEIT
	} else {
		unitType = worrywort.CELSIUS
	}

	sensorId, err := strconv.ParseInt(string(input.SensorId), 10, 32)
	sensorPtr, err := worrywort.FindSensor(map[string]interface{}{"id": sensorId, "user_id": u.Id}, db)
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

	// for actual iso 8601, use "2006-01-02T15:04:05-0700"
	// TODO: test parsing both
	recordedAt, err := time.Parse(time.RFC3339, input.RecordedAt)
	if err != nil {
		// TODO: See what the actual error types are and try to return friendlier errors which are not golang specific messaging
		return nil, err
	}

	t := worrywort.TemperatureMeasurement{Sensor: sensorPtr, SensorId: sensorPtr.Id,
		Temperature: input.Temperature, Units: unitType, RecordedAt: recordedAt, CreatedBy: u, UserId: u.Id}
	if err := t.Save(db); err != nil {
		log.Printf("Failed to save TemperatureMeasurement: %v\n", err)
		return nil, err
	}
	tr := temperatureMeasurementResolver{m: &t}
	result := createTemperatureMeasurementPayload{t: &tr}
	return &result, nil
}

func (r *Resolver) CreateSensor(ctx context.Context, args *struct {
	Input *createSensorInput
}) (*createSensorPayload, error) {
	u, _ := authMiddleware.UserFromContext(ctx)
	if u == nil {
		return nil, ErrUserNotAuthenticated
	}

	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		// TODO: logging with stack info?
		log.Printf("No database in context")
		return nil, ErrServerError
	}

	var inputPtr *createSensorInput = args.Input
	// TODO: make sure input was not nil. Technically the schema does this for us
	// but might be safer to handle here, too, or at least have a test case for it.
	var input createSensorInput = *inputPtr

	s := worrywort.Sensor{Name: input.Name, CreatedBy: u, UserId: u.Id}
	if err := s.Save(db); err != nil {
		log.Printf("Failed to save Sensor: %v\n", err)
		return nil, err
	}
	sr := sensorResolver{s: &s}
	result := createSensorPayload{s: &sr}
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

	token, err := worrywort.GenerateTokenForUser(*user, worrywort.TOKEN_SCOPE_ALL)
	if err != nil {
		log.Printf("*****ERRR*****\n%v\n", err)
		return nil, err
	}
	tokenPtr := &token

	err = tokenPtr.Save(r.db)
	if err != nil {
		log.Printf("%s", err)
		return nil, err
	}
	atr := authTokenResolver{t: token}
	return &atr, err
}
