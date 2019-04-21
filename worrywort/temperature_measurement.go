package worrywort

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"github.com/elgris/sqrl"
	"github.com/jmoiron/sqlx"
	"time"
)

// A single recorded temperature measurement from a temperatureSensor
// This may get some tweaking to play nicely with data stored in Postgres or Influxdb
type TemperatureMeasurement struct {
	Id          string              `db:"id" json:"id"` // use a uuid
	Temperature float64             `db:"temperature" json:"temperature"`
	Units       TemperatureUnitType `db:"units" json:"units"`
	RecordedAt  time.Time           `db:"recorded_at" json:"recorded_at"` // when the measurement was recorded
	// I could leave batch public and set it... it doesn't have to exist on the table.
	// but I think forcing use of Batch() enforces consistency
	// TODO: consider making an FK to batch again because that would allow having a measurement tied to a batch
	// but NOT a sensor such as manually inputting a measurement.
	batch    *Batch
	Sensor   *Sensor `db:"sensor,prefix=ts" json:"-"`
	SensorId *int64  `db:"sensor_id" json:"sensor_id"`

	// not sure createdBy is a useful name in this case vs just `user` but its consistent
	CreatedBy *User  `db:"created_by,prefix=u" json:"-"` // may want this eventually...
	UserId    *int64 `db:"user_id" json:"user_id"`

	// when the record was created
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
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
	// TODO: An interesting take here might be to take a sqrl.Select() with any joins, etc. already
	//			 in place, but maybe there should just be args... a list of `join_cols` with `sensor`, `sensor.user`, etc.
	//       or maybe a generic `join_sensor(sb *sqrl.SelectBuilder, on string, prefix string)` function?
	for _, k := range []string{"id", "user_id", "sensor_id"} {
		// TODO: return error if not ok?
		if v, ok := params[k]; ok {
			query = query.Where(sqrl.Eq{fmt.Sprintf("tm.%s", k): v})
		}
	}

	// TODO: Find a better way to check this... a more generic django style sensor__uuid where it splits
	// would be ideal. A naive could always just use the first as the table alias and update the model as such
	// and yes, this is a bit lazy, but just always join because I am going to want it anyway once I start
	// actually populating the sensor all the time
	query = query.LeftJoin("sensors s on s.id = tm.sensor_id") // left join in case there is not one... I suppose
	if v, ok := params["sensor_uuid"]; ok {
		query = query.Where(sqrl.Eq{"s.uuid": v})
	}

	if v, ok := params["batch_uuid"]; ok {
		// query = query.Where(sqrl.Eq{fmt.Sprintf("s.uuid", k): v})
		// TODO: Good way to add in prefetching the list of associations here, but only conditionally?
		// TODO: Also wondering if this may be better as a subquery
		query = query.Join(
			"batch_sensor_association bsa ON bsa.sensor_id = tm.sensor_id").Join("batches b ON b.id = bsa.batch_id")
		query = query.Where(sqrl.And{
			sqrl.Eq{"b.uuid": v},
			sqrl.Expr("tm.recorded_at >= bsa.associated_at AND (tm.recorded_at <= bsa.disassociated_at OR bsa.disassociated_at IS NULL)"),
			// TODO: follow up with sqrl dev to see if I am doing this wrong. It doesn't seem to like either of these.
			// sqrl.GtOrEq{"tm.recorded_at": "bsa.associated_at"},
			// and nested AND and OR
			// sqrl.And{
			// 	sqrl.Expr("tm.recorded_at >= bsa.associated_at"),
			// 	sqrl.Expr("tm.recorded_at <= bsa.disassociated_at OR bsa.disassociated_at IS NULL"),
			// },
		})
	}

	// TODO: handle sensor_uuid!
	// TODO: handle batch_uuid... could get interesting since batch is not joined to this... measurement.sensor.batch_sensor_assoc.batch.uuid
	// may be a cleaner way... hmmm
	for _, k := range []string{"id", "user_id", "sensor_id", "temperature", "units", "recorded_at", "created_at",
		"updated_at"} {
		query = query.Column(fmt.Sprintf("tm.%s", k))
	}

	if v, ok := params["limit"]; ok {
		query = query.Limit(uint64(v.(int)))
	}
	if v, ok := params["offset"]; ok {
		query = query.Offset(uint64(v.(int)))
	}

	// TODO: join sensor? sensor.user? how far do I nest that?
	// x, vals, _ := query.ToSql()
	// fmt.Printf("\n\n\nSQL: %s\nvals: %#v\n\n", x, vals)
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
