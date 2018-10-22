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

type FermentorStyleType int32

const (
	BUCKET FermentorStyleType = iota
	CARBOY
	CONICAL
)

// TODO: Is this a good idea?  An error for invalid values on functions which take
// a map[string]interface{}
// This could be its own error type with field name, field value, and an Error() which
// formats nicely...
var TypeError error = errors.New("Invalid type specified")

type Batch struct {
	Id                 int            `db:"id"`
	CreatedBy          *User          `db:"created_by,prefix=u"` // TODO: think I will change this to User
	UserId             sql.NullInt64  `db:"user_id"`
	Name               string         `db:"name"`
	BrewNotes          string         `db:"brew_notes"`
	TastingNotes       string         `db:"tasting_notes"`
	BrewedDate         time.Time      `db:"brewed_date"`
	BottledDate        time.Time      `db:"bottled_date"`
	VolumeBoiled       float64        `db:"volume_boiled"` // sql nullfloats?
	VolumeInFermentor  float64        `db:"volume_in_fermentor"`
	VolumeUnits        VolumeUnitType `db:"volume_units"`
	OriginalGravity    float64        `db:"original_gravity"`
	FinalGravity       float64        `db:"final_gravity"` // TODO: sql.nullfloat64?
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
	b := Batch{}
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
	for _, k := range b.queryColumns() {
		selectCols += fmt.Sprintf("b.%s, ", k)
	}

	u := User{}
	for _, k := range u.queryColumns() {
		selectCols += fmt.Sprintf("u.%s \"created_by.%s\", ", k, k)
	}

	// TODO: STOP JOINING user here automatically!
	q := `SELECT ` + strings.Trim(selectCols, ", ") + ` FROM batches b LEFT JOIN users u on u.id = b.user_id ` +
		`WHERE ` + strings.Join(where, " AND ")

	query := db.Rebind(q)
	err := db.Get(&b, query, values...)

	if err != nil {
		return nil, err
	}

	return &b, nil
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

// Return batches owned/created by a User, currently using default ordering only
// with cursor based pagination using the id.  May expand cursor pagination at some point
func BatchesForUser(db *sqlx.DB, u User, count *int, after *int) (*[]Batch, error) {
	batches := []Batch{}
	var queryArgs []interface{}

	selectCols := ""
	// This doesn't seem like it could possibly be a great way to handle this.
	b := Batch{}
	for _, k := range b.queryColumns() {
		selectCols += fmt.Sprintf("b.%s, ", k)
	}

	for _, k := range u.queryColumns() {
		selectCols += fmt.Sprintf("u.%s \"created_by.%s\", ", k, k)
	}

	q := `SELECT ` + strings.Trim(selectCols, ", ") + ` FROM batches b LEFT JOIN users u on u.id = b.user_id ` +
		`WHERE user_id = ? `

	queryArgs = append(queryArgs, u.Id)
	if after != nil {
		q = q + ` and id > ?`
		queryArgs = append(queryArgs, *after)
	}

	if count != nil {
		q = q + fmt.Sprintf(" LIMIT %d", *count)
	}

	err := db.Select(&batches, db.Rebind(q), queryArgs...)
	if err != nil {
		return nil, err
	}
	return &batches, err
}

type Fermentor struct {
	// I could use name + user composite key for pk on these in the db, but I'm probably going to be lazy
	// and take the standard ORM-ish route and use an int or uuid  Int for now.
	Id            int                `db:"id"`
	Name          string             `db:"name"`
	Description   string             `db:"description"`
	Volume        float64            `db:"volume"`
	VolumeUnits   VolumeUnitType     `db:"volume_units"`
	FermentorType FermentorStyleType `db:"fermentor_type"`
	IsActive      bool               `db:"is_active"`
	IsAvailable   bool               `db:"is_available"`
	CreatedBy     *User              `db:"created_by,prefix=u"`
	UserId        sql.NullInt64      `db:"user_id"`

	// TODO: id and fk for current batch if it is attached to a batch?

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// TODO: REMOVE NewFermentor
func NewFermentor(id int, name, description string, volume float64, volumeUnits VolumeUnitType,
	fermentorType FermentorStyleType, isActive, isAvailable bool, createdBy User, createdAt, updatedAt time.Time) Fermentor {
	return Fermentor{Id: id, Name: name, Description: description, Volume: volume, VolumeUnits: volumeUnits,
		FermentorType: fermentorType, IsActive: isActive, IsAvailable: isAvailable, CreatedBy: &createdBy,
		CreatedAt: createdAt, UpdatedAt: updatedAt}
}

func FindFermentor(params map[string]interface{}, db *sqlx.DB) (*Fermentor, error) {

	f := Fermentor{}
	var values []interface{}
	var where []string
	for _, k := range []string{"id", "user_id"} {
		if v, ok := params[k]; ok {
			values = append(values, v)
			// TODO: Deal with values from temperature_sensor OR user table
			where = append(where, fmt.Sprintf("f.%s = ?", k))
		}
	}

	q := `SELECT f.id, f.name, f.description, f.volume, f.volume_units, f.fermentor_type, f.is_active, f.is_available, f.user_id
		FROM fermentors s WHERE ` + strings.Join(where, " AND ")
	query := db.Rebind(q)
	err := db.Get(&f, query, values...)

	if err != nil {
		return nil, err
	}

	return &f, nil
}

// Save a Fermentor - yes, inconsistent spelling.  Will be switchin to OR instead of ER globally.
func SaveFermentor(db *sqlx.DB, f Fermentor) (Fermentor, error) {
	if f.Id != 0 {
		return UpdateFermentor(db, f)
	} else {
		return InsertFermentor(db, f)
	}
}

func InsertFermentor(db *sqlx.DB, f Fermentor) (Fermentor, error) {
	var updatedAt time.Time
	var createdAt time.Time
	var fermentorId int

	query := db.Rebind(`INSERT INTO fermentors (user_id, name, description, volume, volume_units, fermentor_type,
		is_active, is_available, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW()) RETURNING id, created_at, updated_at`)
	err := db.QueryRow(query, f.UserId, f.Name, f.Description, f.Volume, f.VolumeUnits, f.FermentorType,
		f.IsActive, f.IsAvailable).Scan(&fermentorId, &createdAt, &updatedAt)
	if err != nil {
		return f, err
	}

	// TODO: Can I just assign these directly now in Scan()?
	f.Id = fermentorId
	f.CreatedAt = createdAt
	f.UpdatedAt = updatedAt
	return f, nil
}

func UpdateFermentor(db *sqlx.DB, f Fermentor) (Fermentor, error) {
	// TODO: TEST CASE
	var updatedAt time.Time
	// TODO: Use introspection and reflection to set these rather than manually managing this?
	query := db.Rebind(`UPDATE fermentors SET user_id = ?, name = ?, description = ?, volume = ?, volume_units = ?,
		fermentor_type = ?, is_active = ?, is_available = ?, updated_at = NOW() WHERE id = ? RETURNING updated_at`)
	err := db.QueryRow(query, f.UserId, f.Name, f.Description, f.Volume, f.VolumeUnits, f.FermentorType,
		f.IsActive, f.IsAvailable, f.Id).Scan(&updatedAt)
	if err == nil {
		f.UpdatedAt = updatedAt
	}
	return f, err
}

// possibly should live elsewhere

// TemperatureSensor will need some other unique identifier which the unit itself
// can know, ideally.
// TODO: This may also want extra metadata such as model or type?  That is probably
// going too far for now, so keep it simple.
type TemperatureSensor struct {
	Id        int           `db:"id"`
	Name      string        `db:"name"`
	CreatedBy *User         `db:"created_by,prefix=u"`
	UserId    sql.NullInt64 `db:"user_id"`
	// TODO: fk/id for current fermentor and current batch if attached to them?

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Returns a list of the db columns to use for a SELECT query
func (t TemperatureSensor) queryColumns() []string {
	// TODO: Way to dynamically build this using the `db` tag and reflection/introspection
	return []string{"id", "name", "created_at", "updated_at", "user_id"}
}

// Returns a new TemperatureSensor
func NewTemperatureSensor(id int, name string, createdBy *User, createdAt, updatedAt time.Time) TemperatureSensor {
	return TemperatureSensor{Id: id, Name: name, CreatedBy: createdBy, CreatedAt: createdAt, UpdatedAt: updatedAt}
}

// Look up temperature sensor
func FindTemperatureSensor(params map[string]interface{}, db *sqlx.DB) (*TemperatureSensor, error) {
	// TODO: Find a way to just pass in created_by sanely - maybe just manually map that to user_id if needed
	// sqlx may have a good way to do that already.
	t := TemperatureSensor{}
	var values []interface{}
	var where []string
	for _, k := range []string{"id", "user_id"} {
		if v, ok := params[k]; ok {
			values = append(values, v)
			// TODO: Deal with values from temperature_sensor OR user table
			where = append(where, fmt.Sprintf("t.%s = ?", k))
		}
	}

	selectCols := ""
	for _, k := range t.queryColumns() {
		selectCols += fmt.Sprintf("t.%s, ", k)
	}

	// TODO: Can I easily dynamically add in joining and attaching the User to this without overcomplicating the code?
	q := `SELECT ` + strings.Trim(selectCols, ", ") + ` FROM temperature_sensors t WHERE ` + strings.Join(where, " AND ")

	query := db.Rebind(q)
	err := db.Get(&t, query, values...)

	if err != nil {
		return nil, err
	}

	return &t, nil
}

// Look up a TemperatureSensor in the database and returns it with user joined.
// I should delete this rather than leaving commented, but leaving it here for easy reference for now.
// func FindTemperatureSensor(params map[string]interface{}, db *sqlx.DB) (*TemperatureSensor, error) {
// 	// TODO: Find a way to just pass in created_by sanely - maybe just manually map that to user_id if needed
// 	// sqlx may have a good way to do that already.
// 	t := TemperatureSensor{}
// 	var values []interface{}
// 	var where []string
// 	for _, k := range []string{"id", "user_id"} {
// 		if v, ok := params[k]; ok {
// 			values = append(values, v)
// 			// TODO: Deal with values from temperature_sensor OR user table
// 			where = append(where, fmt.Sprintf("t.%s = ?", k))
// 		}
// 	}
// 	selectCols := ""
// 	for _, k := range t.queryColumns() {
// 		selectCols += fmt.Sprintf("t.%s, ", k)
// 	}
// 	// TODO: improve user join (and other join in general) to
// 	// be less duplicated
// 	u := User{}
// 	for _, k := range u.queryColumns() {
// 		selectCols += fmt.Sprintf("u.%s \"created_by.%s\", ", k, k)
// 	}
// 	q := `SELECT ` + strings.Trim(selectCols, ", ") + ` FROM temperature_sensors t LEFT JOIN users u on u.id = t.user_id ` +
// 		`WHERE ` + strings.Join(where, " AND ")
// 	query := db.Rebind(q)
// 	err := db.Get(&t, query, values...)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &t, nil
// }

// Save the User to the database.  If User.Id() is 0
// then an insert is performed, otherwise an update on the User matching that id.
func SaveTemperatureSensor(db *sqlx.DB, tm TemperatureSensor) (TemperatureSensor, error) {
	if tm.Id != 0 {
		return UpdateTemperatureSensor(db, tm)
	} else {
		return InsertTemperatureSensor(db, tm)
	}
}

func InsertTemperatureSensor(db *sqlx.DB, t TemperatureSensor) (TemperatureSensor, error) {
	var updatedAt time.Time
	var createdAt time.Time
	var sensorId int

	query := db.Rebind(`INSERT INTO temperature_sensors (user_id, name, updated_at)
		VALUES (?, ?, NOW()) RETURNING id, created_at, updated_at`)
	err := db.QueryRow(query, t.UserId, t.Name).Scan(&sensorId, &createdAt, &updatedAt)
	if err != nil {
		return t, err
	}

	// TODO: Can I just assign these directly now in Scan()?
	t.Id = sensorId
	t.CreatedAt = createdAt
	t.UpdatedAt = updatedAt
	return t, nil
}

func UpdateTemperatureSensor(db *sqlx.DB, t TemperatureSensor) (TemperatureSensor, error) {
	// TODO: TEST CASE
	var updatedAt time.Time
	// TODO: Use introspection and reflection to set these rather than manually managing this?
	query := db.Rebind(`UPDATE temperature_sensors SET user_id = ?, name = ?, updated_at = NOW()
		WHERE id = ? RETURNING updated_at`)
	err := db.QueryRow(
		query, t.CreatedBy.Id, t.Name, t.Id).Scan(&updatedAt)
	if err == nil {
		t.UpdatedAt = updatedAt
	}
	t.UpdatedAt = updatedAt
	return t, err
}

// A single recorded temperature measurement from a temperatureSensor
// This may get some tweaking to play nicely with data stored in Postgres or Influxdb
type TemperatureMeasurement struct {
	Id                  string              `db:"id"` // use a uuid
	Temperature         float64             `db:"temperature"`
	Units               TemperatureUnitType `db:"units"`
	RecordedAt          time.Time           `db:"recorded_at"` // when the measurement was recorded
	Batch               *Batch              `db:"batch,prefix=b"`
	BatchId             sql.NullInt64       `db:"batch_id"`
	TemperatureSensor   *TemperatureSensor  `db:"temperature_sensor,prefix=ts"`
	TemperatureSensorId sql.NullInt64       `db:"temperature_sensor_id"`
	Fermentor           *Fermentor          `db:"fermentor,prefix=f"` // Do I really care? I might for history.
	FermentorId         sql.NullInt64       `db:"fermentor_id"`

	// not sure createdBy is a useful name in this case vs just `user` but its consistent
	CreatedBy *User         `db:"created_by,prefix=u"`
	UserId    sql.NullInt64 `db:"user_id"`

	// when the record was created
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Save the User to the database.  If User.Id() is 0
// then an insert is performed, otherwise an update on the User matching that id.
func SaveTemperatureMeasurement(db *sqlx.DB, tm TemperatureMeasurement) (TemperatureMeasurement, error) {
	if tm.Id != "" {
		return UpdateTemperatureMeasurement(db, tm)
	} else {
		return InsertTemperatureMeasurement(db, tm)
	}
}

// Inserts the passed in User into the database.
// Returns a new copy of the user with any updated values set upon success.
// Returns the same, unmodified User and errors on error
func InsertTemperatureMeasurement(db *sqlx.DB, tm TemperatureMeasurement) (TemperatureMeasurement, error) {
	var updatedAt time.Time
	var createdAt time.Time
	var measurementId string

	insertVals := []interface{}{tm.UserId, tm.Temperature, tm.Units, tm.RecordedAt, tm.BatchId,
		tm.TemperatureSensorId, tm.FermentorId}

	query := db.Rebind(`INSERT INTO temperature_measurements (user_id, temperature, units, recorded_at, created_at,
		updated_at, batch_id, temperature_sensor_id, fermentor_id)
		VALUES (?, ?, ?, ?, NOW(), NOW(), ?, ?, ?) RETURNING id, created_at, updated_at`)
	err := db.QueryRow(query, insertVals...).Scan(&measurementId, &createdAt, &updatedAt)
	if err != nil {
		return tm, err
	}

	// TODO: Can I just assign these directly now in Scan()?
	tm.Id = measurementId
	tm.CreatedAt = createdAt
	tm.UpdatedAt = updatedAt
	return tm, nil
}

// Saves the passed in user to the database using an UPDATE
// Returns a new copy of the user with any updated values set upon success.
// Returns the same, unmodified User and errors on error
func UpdateTemperatureMeasurement(db *sqlx.DB, tm TemperatureMeasurement) (TemperatureMeasurement, error) {
	var updatedAt time.Time

	paramVals := []interface{}{tm.UserId, tm.Temperature, tm.Units, tm.RecordedAt, tm.BatchId,
		tm.TemperatureSensorId, tm.FermentorId}

	paramVals = append(paramVals, tm.Id)
	// TODO: Use introspection and reflection to set these rather than manually managing this?
	query := db.Rebind(`UPDATE temperature_measurements SET user_id = ?, temperature = ?, units = ?,
		recorded_at = ?, updated_at = NOW(), batch_id = ?, temperature_sensor_id = ?, fermentor_id = ? WHERE id = ? RETURNING updated_at`)
	err := db.QueryRow(query, paramVals...).Scan(&updatedAt)
	if err != nil {
		return tm, err
	}
	tm.UpdatedAt = updatedAt
	return tm, nil
}
