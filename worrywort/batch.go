package worrywort

// Models and functions for brew batch management

import (
	"errors"
	"fmt"
	// "github.com/davecgh/go-spew/spew"
	"github.com/elgris/sqrl"
	"github.com/jmoiron/sqlx"
	// "strings"
	"time"
)

// Seems like these types should go in a different file for clarity, but not sure where
type VolumeUnitType int64

const (
	GALLON VolumeUnitType = iota
	QUART
)

type TemperatureUnitType int64

const (
	FAHRENHEIT TemperatureUnitType = iota
	CELSIUS
)

// TODO: Is this a good idea?  An error for invalid values on functions which take
// a map[string]interface{}
// This could be its own error type with field name, field value, and an Error() which
// formats nicely...
var TypeError error = errors.New("Invalid type specified")

type Batch struct {
	Id                *int64         `db:"id"`
	UUID              string         `db:"uuid"`
	CreatedBy         *User          `db:"created_by,prefix=u"` // TODO: think I will change this to User
	UserId            *int64         `db:"user_id"`
	Name              string         `db:"name"`
	BrewNotes         string         `db:"brew_notes"`
	TastingNotes      string         `db:"tasting_notes"`
	BrewedDate        time.Time      `db:"brewed_date"`
	BottledDate       time.Time      `db:"bottled_date"`
	VolumeBoiled      float64        `db:"volume_boiled"` // sql nullfloats?
	VolumeInFermentor float64        `db:"volume_in_fermentor"`
	VolumeUnits       VolumeUnitType `db:"volume_units"`
	// TODO: Volume bottled?

	// TODO: gravity in other units?  Brix, etc?
	OriginalGravity float64 `db:"original_gravity"`
	FinalGravity    float64 `db:"final_gravity"` // TODO: sql.nullfloat64?
	// TODO: this stuff
	// Should any of these temperatures really be here? they can be queried/calculated from the db already...
	// although maybe should be a property but NOT a db field...
	// and need the units... C or F.  Maybe should just always do F and then convert.
	MaxTemperature     float64 `db:"max_temperature"`
	MinTemperature     float64 `db:"min_temperature"`
	AverageTemperature float64 `db:"average_temperature"`
	// handle this as a string.  It makes nearly everything easier and can easily be run through
	// url.Parse if needed
	RecipeURL string `db:"recipe_url"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Returns a list of the db columns to use for a SELECT query
func (b Batch) queryColumns() []string {
	// TODO: Way to dynamically build this using the `db` tag and reflection/introspection
	return []string{"id", "name", "brew_notes", "tasting_notes", "brewed_date", "bottled_date",
		"volume_boiled", "volume_in_fermentor", "volume_units", "original_gravity", "final_gravity", "recipe_url",
		"max_temperature", "min_temperature", "average_temperature", "created_at", "updated_at", "user_id"}
}

// Performs a comparison of all attributes of the Batches.  Related structs have only their Id compared.
// may rename to just Equal() or that may be used for a simpler Id only type comparison, but that is easy to
// compare anyway.
func (b Batch) StrictEqual(other Batch) bool {
	// TODO: do not follow the object for CreatedBy() to get id, but will need to add a CreatedById() to
	// the batch struct
	// TODO: Use go-cmp for this?
	return *b.Id == *other.Id && b.Name == other.Name && b.BrewNotes == other.BrewNotes &&
		b.TastingNotes == other.TastingNotes && b.VolumeUnits == other.VolumeUnits &&
		b.VolumeInFermentor == other.VolumeInFermentor && b.VolumeBoiled == other.VolumeBoiled &&
		b.OriginalGravity == other.OriginalGravity && b.FinalGravity == other.FinalGravity &&
		b.RecipeURL == other.RecipeURL && b.CreatedBy.Id == other.CreatedBy.Id &&
		b.MaxTemperature == other.MaxTemperature && b.MinTemperature == other.MinTemperature &&
		b.AverageTemperature == other.AverageTemperature &&
		b.BrewedDate.Equal(other.BrewedDate) && b.BottledDate.Equal(other.BottledDate) &&
		b.CreatedAt.Equal(other.CreatedAt) //&& b.UpdatedAt().Equal(other.UpdatedAt())
}

// I wonder if this can be further meged in with buildTemperatureMeasuremensQuery
// and does it need to return the []interface{} for values?
func buildBatchesQuery(params map[string]interface{}, db *sqlx.DB) *sqrl.SelectBuilder {
	query := sqrl.Select().From("batches b")
	for _, k := range []string{"id", "user_id", "uuid"} {
		// TODO: return error if not ok?
		if v, ok := params[k]; ok {
			query = query.Where(sqrl.Eq{fmt.Sprintf("b.%s", k): v})
		}
	}

	// TODO: JOIN THE USER HERE!!!
	// probably more efficient to use Columns() here but I am planning on moving all of the columns names somewhere
	// more central for easier management across querying in multiple places.
	queryCols := []string{"id", "name", "brew_notes", "tasting_notes", "brewed_date", "bottled_date",
		"volume_boiled", "volume_in_fermentor", "volume_units", "original_gravity", "final_gravity", "recipe_url",
		"max_temperature", "min_temperature", "average_temperature", "created_at", "updated_at", "user_id", "uuid"}
	for _, k := range queryCols {
		query = query.Column(fmt.Sprintf("b.%s", k))
	}

	if v, ok := params["limit"]; ok {
		query = query.Limit(uint64(v.(int)))
	}
	if v, ok := params["offset"]; ok {
		query = query.Offset(uint64(v.(int)))
	}

	return query
}

// Find a batch by exact match of attributes
// Currently allows lookup by `id` and `user_id`
// TODO: Use fields() to iterate over the fields and use the `db`
// tag to map field name to db field.
func FindBatch(params map[string]interface{}, db *sqlx.DB) (*Batch, error) {
	batch := new(Batch)
	query, values, err := buildBatchesQuery(params, db).ToSql()
	if err == nil {
		err = db.Get(batch, db.Rebind(query), values...)
	}
	return batch, err
}

func FindBatches(params map[string]interface{}, db *sqlx.DB) ([]*Batch, error) {
	batches := new([]*Batch)
	query, values, err := buildBatchesQuery(params, db).ToSql()
	if err == nil {
		err = db.Select(batches, db.Rebind(query), values...)
	}
	return *batches, err
}

// Save the Batch to the database.  If User.Id() is 0
// then an insert is performed, otherwise an update on the User matching that id.
func (b *Batch) Save(db *sqlx.DB) error {
	if b.Id == nil || *b.Id == 0 {
		return InsertBatch(db, b)
	} else {
		return UpdateBatch(db, b)
	}
}

// Inserts the passed in User into the database.
// Returns a new copy of the user with any updated values set upon success.
// Returns the same, unmodified User and errors on error
func InsertBatch(db *sqlx.DB, b *Batch) error {
	// TODO: TEST CASE
	var updatedAt time.Time
	var createdAt time.Time
	batchId := new(int64)
	batchUUID := new(string)

	// TODO: use sqrl
	query := db.Rebind(`INSERT INTO batches (user_id, name, brew_notes, tasting_notes, brewed_date, bottled_date,
		volume_boiled, volume_in_fermentor, volume_units, original_gravity, final_gravity, recipe_url, max_temperature,
		min_temperature, average_temperature, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, (NOW() at time zone 'utc'), (NOW() at time zone 'utc')) RETURNING id, created_at, updated_at, uuid`)

	err := db.QueryRow(
		query, b.UserId, b.Name, b.BrewNotes, b.TastingNotes, b.BrewedDate, b.BottledDate,
		b.VolumeBoiled, b.VolumeInFermentor, b.VolumeUnits, b.OriginalGravity, b.FinalGravity, b.RecipeURL,
		b.MaxTemperature, b.MinTemperature, b.AverageTemperature).Scan(batchId, &createdAt, &updatedAt, batchUUID)

	if err == nil {
		// TODO: double check to verify we get utc updated_at and created_at both this way and if just using "NOW()"
		// TODO: Can I just assign these directly now in Scan()?
		b.Id = batchId
		b.CreatedAt = createdAt
		b.UpdatedAt = updatedAt
		b.UUID = *batchUUID
	}
	return err
}

// Saves the passed in user to the database using an UPDATE
// Returns a new copy of the user with any updated values set upon success.
// Returns the same, unmodified User and errors on error
func UpdateBatch(db *sqlx.DB, b *Batch) error {
	// TODO: TEST CASE
	var updatedAt time.Time

	// TODO: Use introspection and reflection to set these rather than manually managing this?
	// TODO: use sqrl
	query := db.Rebind(`UPDATE batches SET user_id = ?, name = ?, brew_notes = ?, tasting_notes = ?,
		brewed_date = ?, bottled_date = ?, volume_boiled = ?, volume_in_fermentor = ?, volume_units = ?,
		original_gravity = ?, final_gravity = ?, recipe_url = ?, max_temperature = ?, min_temperature = ?,
		average_temperature = ?, updated_at = (NOW() at time zone 'utc') WHERE id = ? RETURNING updated_at`)
	err := db.QueryRow(
		query, b.UserId, b.Name, b.BrewNotes, b.TastingNotes, b.BrewedDate, b.BottledDate,
		b.VolumeBoiled, b.VolumeInFermentor, b.VolumeUnits, b.OriginalGravity, b.FinalGravity, b.RecipeURL,
		b.MaxTemperature, b.MinTemperature, b.AverageTemperature, b.Id).Scan(&updatedAt)

	if err == nil {
		b.UpdatedAt = updatedAt
	}
	return err
}

// The association between a sensor and a batch. This shows when a sensor
// was actively monitoring a specific batch in some way.
// Not sure if this should live here - it works equally well in sensor.go
// or maybe it should get its own .go file
type BatchSensor struct {
	Id              string     `db:"id"` // use a uuid?. TODO: Make this null/pointer as well
	BatchId         *int64     `db:"batch_id"`
	SensorId        *int64     `db:"sensor_id"`
	Description     string     `db:"description"`
	AssociatedAt    time.Time  `db:"associated_at"`
	DisassociatedAt *time.Time `db:"disassociated_at"`

	// TODO: Do I really want or need these here or the similar functionality on other structs?
	// what if BatchId and Batch get out of sync? Perhaps make these private and use Sensor() and Batch()
	Sensor *Sensor `db:"s,prefix=s"`
	Batch  *Batch  `db:"b,prefix=b"`
	// May make these all pointers - allow unset to be actually null/unset. or pq's sql.NullTime
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Creates the association between a batch and a sensor
// associatedAt is taken so that this can be created at a later date, used to fix missed associations, etc.
// TODO: Just pass batch and sensor ids? The whole struct is not needed here.
// If I pass in pointers, I can safely attach them as well...
// TODO: Why not follow same pattern as other structs with insert, save, etc.?
func AssociateBatchToSensor(batch *Batch, sensor *Sensor, description string, associatedAt *time.Time, db *sqlx.DB) (*BatchSensor, error) {
	var updatedAt time.Time
	var createdAt time.Time
	var assocTime time.Time
	assocId := ""
	if associatedAt == nil {
		// do not modify the associatedAt pointer - we just want to allow it to be nil
		// so a new var is made here and this is what will get assigned to.  sqlx also does not deal with
		// associatedAt being passed in as nil and just using that in db.QueryRow, so this manually deals with that.
		assocTime = time.Now()
	} else {
		assocTime = *associatedAt
	}

	// TODO: This should work, but I am getting errors back about sql.Row has no StructScan.  Why is it a sql.Row and not
	// a sqlx.Row?
	// var bs *BatchSensor
	// query := db.Rebind(`INSERT INTO batch_sensor_association (batch_id, sensor_id, description, associated_at,
	// 	created_at, updated_at) VALUES (?, ?, ?, ?, NOW(), NOW()) RETURNING batch_id, sensor_id, description, associated_at, disassociated_at, created_at, updated_at`)
	//
	// err := db.QueryRow(
	// 	query, batch.Id, sensor.Id, description, associatedAt).StructScan(bs)

	// TODO: use sqrl
	query := db.Rebind(`INSERT INTO batch_sensor_association (batch_id, sensor_id, description, associated_at,
		updated_at) VALUES (?, ?, ?, ?, NOW()) RETURNING id, created_at, updated_at, associated_at`)

	// This overwrites associatedAt with the db's value because otherwise we run into precision differences on input
	// and output which gets weird when comparing
	err := db.QueryRow(
		query, batch.Id, sensor.Id, description, assocTime).Scan(&assocId, &createdAt, &updatedAt, &assocTime)

	if err != nil {
		return nil, err
	}

	// TODO: attach the batch and sensor which were passed in
	bs := BatchSensor{Id: assocId, BatchId: batch.Id, SensorId: sensor.Id, Description: description,
		AssociatedAt: assocTime, UpdatedAt: updatedAt, CreatedAt: createdAt, Batch: batch, Sensor: sensor}
	return &bs, nil
}

func UpdateBatchSensorAssociation(b BatchSensor, db *sqlx.DB) (*BatchSensor, error) {
	// TODO: Tempted to make these take a BatchSensor to modify and a dict of changes... maybe. sort of elixir/ecto style.
	// TODO: not sure how I feel about taking struct, returning pointer to the struct... maybe just take the pointer?
	var updatedAt time.Time

	// TODO: Use introspection and reflection to set these rather than manually managing this?
	// TODO: use sqrl
	query := db.Rebind(`UPDATE batch_sensor_association SET batch_id = ?, sensor_id = ?, description = ?, associated_at = ?, disassociated_at = ?,
		updated_at = NOW() WHERE id = ? RETURNING updated_at`)
	err := db.QueryRow(query, b.BatchId, b.SensorId, b.Description, b.AssociatedAt, b.DisassociatedAt, b.Id).Scan(&updatedAt)
	if err != nil {
		return &b, err
	}
	b.UpdatedAt = updatedAt
	return &b, nil
}

// Build up the query for BatchSensorAssociations
func buildBatchSensorAssociationsQuery(params map[string]interface{}, db *sqlx.DB) *sqrl.SelectBuilder {
	query := sqrl.Select().From("batch_sensor_association ba")
	// probably more efficient to use Columns() here but I am planning on moving all of the columns names somewhere
	// more central for easier management across querying in multiple places.
	queryCols := []string{"id", "batch_id", "sensor_id", "description", "associated_at", "disassociated_at",
		"updated_at", "created_at"}
	for _, k := range queryCols {
		query = query.Column(fmt.Sprintf("ba.%s", k))
	}

	batchQueryCols := []string{"id", "name", "brew_notes", "tasting_notes", "brewed_date", "bottled_date",
		"volume_boiled", "volume_in_fermentor", "volume_units", "original_gravity", "final_gravity", "recipe_url",
		"max_temperature", "min_temperature", "average_temperature", "created_at", "updated_at", "user_id", "uuid"}
	for _, k := range batchQueryCols {
		query = query.Column(fmt.Sprintf("b.%s AS \"b.%s\"", k, k))

	}

	sensorQueryCols := []string{"id", "name", "created_at", "updated_at", "user_id", "uuid"}
	for _, k := range sensorQueryCols {
		query = query.Column(fmt.Sprintf("s.%s AS \"s.%s\"", k, k))
	}

	// TODO: Join createdBy for sensors and batches and populate that stuff as well?
	// maybe have a param for joins or for how deep to join?
	query = query.Join("sensors s ON ba.sensor_id = s.id") // TODO: handle the WHERE
	query = query.Join("batches b ON ba.batch_id = b.id")  // TODO: handle the WHERE

	for _, k := range []string{"batch_id", "sensor_id", "id", "disassociated_at"} {
		if v, ok := params[k]; ok {
			// this even handles nil/IS NULL
			query = query.Where(sqrl.Eq{fmt.Sprintf("ba.%s", k): v})
		}
	}

	if v, ok := params["batch_uuid"]; ok {
		query = query.Where(sqrl.Eq{"b.uuid": v})
	}

	if v, ok := params["sensor_uuid"]; ok {
		query = query.Where(sqrl.Eq{"s.uuid": v})
	}

	if userId, ok := params["user_id"]; ok {
		query = query.Where(sqrl.Eq{"b.user_id": userId})
		query = query.Where(sqrl.Eq{"s.user_id": userId})
	}

	if v, ok := params["limit"]; ok {
		query = query.Limit(uint64(v.(int)))
	}
	if v, ok := params["offset"]; ok {
		query = query.Offset(uint64(v.(int)))
	}

	return query
}

func FindBatchSensorAssociation(params map[string]interface{}, db *sqlx.DB) (*BatchSensor, error) {
	association := new(BatchSensor)
	query, values, err := buildBatchSensorAssociationsQuery(params, db).ToSql()
	if err == nil {
		err = db.Get(association, db.Rebind(query), values...)
	}
	return association, err
}

func FindBatchSensorAssociations(params map[string]interface{}, db *sqlx.DB) ([]*BatchSensor, error) {
	associations := new([]*BatchSensor)
	query, values, err := buildBatchSensorAssociationsQuery(params, db).ToSql()
	if err == nil {
		err = db.Select(associations, db.Rebind(query), values...)
	}
	return *associations, err
}
