package worrywort

import (
	"fmt"
	"github.com/elgris/sqrl"
	"github.com/jmoiron/sqlx"
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
		// TODO: Join the user on here
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
func (tm *TemperatureMeasurement) Save(db *sqlx.DB) error {
	if tm.Id != "" {
		return UpdateTemperatureMeasurement(db, tm)
	} else {
		return InsertTemperatureMeasurement(db, tm)
	}
}

// Insert a new TemperatureMeasurement into the database
func InsertTemperatureMeasurement(db *sqlx.DB, tm *TemperatureMeasurement) error {
	var updatedAt time.Time
	var createdAt time.Time
	var measurementId string

	insertVals := []interface{}{tm.UserId, tm.Temperature, tm.Units, tm.RecordedAt, tm.SensorId}

	query := db.Rebind(`INSERT INTO temperature_measurements (user_id, temperature, units, recorded_at, created_at,
		updated_at, sensor_id)
		VALUES (?, ?, ?, ?, NOW(), NOW(), ?) RETURNING id, created_at, updated_at`)
	err := db.QueryRow(query, insertVals...).Scan(&measurementId, &createdAt, &updatedAt)
	if err == nil {
		tm.Id = measurementId
		tm.CreatedAt = createdAt
		tm.UpdatedAt = updatedAt
	}
	return err
}

// Updates an existing TemperatureMeasurement in the database
func UpdateTemperatureMeasurement(db *sqlx.DB, tm *TemperatureMeasurement) error {
	var updatedAt time.Time
	paramVals := []interface{}{tm.UserId, tm.Temperature, tm.Units, tm.RecordedAt, tm.SensorId}
	paramVals = append(paramVals, tm.Id)
	// TODO: Use introspection and reflection to set these rather than manually managing this?
	query := db.Rebind(`UPDATE temperature_measurements SET user_id = ?, temperature = ?, units = ?,
		recorded_at = ?, updated_at = NOW(), sensor_id = ? WHERE id = ? RETURNING updated_at`)
	err := db.QueryRow(query, paramVals...).Scan(&updatedAt)
	if err == nil {
		tm.UpdatedAt = updatedAt
	}
	return err
}

// Build the query string and values slice for query for temperature measurement(s)
// as needed by sqlx db.Get() and db.Select() and returns them
func buildTemperatureMeasurementsQuery(params map[string]interface{}, db *sqlx.DB) *sqrl.SelectBuilder {
	query := sqrl.Select().From("temperature_measurements tm")

	// TODO: I suspect I will want to sort/filter by datetimes and by temperatures here as well
	// using ranges or gt/lt, not jus a straight equals.
	// TODO: filter by batch id(s) here?
	// TODO: allow multiple sensor ids?
	// TODO: query for multiple measurement ids?
	// TODO: An interesting take here might be to take a sqrl.Select() with any joins, etc. already
	//			 in place, but maybe there should just be args... a list of `join_cols` with `sensor`, `sensor.user`, etc.
	//       or maybe a generic `join_sensor(sb *sqrl.SelectBuilder, on string, prefix string)` function?
	for _, k := range []string{"id", "user_id", "sensor_id"} {
		// TODO: return error if not ok?
		if v, ok := params[k]; ok {
			query = query.Where(sqrl.Eq{fmt.Sprintf("tm.%s", k): v})
		}
	}

	for _, k := range []string{"id", "user_id", "sensor_id", "temperature",
		"units", "recorded_at", "created_at", "updated_at"} {
		query = query.Column(fmt.Sprintf("tm.%s", k))
	}

	if v, ok := params["limit"]; ok {
		query = query.Limit(uint64(v.(int)))
	}
	if v, ok := params["offset"]; ok {
		query = query.Offset(uint64(v.(int)))
	}

	// TODO: join sensor? sensor.user? how far do I nest that?
	return query
}

/*
 * Look up a single TemperatureMeasurement by its id
 */
func FindTemperatureMeasurement(params map[string]interface{}, db *sqlx.DB) (*TemperatureMeasurement, error) {
	measurement := new(TemperatureMeasurement)
	query, values, err := buildTemperatureMeasurementsQuery(params, db).ToSql()
	if err == nil {
		err = db.Get(measurement, db.Rebind(query), values...)
	}
	return measurement, err
}

func FindTemperatureMeasurements(params map[string]interface{}, db *sqlx.DB) ([]*TemperatureMeasurement, error) {
	// TODO: rewrite other Find* like this... I like the simplicity. I do prefer returning nil if there's an error,
	// but I'm going to follow some advice on a very short reddit thread - https://www.reddit.com/r/golang/comments/2xmnvs/returning_nil_for_a_struct/
	measurements := new([]*TemperatureMeasurement)
	query, values, err := buildTemperatureMeasurementsQuery(params, db).ToSql()
	if err == nil {
		err = db.Select(measurements, db.Rebind(query), values...)
	}
	return *measurements, err
}
