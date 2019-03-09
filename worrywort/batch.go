package worrywort

// Models and functions for brew batch management

import (
	"database/sql"
	// "net/url"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"
)

// Seems like these types should go in a different file for clarity, but not sure where
type VolumeUnitType int32

const (
	GALLON VolumeUnitType = iota
	QUART
)

type TemperatureUnitType int32

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
	Id                int            `db:"id"`
	CreatedBy         *User          `db:"created_by,prefix=u"` // TODO: think I will change this to User
	UserId            sql.NullInt64  `db:"user_id"`
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
	return b.Id == other.Id && b.Name == other.Name && b.BrewNotes == other.BrewNotes &&
		b.TastingNotes == other.TastingNotes && b.VolumeUnits == other.VolumeUnits &&
		b.VolumeInFermentor == other.VolumeInFermentor && b.VolumeBoiled == other.VolumeBoiled &&
		b.OriginalGravity == other.OriginalGravity && b.FinalGravity == other.FinalGravity &&
		b.RecipeURL == other.RecipeURL && b.CreatedBy.Id == other.CreatedBy.Id &&
		b.MaxTemperature == other.MaxTemperature && b.MinTemperature == other.MinTemperature &&
		b.AverageTemperature == other.AverageTemperature &&
		b.BrewedDate.Equal(other.BrewedDate) && b.BottledDate.Equal(other.BottledDate) &&
		b.CreatedAt.Equal(other.CreatedAt) //&& b.UpdatedAt().Equal(other.UpdatedAt())
}

// Find a batch by exact match of attributes
// Currently allows lookup by `id` and `user_id`
// TODO: Use fields() to iterate over the fields and use the `db`
// tag to map field name to db field.
func FindBatch(params map[string]interface{}, db *sqlx.DB) (*Batch, error) {
	// TODO: This is a dumb way to do it and I should do it like in temperature_measurement.go
	batches, err := FindBatches(params, db)
	if err == nil && len(batches) >= 1 {
		return batches[0], err
	}
	return nil, err
}

func FindBatches(params map[string]interface{}, db *sqlx.DB) ([]*Batch, error) {
	batches := []*Batch{}
	var values []interface{}
	var where []string
	for _, k := range []string{"id", "user_id"} {
		if v, ok := params[k]; ok {
			values = append(values, v)
			// TODO: Deal with values from batch OR user table
			where = append(where, fmt.Sprintf("b.%s = ?", k))
		}
	}

	selectCols := ""
	queryCols := []string{"id", "name", "brew_notes", "tasting_notes", "brewed_date", "bottled_date",
		"volume_boiled", "volume_in_fermentor", "volume_units", "original_gravity", "final_gravity", "recipe_url",
		"max_temperature", "min_temperature", "average_temperature", "created_at", "updated_at", "user_id"}
	for _, k := range queryCols {
		selectCols += fmt.Sprintf("b.%s, ", k)
	}

	q := `SELECT ` + strings.Trim(selectCols, ", ") + ` FROM batches b WHERE ` +
		strings.Join(where, " AND ")

	query := db.Rebind(q)
	err := db.Select(&batches, query, values...)

	if err != nil {
		return nil, err
	}
	return batches, nil
}

// Save the User to the database.  If User.Id() is 0
// then an insert is performed, otherwise an update on the User matching that id.
func SaveBatch(db *sqlx.DB, b Batch) (Batch, error) {
	// TODO: TEST CASE
	if b.Id != 0 {
		return UpdateBatch(db, b)
	} else {
		return InsertBatch(db, b)
	}
}

// Inserts the passed in User into the database.
// Returns a new copy of the user with any updated values set upon success.
// Returns the same, unmodified User and errors on error
func InsertBatch(db *sqlx.DB, b Batch) (Batch, error) {
	// TODO: TEST CASE
	var updatedAt time.Time
	var createdAt time.Time
	var batchId int

	query := db.Rebind(`INSERT INTO batches (user_id, name, brew_notes, tasting_notes, brewed_date, bottled_date,
		volume_boiled, volume_in_fermentor, volume_units, original_gravity, final_gravity, recipe_url, max_temperature,
		min_temperature, average_temperature, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW()) RETURNING id, created_at, updated_at`)

	err := db.QueryRow(
		query, b.UserId, b.Name, b.BrewNotes, b.TastingNotes, b.BrewedDate, b.BottledDate,
		b.VolumeBoiled, b.VolumeInFermentor, b.VolumeUnits, b.OriginalGravity, b.FinalGravity, b.RecipeURL,
		b.MaxTemperature, b.MinTemperature, b.AverageTemperature).Scan(&batchId, &createdAt, &updatedAt)
	if err != nil {
		return b, err
	}

	// TODO: Can I just assign these directly now in Scan()?
	b.Id = batchId
	b.CreatedAt = createdAt
	b.UpdatedAt = updatedAt
	return b, nil
}

// Saves the passed in user to the database using an UPDATE
// Returns a new copy of the user with any updated values set upon success.
// Returns the same, unmodified User and errors on error
func UpdateBatch(db *sqlx.DB, b Batch) (Batch, error) {
	// TODO: TEST CASE
	var updatedAt time.Time

	// TODO: Use introspection and reflection to set these rather than manually managing this?
	query := db.Rebind(`UPDATE batches SET user_id = ?, name = ?, brew_notes = ?, tasting_notes = ?,
		brewed_date = ?, bottled_date = ?, volume_boiled = ?, volume_in_fermentor = ?, volume_units = ?,
		original_gravity = ?, final_gravity = ?, recipe_url = ?, max_temperature = ?, min_temperature = ?,
		average_temperature = ?, updated_at = NOW() WHERE id = ? RETURNING updated_at`)
	err := db.QueryRow(
		query, b.UserId, b.Name, b.BrewNotes, b.TastingNotes, b.BrewedDate, b.BottledDate,
		b.VolumeBoiled, b.VolumeInFermentor, b.VolumeUnits, b.OriginalGravity, b.FinalGravity, b.RecipeURL,
		b.MaxTemperature, b.MinTemperature, b.AverageTemperature).Scan(&updatedAt)
	if err != nil {
		return b, err
	}
	b.UpdatedAt = updatedAt
	return b, nil
}

// The association between a sensor and a batch. This shows when a sensor
// was actively monitoring a specific batch in some way.
// Not sure if this should live here - it works equally well in sensor.go
// or maybe it should get its own .go file
type BatchSensor struct {
	Id string `db:"id"` // use a uuid
	BatchId         sql.NullInt64        `db:"batch_id"`
	SensorId        sql.NullInt64        `db:"sensor_id"`
	Description     string     `db:"description"`
	AssociatedAt    time.Time  `db:"associated_at"`
	DisassociatedAt *time.Time `db:"disassociated_at"`

	// TODO: Do I really want or need these here or the similar functionality on other structs?
	// what if BatchId and Batch get out of sync? Perhaps make these private and use Sensor() and Batch()
	Sensor *Sensor `db:"sensor,prefix=s"`
	Batch  *Batch `db:"batch,prefix=b"`
	// May make these all pointers - allow unset to be actually null/unset. or pq's sql.NullTime
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Creates the association between a batch and a sensor
// associatedAt is taken so that this can be created at a later date, used to fix missed associations, etc.
// TODO: Just pass batch and sensor ids? The whole struct is not needed here.
// If I pass in pointers, I can safely attach them as well...
// TODO: Why not follow same pattern as other structs with insert, save, etc.?
func AssociateBatchToSensor(batch Batch, sensor Sensor, description string, associatedAt *time.Time, db *sqlx.DB) (*BatchSensor, error) {
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

	batchId := sql.NullInt64{Int64: int64(batch.Id), Valid: true}
	sensorId := sql.NullInt64{Int64: int64(sensor.Id), Valid: true}
	query := db.Rebind(`INSERT INTO batch_sensor_association (batch_id, sensor_id, description, associated_at,
		updated_at) VALUES (?, ?, ?, ?, NOW()) RETURNING id, created_at, updated_at, associated_at`)

	// This overwrites associatedAt with the db's value because otherwise we run into precision differences on input
	// and output which gets weird when comparing
	err := db.QueryRow(
		query, batchId, sensorId, description, assocTime).Scan(&assocId, &createdAt, &updatedAt, &assocTime)

	if err != nil {
		return nil, err
	}

	bs := BatchSensor{Id: assocId, BatchId: batchId, SensorId: sensorId, Description: description,
		AssociatedAt: assocTime, UpdatedAt: updatedAt, CreatedAt: createdAt}
	return &bs, nil
}

func UpdateBatchSensorAssociation(b BatchSensor, db *sqlx.DB) (*BatchSensor, error) {
	// TODO: Tempted to make these take a BatchSensor to modify and a dict of changes... maybe. sort of elixir/ecto style.
	// TODO: not sure how I feel about taking struct, returning pointer to the struct... maybe just take the pointer?
	var updatedAt time.Time

	// TODO: Use introspection and reflection to set these rather than manually managing this?
	query := db.Rebind(`UPDATE batch_sensor_association SET batch_id = ?, sensor_id = ?, description = ?, associated_at = ?, disassociated_at = ?,
		updated_at = NOW() WHERE id = ? RETURNING updated_at`)
	err := db.QueryRow(query, b.BatchId, b.SensorId, b.Description, b.AssociatedAt, b.DisassociatedAt, b.Id).Scan(&updatedAt)
	if err != nil {
		return &b, err
	}
	b.UpdatedAt = updatedAt
	return &b, nil
}

func FindBatchSensorAssociation(params map[string]interface{}, db *sqlx.DB) (*BatchSensor, error) {
	// var association *BatchSensor = nil
	// TODO: join batch and sensor tables and pre-populate the nested batch and sensor?
	association := BatchSensor{}
	assocPtr := &association
	var values []interface{}
	var where []string

	selectCols := ""
	queryCols := []string{"id", "batch_id", "sensor_id", "description", "associated_at", "disassociated_at",
		"updated_at", "created_at"}
	for _, k := range queryCols {
		selectCols += fmt.Sprintf("ba.%s, ", k)
	}

	userId, ok := params["user_id"]
	joins := `INNER JOIN sensors s ON bsa.sensor_id = s.id `
	if ok && userId != nil {
			joins = joins + ` AND s.user_id = ? `
			values = append(values, userId)
	}
	// this seems dumb and repetitive. It works for now, though.
	joins = joins + ` INNER_JOIN batches b on bsa.batch_id = b.id`
	if ok && userId != nil {
			joins = joins + ` AND b.user_id = ? `
			values = append(values, userId)
	}

	for _, k := range []string{"batch_id", "sensor_id", "id", "disassociated_at"} {
		if v, ok := params[k]; ok {
			if v != nil {
				values = append(values, v)
				// TODO: Deal with values from batch OR user table
				where = append(where, fmt.Sprintf("ba.%s = ?", k))
			} else {
				where = append(where, fmt.Sprintf("ba.%s IS NULL ", k))
			}
		}
	}

	q := `SELECT ` + strings.Trim(selectCols, ", ") + `s.*, b.* FROM batch_sensor_association ba WHERE ` +
		strings.Join(where, " AND ")

	query := db.Rebind(q)
	err := db.Get(assocPtr, query, values...)
	if err != nil {
		// TODO: seems like I should be able to just have assoc be a nil ptr in the first place
		// then I would not need to do this.  This bit is here becaus assoc is a zero value, not nil,
		// so assocPtr is not ever nil
		assocPtr = nil
	}
	return assocPtr, err
}
