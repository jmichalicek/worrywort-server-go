package worrywort

// TODO: Currently not using this at all. Need to remove it or put it back into use.
import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"
)

type FermentorStyleType int64

const (
	BUCKET FermentorStyleType = iota
	CARBOY
	CONICAL
)

type Fermentor struct {
	// I could use name + user composite key for pk on these in the db, but I'm probably going to be lazy
	// and take the standard ORM-ish route and use an int or uuid  Int for now.
	Id            *int64             `db:"id"`
	Uuid              string         `db:"uuid"`
	Name          string             `db:"name"`
	Description   string             `db:"description"`
	Volume        float64            `db:"volume"`
	VolumeUnits   VolumeUnitType     `db:"volume_units"`
	FermentorType FermentorStyleType `db:"fermentor_type"`
	IsActive      bool               `db:"is_active"`
	IsAvailable   bool               `db:"is_available"`
	CreatedBy     *User              `db:"created_by,prefix=u"`
	UserId        *int64             `db:"user_id"`
	Batch         *Batch
	BatchId       *int64 `db:"batch_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func FindFermentor(params map[string]interface{}, db *sqlx.DB) (*Fermentor, error) {

	f := Fermentor{}
	var values []interface{}
	var where []string
	for _, k := range []string{"id", "user_id"} {
		if v, ok := params[k]; ok {
			values = append(values, v)
			// TODO: Deal with values from sensor OR user table
			where = append(where, fmt.Sprintf("f.%s = ?", k))
		}
	}

	q := `SELECT f.id, f.name, f.description, f.volume, f.volume_units, f.fermentor_type, f.is_active, f.is_available, f.user_id,
		f.batch_id, FROM fermentors s WHERE ` + strings.Join(where, " AND ")
	query := db.Rebind(q)
	err := db.Get(&f, query, values...)

	if err != nil {
		return nil, err
	}

	return &f, nil
}

// Save a Fermentor - yes, inconsistent spelling.  Will be switchin to OR instead of ER globally.
func SaveFermentor(db *sqlx.DB, f Fermentor) (Fermentor, error) {
	if f.Id != nil && *f.Id != 0 {
		return UpdateFermentor(db, f)
	} else {
		return InsertFermentor(db, f)
	}
}

func InsertFermentor(db *sqlx.DB, f Fermentor) (Fermentor, error) {
	var updatedAt time.Time
	var createdAt time.Time
	fermentorId := new(int64)

	query := db.Rebind(`INSERT INTO fermentors (user_id, name, description, volume, volume_units, fermentor_type,
		is_active, is_available, batch_id, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW()) RETURNING id, created_at, updated_at`)
	err := db.QueryRow(query, f.UserId, f.Name, f.Description, f.Volume, f.VolumeUnits, f.FermentorType,
		f.IsActive, f.IsAvailable, f.BatchId).Scan(fermentorId, &createdAt, &updatedAt)
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
		fermentor_type = ?, is_active = ?, is_available = ?, batch_id = ?, updated_at = NOW() WHERE id = ? RETURNING updated_at`)
	err := db.QueryRow(query, f.UserId, f.Name, f.Description, f.Volume, f.VolumeUnits, f.FermentorType,
		f.IsActive, f.IsAvailable, f.BatchId, f.Id).Scan(&updatedAt)
	if err == nil {
		f.UpdatedAt = updatedAt
	}
	return f, err
}
