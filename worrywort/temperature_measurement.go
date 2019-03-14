package worrywort

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"
)

// A single recorded temperature measurement from a temperatureSensor
// This may get some tweaking to play nicely with data stored in Postgres or Influxdb
type TemperatureMeasurement struct {
	Id          string              `db:"id"` // use a uuid
	Temperature float64             `db:"temperature"`
	Units       TemperatureUnitType `db:"units"`
	RecordedAt  time.Time           `db:"recorded_at"` // when the measurement was recorded
	// I could leave batch public and set it... it doesn't have to exist on the table.
	// but I think forcing use of Batch() enforces consistency
	batch    *Batch
	Sensor   *Sensor `db:"sensor,prefix=ts"`
	SensorId *int64  `db:"sensor_id"`

	// not sure createdBy is a useful name in this case vs just `user` but its consistent
	CreatedBy *User  `db:"created_by,prefix=u"`
	UserId    *int64 `db:"user_id"`

	// when the record was created
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (tm *TemperatureMeasurement) Batch(db *sqlx.DB) (*Batch, error) {
	// TODO: Is this a good idea? it's not going to scale with list queries on graphql stuff
	if tm.batch == nil {
		// TODO: again... sure I should be able to just make a nil pointer to Batch right off here.
		b := Batch{}
		values := []interface{}{tm.RecordedAt, tm.RecordedAt, tm.SensorId}

		// TODO: I would rather have just a central ORM-ish function to do this, but it's way easier
		// to write an efficient query for it here.
		// Because associated_at can be modified, in the case that disassociated is null, we could potentially
		// get the wrong association without checking the associated_at time as well
		q := `SELECT b.* FROM batches b LEFT JOIN batch_sensor_association bsa
			ON bsa.batch_id = b.id AND bsa.associated_at <= ?
			AND (bsa.disassociated_at IS NULL OR bsa.disassociated_at >= ?) WHERE bsa.sensor_id = ?
			LIMIT 1`
		// q := `SELECT b.* FROM batch_sensor_association bsa INNER JOIN batches b ON batch.id = bsa.batch_id WHERE
		//  AND bsa.associated_at <= ? (bsa.disassociated_at IS NULL OR bsa.disassociated_at > ?) AND bsa.sensor_id = ? LIMIT 1`

		query := db.Rebind(q)
		err := db.Get(&b, query, values...)
		if err != nil {
			// TODO: log error
			return nil, err
		} else {
			tm.batch = &b
		}
	}
	return tm.batch, nil
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

	insertVals := []interface{}{tm.UserId, tm.Temperature, tm.Units, tm.RecordedAt, tm.SensorId}

	query := db.Rebind(`INSERT INTO temperature_measurements (user_id, temperature, units, recorded_at, created_at,
		updated_at, sensor_id)
		VALUES (?, ?, ?, ?, NOW(), NOW(), ?) RETURNING id, created_at, updated_at`)
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
	paramVals := []interface{}{tm.UserId, tm.Temperature, tm.Units, tm.RecordedAt, tm.SensorId}
	paramVals = append(paramVals, tm.Id)
	// TODO: Use introspection and reflection to set these rather than manually managing this?
	query := db.Rebind(`UPDATE temperature_measurements SET user_id = ?, temperature = ?, units = ?,
		recorded_at = ?, updated_at = NOW(), sensor_id = ? WHERE id = ? RETURNING updated_at`)
	err := db.QueryRow(query, paramVals...).Scan(&updatedAt)
	if err != nil {
		return tm, err
	}
	tm.UpdatedAt = updatedAt
	return tm, nil
}

// Build the query string and values slice for query for temperature measurement(s)
// as needed by sqlx db.Get() and db.Select() and returns them
func buildTemperatureMeasurementsQuery(params map[string]interface{}, db *sqlx.DB) (string, []interface{}, error) {
	// TODO: Pass in limit, offset!
	var values []interface{}
	var where []string
	// TODO: I suspect I will want to sort/filter by datetimes and by temperatures here as well
	// using ranges or gt/lt, not jus a straight equals.
	for _, k := range []string{"id", "user_id", "batch_id", "sensor_id"} {
		if v, ok := params[k]; ok {
			values = append(values, v)
			// TODO: Deal with values from sensor OR user table
			where = append(where, fmt.Sprintf("tm.%s = ?", k))
		}
	}

	selectCols := ""
	// as in BatchesForUser, this now seems dumb
	// queryCols := []string{"id", "name", "created_at", "updated_at", "user_id"}
	// If I need this many places, maybe make a const
	for _, k := range []string{"id", "user_id", "batch_id", "sensor_id", "temperature",
		"units", "recorded_at", "created_at", "updated_at"} {
		selectCols += fmt.Sprintf("tm.%s, ", k)
	}

	// TODO: Can I easily dynamically add in joining and attaching the User to this without overcomplicating the code?
	q := `SELECT ` + strings.Trim(selectCols, ", ") + ` FROM temperature_measurements tm WHERE ` + strings.Join(where, " AND ")

	return db.Rebind(q), values, nil
}

/*
 * Look up a single TemperatureMeasurement by its id
 */
func FindTemperatureMeasurement(params map[string]interface{}, db *sqlx.DB) (*TemperatureMeasurement, error) {
	query, values, err := buildTemperatureMeasurementsQuery(params, db)
	measurement := TemperatureMeasurement{}
	err = db.Get(&measurement, query, values...)

	if err != nil {
		return nil, err
	}

	return &measurement, err
}

func FindTemperatureMeasurements(params map[string]interface{}, db *sqlx.DB) ([]*TemperatureMeasurement, error) {
	query, values, err := buildTemperatureMeasurementsQuery(params, db)
	measurements := []*TemperatureMeasurement{}
	err = db.Select(&measurements, query, values...)

	if err != nil {
		return nil, err
	}

	return measurements, err
}
