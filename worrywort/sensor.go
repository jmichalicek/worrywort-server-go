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
	for _, k := range []string{"id", "user_id", "uuid"} {
		// TODO: return error if not ok?
		if v, ok := params[k]; ok {
			query = query.Where(sqrl.Eq{fmt.Sprintf("s.%s", k): v})
		}
	}

	// Careful here - user_id is nullable but that will cause this to not return those sensors.
	// In all cases of user oriented queries that is desired, but for an admin type lookup it might not be.
	query = query.Join("users u ON s.user_id = u.id")

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
	sensor := new(Sensor)
	query, values, err := buildSensorsQuery(params, db).ToSql()
	if err == nil {
		err = db.Get(sensor, db.Rebind(query), values...)
	}
	return sensor, err
}

func FindSensors(params map[string]interface{}, db *sqlx.DB) ([]*Sensor, error) {
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
