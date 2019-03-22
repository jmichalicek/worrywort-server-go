package worrywort

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSaveFermentor(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	t.Run("New Fermentor", func(t *testing.T) {
		fermentor, err := SaveFermentor(db, Fermentor{Name: "Fermentor", Description: "Fermentor Desc", Volume: 5.0,
			VolumeUnits: GALLON, FermentorType: BUCKET, IsActive: true, IsAvailable: true, UserId: u.Id})
		if err != nil {
			t.Errorf("%v", err)
		}
		if fermentor.Id == nil || *fermentor.Id == 0 {
			t.Errorf("SaveFermentor ended with unexpected id %v", fermentor.Id)
		}

		if fermentor.UpdatedAt.IsZero() {
			t.Errorf("SaveFermentor did not set UpdatedAt")
		}

		if fermentor.CreatedAt.IsZero() {
			t.Errorf("SaveFermentor did not set CreatedAt")
		}
	})

	t.Run("Update Fermentor", func(t *testing.T) {
		fermentor, err := SaveFermentor(db, Fermentor{Name: "Fermentor", Description: "Fermentor Desc", Volume: 5.0,
			VolumeUnits: GALLON, FermentorType: BUCKET, IsActive: true, IsAvailable: true, UserId: u.Id})
		// set date back in the past so that our date comparison consistenyly works
		fermentor.UpdatedAt = fermentor.UpdatedAt.AddDate(0, 0, -1)
		fermentor.Name = "Updated Name"
		updatedFermentor, err := SaveFermentor(db, fermentor)
		if err != nil {
			t.Errorf("%v", err)
		}
		if updatedFermentor.Name != "Updated Name" {
			t.Errorf("SaveFermentor did not update Name")
		}

		if fermentor.UpdatedAt == updatedFermentor.UpdatedAt {
			t.Errorf("SaveFermentor did not update UpdatedAt")
		}
	})
}

func TestFindSensor(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor := Sensor{Name: "Test Sensor", UserId: u.Id}
	sensor, err = SaveSensor(db, sensor)
	if err != nil {
		t.Fatalf("%v", err)
	}
	params := make(map[string]interface{})
	params["user_id"] = *u.Id
	params["id"] = *sensor.Id
	foundSensor, err := FindSensor(params, db)
	// foundSensor, err := FindSensor(map[string]interface{}{"user_id": u.Id, "id": sensor.Id}, db)
	if err != nil {
		t.Fatalf("%v", err)
	}
	// Have to be careful with this. if we do want all pointer to match up, then there is an issue here
	// because cmp dereferences nested pointers nicely.
	if foundSensor == nil || !cmp.Equal(*foundSensor, sensor) {
		t.Fatalf(cmp.Diff(*foundSensor, sensor))
	}
}

func TestSaveSensor(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	t.Run("Save New Sensor", func(t *testing.T) {
		// TODO: should be able to use go-cmp to do this now.
		sensor, err := SaveSensor(db, Sensor{Name: "Test Sensor", CreatedBy: &u})
		if err != nil {
			t.Errorf("%v", err)
		}

		if *sensor.Id == 0 || sensor.Id == nil {
			t.Errorf("SaveSensor returned with unexpected sensor id %v", sensor.Id)
		}

		if sensor.UpdatedAt.IsZero() {
			t.Errorf("SaveSensor did not set UpdatedAt")
		}

		if sensor.CreatedAt.IsZero() {
			t.Errorf("SaveSensor did not set CreatedAt")
		}
	})

	t.Run("Update Sensor", func(t *testing.T) {
		sensor, err := SaveSensor(db, Sensor{Name: "Test Sensor", CreatedBy: &u})
		// set date back in the past so that our date comparison consistenyly works
		sensor.UpdatedAt = sensor.UpdatedAt.AddDate(0, 0, -1)
		sensor.Name = "Updated Name"
		updatedSensor, err := SaveSensor(db, sensor)
		if err != nil {
			t.Errorf("%v", err)
		}
		if updatedSensor.Name != "Updated Name" {
			t.Errorf("SaveSensor did not update Name")
		}

		if sensor.UpdatedAt == updatedSensor.UpdatedAt {
			t.Errorf("SaveSensor did not update UpdatedAt")
		}
	})
}

func TestSaveTemperatureMeasurement(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor, err := SaveSensor(db, Sensor{Name: "Test Sensor", CreatedBy: &u})
	if err != nil {
		t.Fatalf("%v", err)
	}

	t.Run("Save New Measurement With All Fields", func(t *testing.T) {
		m, err := SaveTemperatureMeasurement(db,
			TemperatureMeasurement{CreatedBy: &u, UserId: u.Id, Sensor: &sensor, SensorId: sensor.Id,
				Temperature: 70.0, Units: FAHRENHEIT, RecordedAt: time.Now()})
		if err != nil {
			t.Errorf("%v", err)
		}
		if m.Id == "" {
			t.Errorf("SaveTemperatureMeasurement did not set id on new TemperatureMeasurement")
		}

		if m.UpdatedAt.IsZero() {
			t.Errorf("SaveTemperatureMeasurement did not set UpdatedAt")
		}

		if m.CreatedAt.IsZero() {
			t.Errorf("SaveTemperatureMeasurement did not set CreatedAt")
		}
		// TODO: Just query for the expected measurement
		newMeasurement := TemperatureMeasurement{}
		selectCols := ""
		for _, k := range u.queryColumns() {
			selectCols += fmt.Sprintf("u.%s \"created_by.%s\", ", k, k)
		}
		selectCols += fmt.Sprintf("ts.id \"sensor.id\", ts.name \"sensor.name\", ")
		q := `SELECT tm.temperature, tm.units,  ` + strings.Trim(selectCols, ", ") + ` from temperature_measurements tm LEFT JOIN users u ON u.id = tm.user_id LEFT JOIN sensors ts ON ts.id = tm.sensor_id WHERE tm.id = ? AND tm.user_id = ? AND tm.sensor_id = ?`
		query := db.Rebind(q)
		err = db.Get(&newMeasurement, query, m.Id, u.Id, sensor.Id)

		if err != nil {
			t.Errorf("%v", err)
		}
	})

	t.Run("Save New Measurement Without Optional Fields", func(t *testing.T) {
		m, err := SaveTemperatureMeasurement(db,
			TemperatureMeasurement{CreatedBy: &u, UserId: u.Id, SensorId: sensor.Id, Sensor: &sensor, Temperature: 70.0,
				Units: FAHRENHEIT, RecordedAt: time.Now()})
		if err != nil {
			t.Errorf("%v", err)
		}
		if m.Id == "" {
			t.Errorf("SaveTemperatureMeasurement did not set id on new TemperatureMeasurement")
		}

		if m.UpdatedAt.IsZero() {
			t.Errorf("SaveTemperatureMeasurement did not set UpdatedAt")
		}

		if m.CreatedAt.IsZero() {
			t.Errorf("SaveTemperatureMeasurement did not set CreatedAt")
		}

		newMeasurement := TemperatureMeasurement{}
		q := `SELECT tm.temperature, tm.units, tm.user_id, tm.sensor_id from temperature_measurements tm LEFT JOIN users u ON u.id = tm.user_id LEFT JOIN sensors ts ON ts.id = tm.sensor_id WHERE tm.id = ? AND tm.user_id = ? AND tm.sensor_id = ?`
		query := db.Rebind(q)
		err = db.Get(&newMeasurement, query, m.Id, u.Id, sensor.Id)

		if err != nil {
			t.Errorf("%v", err)
		}
	})

	t.Run("Update Temperature Measurement", func(t *testing.T) {
		m, err := SaveTemperatureMeasurement(db,
			TemperatureMeasurement{CreatedBy: &u, UserId: u.Id, SensorId: sensor.Id, Sensor: &sensor, Temperature: 70.0,
				Units: FAHRENHEIT, RecordedAt: time.Now()})
		if err != nil {
			t.Errorf("%v", err)
		}
		// set date back in the past so that our date comparison consistenyly works
		m.UpdatedAt = sensor.UpdatedAt.AddDate(0, 0, -1)
		// TODO: Intend to change this so that we set BatchId and save to update the Batch, not assign an object
		updatedMeasurement, err := SaveTemperatureMeasurement(db, m)
		if err != nil {
			t.Errorf("%v", err)
		}

		if m.UpdatedAt == updatedMeasurement.UpdatedAt {
			t.Errorf("SaveSensor did not update UpdatedAt. Expected: %v\nGot: %v", m.UpdatedAt, updatedMeasurement.UpdatedAt)
		}
	})
}

func TestTemperatureMeasurementModel(t *testing.T) {
	// Set up the db using sql.Open() and sqlx.NewDb() rather than sqlx.Open() so that the custom
	// `txdb` db type may be used with Open() but can still be registered as postgres with sqlx
	// so that sqlx' Rebind() functions.

	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	createdAt := time.Now().Round(time.Microsecond)
	updatedAt := time.Now().Round(time.Microsecond)
	// THe values when returned by postgres will be microsecond accuracy, but golang default
	// is nanosecond, so we round these for easy comparison. This can probably now be
	// dealt with by go-cmp
	brewedDate := time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute).Round(time.Microsecond)
	batch := Batch{UserId: u.Id, BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5,
		VolumeInFermentor: 4.5, VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020,
		CreatedAt: createdAt, UpdatedAt: updatedAt, BrewNotes: "Brew Notes", TastingNotes: "Taste Notes",
		RecipeURL: "http://example.org/beer"}
	err = batch.Save(db)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}
	sensor, err := SaveSensor(db, Sensor{Name: "Test Sensor", CreatedBy: &u})
	if err != nil {
		t.Fatalf("%v", err)
	}

	// to ensure it is in the past
	mTime := time.Now().Add(time.Duration(-5) * time.Minute).Round(time.Microsecond)
	_, err = AssociateBatchToSensor(batch, sensor, "", &mTime, db)
	measurement, err := SaveTemperatureMeasurement(db,
		TemperatureMeasurement{CreatedBy: &u, UserId: u.Id, SensorId: sensor.Id,
			Temperature: 70.0, Units: FAHRENHEIT, RecordedAt: time.Now()})
	if err != nil {
		t.Fatalf("%v", err)
	}

	t.Run("Batch()", func(t *testing.T) {
		// TODO: Add test to ensure that if the association is outside of the measurement time
		// that does not result in returning the batch and ensure that batches for different
		// sensor are not returned
		b, err := measurement.Batch(db)
		if err != nil {
			t.Errorf("%v", err)
		}

		if !cmp.Equal(&batch, b) {
			t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(batch, b))
		}
	})
}

func TestFindBatch(t *testing.T) {
	// Set up the db using sql.Open() and sqlx.NewDb() rather than sqlx.Open() so that the custom
	// `txdb` db type may be used with Open() but can still be registered as postgres with sqlx
	// so that sqlx' Rebind() functions.

	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	createdAt := time.Now().Round(time.Microsecond)
	updatedAt := time.Now().Round(time.Microsecond)
	// THe values when returned by postgres will be microsecond accuracy, but golang default
	// is nanosecond, so we round these for easy comparison
	brewedDate := time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute).Round(time.Microsecond)
	b := Batch{UserId: u.Id, BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5, VolumeInFermentor: 4.5,
		VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedAt: createdAt, UpdatedAt: updatedAt,
		BrewNotes: "Brew Notes", TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer"}
	err = b.Save(db)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	batchArgs := make(map[string]interface{})
	batchArgs["user_id"] = u.Id
	batchArgs["id"] = b.Id
	found, err := FindBatch(batchArgs, db)
	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	} else if !cmp.Equal(&b, found) {
		t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(&b, found))
		// t.Errorf("Expected: %v\nGot: %v\n", &b, found)
	}
}

func TestFindBatches(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	u2 := User{Email: "user2@example.com", FirstName: "Justin", LastName: "M"}
	err = u2.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	createdAt := time.Now().Round(time.Microsecond)
	updatedAt := time.Now().Round(time.Microsecond)
	// THe values when returned by postgres will be microsecond accuracy, but golang default
	// is nanosecond, so we round these for easy comparison
	brewedDate := time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute).Round(time.Microsecond)
	b := Batch{Name: "Testing", UserId: u.Id, BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5,
		VolumeInFermentor: 4.5, VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedAt: createdAt,
		UpdatedAt: updatedAt, BrewNotes: "Brew Notes", TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer"}
	err = b.Save(db)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	b2 := Batch{Name: "Testing 2", UserId: u.Id, BrewedDate: time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond), VolumeBoiled: 5, VolumeInFermentor: 4.5,
		VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, CreatedAt: createdAt, UpdatedAt: updatedAt,
		BrewNotes: "Brew Notes", TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer", BottledDate: time.Now().Add(time.Duration(5) * time.Minute).Round(time.Microsecond)}
	err = b2.Save(db)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	u2batch := Batch{Name: "Testing 2", UserId: u2.Id, BrewedDate: time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond),
		VolumeBoiled: 5, VolumeInFermentor: 4.5, VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020,
		CreatedAt: createdAt, UpdatedAt: updatedAt, BrewNotes: "Brew Notes", TastingNotes: "Taste Notes",
		RecipeURL: "http://example.org/beer", BottledDate: time.Now().Add(time.Duration(5) * time.Minute).Round(time.Microsecond)}

	err = u2batch.Save(db)
	if err != nil {
		t.Fatalf("Unexpected error saving batch: %s", err)
	}

	// TODO: split up into sub tests for different functionality... no pagination, pagination, etc.
	batches, err := FindBatches(map[string]interface{}{"user_id": u.Id}, db)
	if err != nil {
		t.Fatalf("\n%v\n", err)
	}

	expected := []*Batch{&b, &b2}
	if !cmp.Equal(expected, batches) {
		t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(expected, batches))
	}
}

// TODO: WRITE THESE NOW
func TestInsertBatch(t *testing.T) {}
func TestUpdateBatch(t *testing.T) {}
func TestSaveBatch(t *testing.T)   {}

func TestBatchSenssorAssociations(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u := User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	brewedDate := time.Now().Add(time.Duration(1) * time.Minute).Round(time.Microsecond)
	bottledDate := brewedDate.Add(time.Duration(10) * time.Minute).Round(time.Microsecond)
	batch := Batch{Name: "Testing", UserId: u.Id, BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5,
		VolumeInFermentor: 4.5, VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, BrewNotes: "Brew Notes",
		TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer"}
	err = batch.Save(db)
	if err != nil {
		t.Fatalf("%v", err)
	}

	sensor, err := SaveSensor(db, Sensor{Name: "Test Sensor", CreatedBy: &u})
	if err != nil {
		t.Fatalf("%v", err)
	}

	cleanAssociations := func() {
		q := `DELETE FROM batch_sensor_association WHERE sensor_id = ? AND batch_id = ?`
		q = db.Rebind(q)
		_, err := db.Exec(q, sensor.Id, batch.Id)
		if err != nil {
			panic(err)
		}
	}

	t.Run("AssociateBatchToSensor()", func(t *testing.T) {
		defer cleanAssociations()
		association, err := AssociateBatchToSensor(batch, sensor, "Testing", nil, db)
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		var newAssociation BatchSensor
		q := `SELECT bs.id, bs.sensor_id, bs.batch_id, bs.description, bs.associated_at, bs.disassociated_at, bs.created_at,
			bs.updated_at FROM batch_sensor_association bs WHERE bs.id = ? AND bs.sensor_id = ? AND bs.batch_id = ?
			AND bs.description = ? AND bs.associated_at = ? AND bs.created_at = ? AND bs.updated_at = ?
			AND bs.disassociated_at IS NULL`
		query := db.Rebind(q)
		err = db.Get(&newAssociation, query, association.Id, sensor.Id, batch.Id, "Testing", association.AssociatedAt,
			association.CreatedAt, association.UpdatedAt)

		if err != nil {
			t.Fatalf("%v", err)
		}

		// Make sure these really got set
		if (*association).AssociatedAt.IsZero() {
			t.Errorf("AssociateBatchToSensor did not set AssociatedAt")
		}

		if (*association).UpdatedAt.IsZero() {
			t.Errorf("AssociateBatchToSensor did not set UpdatedAt")
		}

		if (*association).CreatedAt.IsZero() {
			t.Errorf("AssociateBatchToSensor did not set CreatedAt")
		}
	})

	t.Run("UpdateBatchSensorAssociation()", func(t *testing.T) {
		defer cleanAssociations()
		aPtr, err := AssociateBatchToSensor(batch, sensor, "Testing", nil, db)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		association := *aPtr
		association.Description = "Updated"
		updated, err := UpdateBatchSensorAssociation(association, db)
		if err != nil {
			t.Fatalf("%v", err)
		}
		// Making sure the change was persisted to the db
		updated2 := BatchSensor{}
		q := `SELECT id, sensor_id, batch_id, description, associated_at, disassociated_at, created_at,
			updated_at FROM batch_sensor_association bs WHERE id = ? AND sensor_id = ? AND batch_id = ? AND description = ?
			AND disassociated_at IS NULL`
		query := db.Rebind(q)
		err = db.Get(&updated2, query, association.Id, association.SensorId, association.BatchId, "Updated")
		if err != nil {
			t.Fatalf("%v", err)
		}

		if !reflect.DeepEqual(*updated, updated2) {
			t.Errorf("Expected: %s\nGot: %s. Changes may not have persisted to the database.", spew.Sdump(updated), spew.Sdump(updated2))
		}
	})
}
