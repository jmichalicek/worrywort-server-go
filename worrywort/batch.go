package worrywort

// Models and functions for brew batch management

import (
	// "net/url"
	"fmt"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"
)

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

type FermenterStyleType int32

const (
	BUCKET FermenterStyleType = iota
	CARBOY
	CONICAL
)

// Should these be exportable if I am going to use factory methods?  NewBatch() etc?
// as long as I provide a Batcher interface or whatever?
type batch struct {
	ID                 int            `db:"id"`
	CreatedBy          User           `db:"created_by,prefix=u"`
	Name               string         `db:"name"`
	BrewNotes          string         `db:"brew_notes"`
	TastingNotes       string         `db:"tasting_notes"`
	BrewedDate         time.Time      `db:"brewed_date"`
	BottledDate        time.Time      `db:"bottled_date"`
	VolumeBoiled       float64        `db:"volume_boiled"`
	VolumeInFermenter  float64        `db:"volume_in_fermenter"`
	VolumeUnits        VolumeUnitType `db:"volume_units"`
	OriginalGravity    float64        `db:"original_gravity"`
	FinalGravity       float64        `db:"final_gravity"`
	MaxTemperature     float64        `db:"max_temperature"`
	MinTemperature     float64        `db:"min_temperature"`
	AverageTemperature float64        `db:"average_temperature"`
	// handle this as a string.  It makes nearly everything easier and can easily be run through
	// url.Parse if needed
	RecipeURL string `db:"recipe_url"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Returns a list of the db columns to use for a SELECT query
func (b batch) queryColumns() []string {
	// TODO: Way to dynamically build this using the `db` tag and reflection/introspection
	return []string{"id", "name", "brew_notes", "tasting_notes", "brewed_date", "bottled_date",
		"volume_boiled", "volume_in_fermenter", "volume_units", "original_gravity", "final_gravity", "recipe_url",
		"max_temperature", "min_temperature", "average_temperature", "created_at", "updated_at"}
}

type Batch struct {
	batch
}

func (b Batch) ID() int                     { return b.batch.ID }
func (b Batch) Name() string                { return b.batch.Name }
func (b Batch) BrewNotes() string           { return b.batch.BrewNotes }
func (b Batch) TastingNotes() string        { return b.batch.TastingNotes }
func (b Batch) BrewedDate() time.Time       { return b.batch.BrewedDate }
func (b Batch) BottledDate() time.Time      { return b.batch.BottledDate }
func (b Batch) VolumeBoiled() float64       { return b.batch.VolumeBoiled }
func (b Batch) VolumeInFermenter() float64  { return b.batch.VolumeInFermenter }
func (b Batch) VolumeUnits() VolumeUnitType { return b.batch.VolumeUnits }
func (b Batch) OriginalGravity() float64    { return b.batch.OriginalGravity }
func (b Batch) FinalGravity() float64       { return b.batch.FinalGravity }
func (b Batch) RecipeURL() string           { return b.batch.RecipeURL } // this could even return a parsed URL object...
func (b Batch) CreatedAt() time.Time        { return b.batch.CreatedAt }
func (b Batch) UpdatedAt() time.Time        { return b.batch.UpdatedAt }
func (b Batch) CreatedBy() User             { return b.batch.CreatedBy }
func (b Batch) MaxTemperature() float64     { return b.batch.MaxTemperature }
func (b Batch) MinTemperature() float64     { return b.batch.MinTemperature }
func (b Batch) AverageTemperature() float64 { return b.batch.AverageTemperature }

// Performs a comparison of all attributes of the Batches.  Related structs have only their ID() compared.
// may rename to just Equal() or that may be used for a simpler ID() only type comparison, but that is easy to
// compare anyway.
func (b Batch) StrictEqual(other Batch) bool {
	// TODO: do not follow the object for CreatedBy() to get id, but will need to add a CreatedById() to
	// the batch struct
	fmt.Printf("%v", b.CreatedAt())
	fmt.Printf("%v", other.CreatedAt())
	return b.ID() == other.ID() && b.Name() == other.Name() && b.BrewNotes() == other.BrewNotes() &&
		b.TastingNotes() == other.TastingNotes() && b.VolumeUnits() == other.VolumeUnits() &&
		b.VolumeInFermenter() == other.VolumeInFermenter() && b.VolumeBoiled() == other.VolumeBoiled() &&
		b.OriginalGravity() == other.OriginalGravity() && b.FinalGravity() == other.FinalGravity() &&
		b.RecipeURL() == other.RecipeURL() && b.CreatedBy().ID() == other.CreatedBy().ID() &&
		b.MaxTemperature() == other.MaxTemperature() && b.MinTemperature() == other.MinTemperature() &&
		b.AverageTemperature() == other.AverageTemperature() &&
		b.BrewedDate().Equal(other.BrewedDate()) && b.BottledDate().Equal(other.BottledDate()) &&
		b.CreatedAt().Equal(other.CreatedAt()) //&& b.UpdatedAt().Equal(other.UpdatedAt())
}

// Initializes and returns a new Batch instance
func NewBatch(id int, name string, brewedDate, bottledDate time.Time, volumeBoiled, volumeInFermenter float64,
	volumeUnits VolumeUnitType, originalGravity, finalGravity float64, createdBy User, createdAt, updatedAt time.Time,
	brewNotes, tastingNotes string, recipeURL string) Batch {
	return Batch{batch: batch{ID: id, Name: name, BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: volumeBoiled,
		VolumeInFermenter: volumeInFermenter, VolumeUnits: volumeUnits, CreatedBy: createdBy, CreatedAt: createdAt,
		UpdatedAt: updatedAt, BrewNotes: brewNotes, TastingNotes: tastingNotes, RecipeURL: recipeURL,
		OriginalGravity: originalGravity, FinalGravity: finalGravity}}
}

// Find a batch by exact match of attributes
// Currently allows lookup by `id` and `created_by_user_id`
// TODO: Use fields() to iterate over the fields and use the `db`
// tag to map field name to db field.
func FindBatch(params map[string]interface{}, db *sqlx.DB) (*Batch, error) {
	b := Batch{}
	var values []interface{}
	var where []string
	for _, k := range []string{"id", "created_by_user_id"} {
		if v, ok := params[k]; ok {
			values = append(values, v)
			// TODO: Deal with values from batch OR user table
			where = append(where, fmt.Sprintf("b.%s = ?", k))
		}
	}

	selectCols := ""
	for _, k := range b.queryColumns() {
		selectCols += fmt.Sprintf("b.%s, ", k)
	}

	u := User{}
	for _, k := range u.queryColumns() {
		selectCols += fmt.Sprintf("u.%s \"created_by.%s\", ", k, k)
	}

	q := `SELECT ` + strings.Trim(selectCols, ", ") + ` FROm batches b LEFT JOIN users u on u.id = b.created_by_user_id ` +
		`WHERE ` + strings.Join(where, " AND ")

	query := db.Rebind(q)
	err := db.Get(&b, query, values...)

	if err != nil {
		return nil, err
	}

	return &b, nil
}

// Save the User to the database.  If User.ID() is 0
// then an insert is performed, otherwise an update on the User matching that id.
func SaveBatch(db *sqlx.DB, b Batch) (Batch, error) {
	// TODO: TEST CASE
	if b.ID() != 0 {
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

	query := db.Rebind(`INSERT INTO batches (created_by_user_id, name, brew_notes, tasting_notes, brewed_date, bottled_date,
		volume_boiled, volume_in_fermenter, volume_units, original_gravity, final_gravity, recipe_url, max_temperature,
		min_temperature, average_temperature, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW()) RETURNING id, created_at, updated_at`)
	// TODO: Make the dates strings for sql to be happy
	err := db.QueryRow(
		query, b.CreatedBy().ID(), b.Name(), b.BrewNotes(), b.TastingNotes(), b.BrewedDate(), b.BottledDate(),
		b.VolumeBoiled(), b.VolumeInFermenter(), b.VolumeUnits(), b.OriginalGravity(), b.FinalGravity(), b.RecipeURL(),
		b.MaxTemperature(), b.MinTemperature(), b.AverageTemperature()).Scan(&batchId, &createdAt, &updatedAt)
	if err != nil {
		return b, err
	}

	b.batch.ID = batchId
	b.batch.CreatedAt = createdAt
	b.batch.UpdatedAt = updatedAt
	return b, nil
}

// Saves the passed in user to the database using an UPDATE
// Returns a new copy of the user with any updated values set upon success.
// Returns the same, unmodified User and errors on error
func UpdateBatch(db *sqlx.DB, b Batch) (Batch, error) {
	// TODO: TEST CASE
	var updatedAt time.Time

	// TODO: Use introspection and reflection to set these rather than manually managing this?
	query := db.Rebind(`UPDATE users SET created_by_user_id = ?, name = ?, brew_notes = ?, tasting_notes = ?,
		brewed_date = ?, bottled_date = ?, volume_boiled = ?, volume_in_fermenter = ?, volume_units = ?,
		original_gravity = ?, final_gravity = ?, recipe_url = ?, max_temperature = ?, min_temperature = ?,
		average_temperature = ?, updated_at = NOW() WHERE id = ?) RETURNING updated_at`)
	err := db.QueryRow(
		query, b.CreatedBy().ID(), b.Name(), b.BrewNotes(), b.TastingNotes(), b.BrewedDate(), b.BottledDate(),
		b.VolumeBoiled(), b.VolumeInFermenter(), b.VolumeUnits(), b.OriginalGravity(), b.FinalGravity(), b.RecipeURL(),
		b.MaxTemperature(), b.MinTemperature(), b.AverageTemperature()).Scan(&updatedAt)
	if err != nil {
		return b, err
	}
	b.batch.UpdatedAt = updatedAt
	return b, nil
}

// Return batches owned/created by a User, currently using default ordering only
// with cursor based pagination using the id.  May expand cursor pagination at some point
func BatchesForUser(db *sqlx.DB, u User, count *int, after *int) ([]Batch, error) {
	batches := []Batch{}
	var queryArgs []interface{}

	q := `SELECT * FROM batches WHERE created_by_user_id = ?`
	queryArgs = append(queryArgs, u.ID())
	if after != nil {
		q = q + ` and id > ?`
		queryArgs = append(queryArgs, *after)
	}

	if count != nil {
		q = fmt.Sprintf("%s LIMIT %d", *count)
	}

	err := db.Select(&batches, q, queryArgs...)
	return batches, err
}

type fermenter struct {
	// I could use name + user composite key for pk on these in the db, but I'm probably going to be lazy
	// and take the standard ORM-ish route and use an int or uuid  Int for now.
	ID            int
	Name          string
	Description   string
	Volume        float64
	VolumeUnits   VolumeUnitType
	FermenterType FermenterStyleType
	IsActive      bool
	IsAvailable   bool
	CreatedBy     User

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Fermenter struct {
	fermenter
}

func (f Fermenter) ID() int                           { return f.fermenter.ID }
func (f Fermenter) Name() string                      { return f.fermenter.Name }
func (f Fermenter) Description() string               { return f.fermenter.Description }
func (f Fermenter) VolumeUnits() VolumeUnitType       { return f.fermenter.VolumeUnits }
func (f Fermenter) FermenterType() FermenterStyleType { return f.fermenter.FermenterType }
func (f Fermenter) IsActive() bool                    { return f.fermenter.IsActive }
func (f Fermenter) IsAvailable() bool                 { return f.fermenter.IsAvailable }
func (f Fermenter) CreatedBy() User                   { return f.fermenter.CreatedBy }
func (f Fermenter) CreatedAt() time.Time              { return f.fermenter.CreatedAt }
func (f Fermenter) UpdatedAt() time.Time              { return f.fermenter.UpdatedAt }

func NewFermenter(id int, name, description string, volume float64, volumeUnits VolumeUnitType,
	fermenterType FermenterStyleType, isActive, isAvailable bool, createdBy User, createdAt, updatedAt time.Time) Fermenter {
	return Fermenter{fermenter{ID: id, Name: name, Description: description, Volume: volume, VolumeUnits: volumeUnits,
		FermenterType: fermenterType, IsActive: isActive, IsAvailable: isAvailable, CreatedBy: createdBy,
		CreatedAt: createdAt, UpdatedAt: updatedAt}}
}

// possibly should live elsewhere

// Thermometer will need some other unique identifier which the unit itself
// can know, ideally.
// TODO: This may also want extra metadata such as model or type?  That is probably
// going too far for now, so keep it simple.
type thermometer struct {
	ID        int
	Name      string
	CreatedBy User

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Thermometer struct {
	thermometer
}

func (t Thermometer) ID() int              { return t.thermometer.ID }
func (t Thermometer) Name() string         { return t.thermometer.Name }
func (t Thermometer) CreatedBy() User      { return t.thermometer.CreatedBy }
func (t Thermometer) CreatedAt() time.Time { return t.thermometer.CreatedAt }
func (t Thermometer) UpdatedAt() time.Time { return t.thermometer.UpdatedAt }

// Returns a new Thermometer
func NewThermometer(id int, name string, createdBy User, createdAt, updatedAt time.Time) Thermometer {
	return Thermometer{thermometer{ID: id, Name: name, CreatedBy: createdBy, CreatedAt: createdAt, UpdatedAt: updatedAt}}
}

// A single recorded temperature measurement from a thermometer
// This may get some tweaking to play nicely with data stored in Postgres or Influxdb
type temperatureMeasurement struct {
	ID           string // use a uuid
	Temperature  float64
	Units        TemperatureUnitType
	TimeRecorded time.Time // when the measurement was recorded
	Batch        Batch
	Thermometer  Thermometer
	Fermenter    Fermenter

	// not sure createdBy is a useful name in this case vs just `user` but its consistent
	CreatedBy User

	// when the record was created
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TemperatureMeasurement struct {
	temperatureMeasurement
}

func (t TemperatureMeasurement) ID() string                 { return t.temperatureMeasurement.ID }
func (t TemperatureMeasurement) Temperature() float64       { return t.temperatureMeasurement.Temperature }
func (t TemperatureMeasurement) Units() TemperatureUnitType { return t.temperatureMeasurement.Units }
func (t TemperatureMeasurement) TimeRecorded() time.Time    { return t.temperatureMeasurement.TimeRecorded }
func (t TemperatureMeasurement) Batch() Batch               { return t.temperatureMeasurement.Batch }
func (t TemperatureMeasurement) Thermometer() Thermometer   { return t.temperatureMeasurement.Thermometer }
func (t TemperatureMeasurement) Fermenter() Fermenter       { return t.temperatureMeasurement.Fermenter }
func (t TemperatureMeasurement) CreatedBy() User            { return t.temperatureMeasurement.CreatedBy }
func (t TemperatureMeasurement) CreatedAt() time.Time       { return t.temperatureMeasurement.CreatedAt }
func (t TemperatureMeasurement) UpdatedAt() time.Time       { return t.temperatureMeasurement.UpdatedAt }

func NewTemperatureMeasurement(id string, temperature float64, units TemperatureUnitType, batch Batch,
	thermometer Thermometer, fermenter Fermenter, timeRecorded, createdAt, updatedAt time.Time, createdBy User) TemperatureMeasurement {
	return TemperatureMeasurement{temperatureMeasurement{ID: id, Temperature: temperature, Units: units, Batch: batch,
		Thermometer: thermometer, Fermenter: fermenter, TimeRecorded: timeRecorded, CreatedAt: createdAt,
		UpdatedAt: updatedAt, CreatedBy: createdBy}}
}
