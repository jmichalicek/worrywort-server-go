package worrywort

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"
)

// Sensor will need some other unique identifier which the unit itself
// can know, ideally.
// TODO: This may also want extra metadata such as model or type?  That is probably
// going too far for now, so keep it simple.
type Sensor struct {
	Id        *int32           `db:"id"`
	Name      string        `db:"name"`
	CreatedBy *User         `db:"created_by,prefix=u"`
	UserId    *int32 `db:"user_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Look up a single temperature sensor
// returns the first match, like .first() in Django
// May change this up to just look up by id and then any other comparisons could
// be done directly on the object
func FindSensor(params map[string]interface{}, db *sqlx.DB) (*Sensor, error) {
	sensors, err := FindSensors(params, db)
	if err == nil && len(sensors) >= 1 {
		return sensors[0], err
	}
	return nil, err
}

func FindSensors(params map[string]interface{}, db *sqlx.DB) ([]*Sensor, error) {
	// TODO: Find a way to just pass in created_by sanely - maybe just manually map that to user_id if needed
	// sqlx may have a good way to do that already.
	// TODO: Pass in limit, offset!
	// TODO: Maybe.  Move most of this logic to a function shared by FindSensor and
	// FIndSensors so they just need to build the query with the shared logic then
	// use db.Get() or db.Select()... only true if desired to have single error if more than 1 result
	sensors := []*Sensor{}
	var values []interface{}
	var where []string
	for _, k := range []string{"id", "user_id"} {
		if v, ok := params[k]; ok {
			values = append(values, v)
			// TODO: Deal with values from sensor OR user table
			where = append(where, fmt.Sprintf("t.%s = ?", k))
		}
	}

	selectCols := ""
	// as in BatchesForUser, this now seems dumb
	// queryCols := []string{"id", "name", "created_at", "updated_at", "user_id"}
	// If I need this many places, maybe make a const
	for _, k := range []string{"id", "name", "created_at", "updated_at", "user_id"} {
		selectCols += fmt.Sprintf("t.%s, ", k)
	}

	// TODO: Can I easily dynamically add in joining and attaching the User to this without overcomplicating the code?
	q := `SELECT ` + strings.Trim(selectCols, ", ") + ` FROM sensors t WHERE ` + strings.Join(where, " AND ")
	query := db.Rebind(q)
	err := db.Select(&sensors, query, values...)

	if err != nil {
		return nil, err
	}

	return sensors, nil
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
func SaveSensor(db *sqlx.DB, tm Sensor) (Sensor, error) {
	if tm.Id == nil || *tm.Id == 0 {
		return InsertSensor(db, tm)
	} else {
		return UpdateSensor(db, tm)
	}
}

func InsertSensor(db *sqlx.DB, t Sensor) (Sensor, error) {
	var updatedAt time.Time
	var createdAt time.Time
	sensorId := new(int32)

	query := db.Rebind(`INSERT INTO sensors (user_id, name, updated_at)
		VALUES (?, ?, NOW()) RETURNING id, created_at, updated_at`)
	err := db.QueryRow(query, t.UserId, t.Name).Scan(sensorId, &createdAt, &updatedAt)
	if err != nil {
		return t, err
	}

	// TODO: Can I just assign these directly now in Scan()?
	t.Id = sensorId
	t.CreatedAt = createdAt
	t.UpdatedAt = updatedAt
	return t, nil
}

func UpdateSensor(db *sqlx.DB, t Sensor) (Sensor, error) {
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
	t.UpdatedAt = updatedAt
	return t, err
}
