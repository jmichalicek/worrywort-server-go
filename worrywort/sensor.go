package worrywort

import (
	"fmt"
	"github.com/elgris/sqrl"
	"github.com/jmoiron/sqlx"
	"time"
)

// Sensor will need some other unique identifier which the unit itself
// can know, ideally.
// TODO: This may also want extra metadata such as model or type?  That is probably
// going too far for now, so keep it simple.
type Sensor struct {
	Id        *int64 `db:"id"`
	UUID      string `db:"uuid"`
	Name      string `db:"name"`
	CreatedBy *User  `db:"u"`
	UserId    *int64 `db:"user_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func buildSensorsQuery(params map[string]interface{}, db *sqlx.DB) *sqrl.SelectBuilder {
	query := sqrl.Select().From("sensors s")
	// TODO: test for filter by name... or make a more generic setup which just accepts anything
	// or a variadic list of sqrl stuff or a new type with name, comparison, value...
	// or maybe can leverage sqrl for that somehow?
	for _, k := range []string{"id", "user_id", "uuid", "name"} {
		// TODO: return error if not ok?
		if v, ok := params[k]; ok {
			query = query.Where(sqrl.Eq{fmt.Sprintf("s.%s", k): v})
		}
	}

	// TODO: nice API around letting this be optional? Leaning towards functions like
	// FindSensor and FindSensors should return the query or an object which has the query
	// plus ability to execute, allowing joins to be done later if desired rather than forcing.
	// But keep it simple and what I need 99% of the time for now.
	// TODO: related to above TODO, consider functional options - https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
	query = query.LeftJoin("users u ON s.user_id = u.id")

	for _, k := range []string{"id", "uuid", "name", "created_at", "updated_at", "user_id"} {
		query = query.Column(fmt.Sprintf("s.%s", k))
	}

	for _, k := range []string{"id", "uuid", "full_name", "username", "email", "password", "created_at", "updated_at"} {
		query = query.Column(fmt.Sprintf("u.%s \"u.%s\"", k, k))
	}

	// might make sense to just put this in FindSensors().  It's not useful to FindSensor()
	if v, ok := params["limit"]; ok {
		query = query.Limit(uint64(v.(int)))
	}
	if v, ok := params["offset"]; ok {
		query = query.Offset(uint64(v.(int)))
	}

	// q, v, _ := query.ToSql()
	// fmt.Printf("\n\nSQL:\n%s\nvals:%#v\n", q, v)
	return query
}

// Look up a single temperature sensor
// returns the first match, like .first() in Django
func FindSensor(params map[string]interface{}, db *sqlx.DB) (*Sensor, error) {
	// TODO: tempted to return the value rather than pointer to Sensor
	// was using nil return to know it was not found, but could use error
	// and a sensor with all zero values returned but this provides better consistency
	// with the plural FindFoo() stuff which returns a slice of pointers.
	sensor := new(Sensor)
	query, values, err := buildSensorsQuery(params, db).ToSql()
	if err == nil {
		err = db.Get(sensor, db.Rebind(query), values...)
	}
	return sensor, err
}

func FindSensors(params map[string]interface{}, db *sqlx.DB) ([]*Sensor, error) {
	// TODO:
	sensors := new([]*Sensor)
	query, values, err := buildSensorsQuery(params, db).ToSql()
	if err == nil {
		err = db.Select(sensors, db.Rebind(query), values...)
	}
	return *sensors, err
}

// Save the User to the database.  If User.Id() is 0
// then an insert is performed, otherwise an update on the User matching that id.
func (s *Sensor) Save(db *sqlx.DB) error {
	if s.Id == nil || *s.Id == 0 {
		return InsertSensor(db, s)
	} else {
		return UpdateSensor(db, s)
	}
}

func InsertSensor(db *sqlx.DB, t *Sensor) error {
	var updatedAt time.Time
	var createdAt time.Time
	sensorId := new(int64)
	_uuid := new(string)

	query := db.Rebind(`INSERT INTO sensors (user_id, name, updated_at)
		VALUES (?, ?, NOW()) RETURNING id, uuid, created_at, updated_at`)
	err := db.QueryRow(query, t.UserId, t.Name).Scan(sensorId, _uuid, &createdAt, &updatedAt)

	// I prefer handling the error case in the if, but this actually makes for slightly less code
	if err == nil {
		// TODO: Can I just assign these directly now in Scan()?
		t.Id = sensorId
		t.UUID = *_uuid
		t.CreatedAt = createdAt
		t.UpdatedAt = updatedAt
	}

	return err
}

func UpdateSensor(db *sqlx.DB, t *Sensor) error {
	// TODO: TEST CASE
	var updatedAt time.Time
	// TODO: Use introspection and reflection to set these rather than manually managing this?
	query := db.Rebind(`UPDATE sensors SET user_id = ?, name = ?, updated_at = NOW()
		WHERE id = ? RETURNING updated_at`)
	err := db.QueryRow(
		query, t.UserId, t.Name, t.Id).Scan(&updatedAt)
	if err == nil {
		t.UpdatedAt = updatedAt
	}
	return err
}
