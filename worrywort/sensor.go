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
	Uuid      string `db:"uuid"`
	Name      string `db:"name"`
	CreatedBy *User  `db:"created_by,prefix=u"`
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

	// TODO: join CreatedBy back in here after tests without that are all passing again

	for _, k := range []string{"id", "uuid", "name", "created_at", "updated_at", "user_id"} {
		query = query.Column(fmt.Sprintf("s.%s", k))
	}

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

// Look up a Sensor in the database and returns it with user joined.
// I should delete this rather than leaving commented, but leaving it here for easy reference for now.
// func FindSensor(params map[string]interface{}, db *sqlx.DB) (*Sensor, error) {
// 	// TODO: Find a way to just pass in created_by sanely - maybe just manually map that to user_id if needed
// 	// sqlx may have a good way to do that already.
// 	t := Sensor{}
// 	var values []interface{}
// 	var where []string
// 	for _, k := range []string{"id", "user_id"} {
// 		if v, ok := params[k]; ok {
// 			values = append(values, v)
// 			// TODO: Deal with values from sensor OR user table
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
// 	q := `SELECT ` + strings.Trim(selectCols, ", ") + ` FROM sensors t LEFT JOIN users u on u.id = t.user_id ` +
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
		t.Uuid = *_uuid
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
