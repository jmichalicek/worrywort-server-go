package worrywort

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"strings"
	"testing"
	"time"
)

// utility to add a given number of minutes to a time.Time and round to match
// what postgres returns
func addMinutes(d time.Time, increment int) time.Time {
	return d.Add(time.Duration(increment) * time.Minute).Round(time.Microsecond)
}

// Make a standard, generic batch for testing
// optionally attach the user
func makeTestBatch(u *User, attachUser bool) Batch {
	b := Batch{Name: "Testing", BrewedDate: addMinutes(time.Now(), -60),
		BottledDate: addMinutes(time.Now(), -10), VolumeBoiled: 5, VolumeInFermentor: 4.5, VolumeUnits: GALLON,
		OriginalGravity: 1.060, FinalGravity: 1.020, UserId: u.Id, BrewNotes: "Brew notes", TastingNotes: "Taste notes",
		RecipeURL: "http://example.org/beer"}
	if attachUser {
		b.CreatedBy = u
	}
	return b
}

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
		fermentor := Fermentor{Name: "Fermentor", Description: "Fermentor Desc", Volume: 5.0,
			VolumeUnits: GALLON, FermentorType: BUCKET, IsActive: true, IsAvailable: true, UserId: u.Id}
		if err := fermentor.Save(db); err != nil {
			t.Fatalf("%v", err)
		}
		if fermentor.Id == nil || *fermentor.Id == 0 {
			t.Errorf("Save() ended with unexpected id %v", fermentor.Id)
		}

		if fermentor.UpdatedAt.IsZero() {
			t.Errorf("Save() did not set UpdatedAt")
		}

		if fermentor.CreatedAt.IsZero() {
			t.Errorf("Save() did not set CreatedAt")
		}
	})

	t.Run("Update Fermentor", func(t *testing.T) {
		fermentor := Fermentor{Name: "Fermentor", Description: "Fermentor Desc", Volume: 5.0,
			VolumeUnits: GALLON, FermentorType: BUCKET, IsActive: true, IsAvailable: true, UserId: u.Id}
		if err := fermentor.Save(db); err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		fermentor.Name = "Updated Name"
		if err := fermentor.Save(db); err != nil {
			t.Fatalf("%v", err)
		}

		updated, err := FindFermentor(map[string]interface{}{"id": *fermentor.Id}, db)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}

		if !cmp.Equal(&fermentor, updated) {
			t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(&fermentor, updated))
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

	sensor := Sensor{Name: "Test Sensor", UserId: u.Id, CreatedBy: &u}
	if err := sensor.Save(db); err != nil {
		t.Fatalf("%v", err)
	}
	params := make(map[string]interface{})
	params["user_id"] = *u.Id
	params["id"] = *sensor.Id
	foundSensor, err := FindSensor(params, db)
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
		sensor := Sensor{Name: "Test Sensor", CreatedBy: &u, UserId: u.Id}
		if err := sensor.Save(db); err != nil {
			t.Errorf("%v", err)
		}

		if *sensor.Id == 0 || sensor.Id == nil {
			t.Errorf("sensor.Save() returned with unexpected sensor id %v", sensor.Id)
		}

		if sensor.UpdatedAt.IsZero() {
			t.Errorf("sensor.Save() did not set UpdatedAt")
		}

		if sensor.CreatedAt.IsZero() {
			t.Errorf("sensor.Save() did not set CreatedAt")
		}
	})

	t.Run("Update Sensor", func(t *testing.T) {
		sensor := Sensor{Name: "Test Sensor", UserId: u.Id, CreatedBy: &u}
		if err := sensor.Save(db); err != nil {
			t.Errorf("%v", err)
		}
		sensor.Name = "Updated Name"
		if err := sensor.Save(db); err != nil {
			t.Errorf("%v", err)
		}

		updated, err := FindSensor(map[string]interface{}{"id": sensor.Id}, db)
		if err != nil {
			t.Errorf("Got unexpected error: %s\n", err)
		}
		if !cmp.Equal(&sensor, updated) {
			t.Errorf("Got: - | Expected: +\n%s", cmp.Diff(&sensor, updated))
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
	sensor := Sensor{Name: "Test Sensor", UserId: u.Id}
	if err := sensor.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	// to ensure it is in the past
	mTime := time.Now().Add(time.Duration(-5) * time.Minute).Round(time.Microsecond)
	_, err = AssociateBatchToSensor(&batch, &sensor, "", &mTime, db)
	measurement := TemperatureMeasurement{CreatedBy: &u, UserId: u.Id, SensorId: sensor.Id,
		Temperature: 70.0, Units: FAHRENHEIT, RecordedAt: time.Now()}
	if err := measurement.Save(db); err != nil {
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

	t.Run("Save() New With All Fields", func(t *testing.T) {
		m := TemperatureMeasurement{CreatedBy: &u, UserId: u.Id, Sensor: &sensor, SensorId: sensor.Id,
			Temperature: 70.0, Units: FAHRENHEIT, RecordedAt: time.Now()}
		if err := m.Save(db); err != nil {
			t.Fatalf("%v", err)
		}
		if m.Id == "" {
			t.Errorf("Save() did not set id on new TemperatureMeasurement")
		}

		if m.UpdatedAt.IsZero() {
			t.Errorf("Save() did not set UpdatedAt")
		}

		if m.CreatedAt.IsZero() {
			t.Errorf("Save() did not set CreatedAt")
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

	t.Run("Save() New Without Optional Fields", func(t *testing.T) {
		m := TemperatureMeasurement{CreatedBy: &u, UserId: u.Id, SensorId: sensor.Id, Sensor: &sensor, Temperature: 70.0,
			Units: FAHRENHEIT, RecordedAt: time.Now()}
		if err := m.Save(db); err != nil {
			t.Fatalf("%v", err)
		}

		if m.Id == "" {
			t.Errorf("Save() did not set id on new TemperatureMeasurement")
		}

		if m.UpdatedAt.IsZero() {
			t.Errorf("Save() did not set UpdatedAt")
		}

		if m.CreatedAt.IsZero() {
			t.Errorf("Save() did not set CreatedAt")
		}

		newMeasurement := TemperatureMeasurement{}
		q := `SELECT tm.temperature, tm.units, tm.user_id, tm.sensor_id from temperature_measurements tm LEFT JOIN users u ON u.id = tm.user_id LEFT JOIN sensors ts ON ts.id = tm.sensor_id WHERE tm.id = ? AND tm.user_id = ? AND tm.sensor_id = ?`
		query := db.Rebind(q)
		err = db.Get(&newMeasurement, query, m.Id, u.Id, sensor.Id)

		if err != nil {
			t.Errorf("%v", err)
		}
	})

	t.Run("Save() existing", func(t *testing.T) {
		m := TemperatureMeasurement{UserId: u.Id, SensorId: sensor.Id, Temperature: 70.0, Units: FAHRENHEIT,
			RecordedAt: time.Now().Round(time.Microsecond)}
		if err := m.Save(db); err != nil {
			t.Fatalf("%v", err)
		}

		m.Temperature = 71.0
		if err := m.Save(db); err != nil {
			t.Fatalf("%v", err)
		}

		updated, err := FindTemperatureMeasurement(map[string]interface{}{"id": m.Id}, db)
		if err != nil {
			t.Errorf("%v", err)
		}

		// Not 100% sure IgnoreUnexported is the best way to go here. Mostly want to ignore m.batch, but this will
		// ignore other unexported things as well if they are added
		cmpOpts := []cmp.Option{
			cmpopts.IgnoreUnexported(m),
			// cmp.AllowUnexported(*m.batch),
		}
		if !cmp.Equal(&m, updated, cmpOpts...) {
			t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(&m, updated, cmpOpts...))
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

	var testmatrix = []struct {
		name     string
		inputs   map[string]interface{}
		expected []*Batch
	}{
		// basic filters
		// This is ok for now, but really don't want to write one test per potential filter as those grow
		// will at least add user uuid probably.
		{"Unfiltered", map[string]interface{}{}, []*Batch{&b, &b2, &u2batch}},
		{"By batch.Id", map[string]interface{}{"id": *b.Id}, []*Batch{&b}},
		{"By batch.Uuid", map[string]interface{}{"uuid": b.Uuid}, []*Batch{&b}},
		{"By batch.user_id", map[string]interface{}{"user_id": *u2.Id}, []*Batch{&u2batch}},
		// pagination
		{"Paginated no offset", map[string]interface{}{"limit": 1}, []*Batch{&b}},
		{"Paginated with offset", map[string]interface{}{"limit": 1, "offset": 1}, []*Batch{&b2}},
	}

	for _, tm := range testmatrix {
		t.Run(tm.name, func(t *testing.T) {
			batches, err := FindBatches(tm.inputs, db)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !cmp.Equal(tm.expected, batches) {
				t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(tm.expected, batches))
			}
		})
	}
}

// TODO: WRITE THESE NOW
func TestInsertBatch(t *testing.T) {}
func TestUpdateBatch(t *testing.T) {}
func TestSaveBatch(t *testing.T)   {}

func TestBatch(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Error setting up database: %s", err)
	}
	defer db.Close()

	u := User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	t.Run("Save() new batch", func(t *testing.T) {
		// TODO: FindBatch() will start joining the user and populating, at which point this also needs
		// CreatedBy: &u set.
		b := Batch{Name: "Testing", BrewedDate: time.Date(2019, time.January, 01, 12, 0, 0, 0, time.UTC),
			BottledDate: time.Date(2019, time.January, 24, 12, 0, 0, 0, time.UTC), VolumeBoiled: 5, VolumeInFermentor: 4.5,
			VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, UserId: u.Id, BrewNotes: "Brew notes",
			TastingNotes: "Taste notes", RecipeURL: "http://example.org/beer"}

		if err := b.Save(db); err != nil {
			t.Fatalf("Error inserting batch: %v", err)
		}
		if b.Id == nil {
			t.Errorf("Save() on new batch did not set an id")
		}

		if b.UpdatedAt.IsZero() {
			t.Errorf("Save() on new batch did not set UpdatedAt")
		}

		if b.CreatedAt.IsZero() {
			t.Errorf("Save() on new batch did not set CreatedAt")
		}

		if b.Uuid == "" {
			t.Errorf("Save() on new batch did not set Uuid")
		}

		batchArgs := make(map[string]interface{})
		batchArgs["user_id"] = u.Id
		batchArgs["id"] = b.Id
		found, err := FindBatch(batchArgs, db)
		if err != nil {
			t.Fatalf("Error looking up batch: %s", err)
		}
		if !cmp.Equal(&b, found) {
			t.Errorf("Expected: - | Got: +\n%v", cmp.Diff(&b, found))
		}
	})

	t.Run("uSave() update existing batch", func(t *testing.T) {
		// TODO: FindBatch() will start joining the user and populating, at which point this also needs
		// CreatedBy: &u set.
		b := Batch{Name: "Testing", BrewedDate: time.Date(2019, time.January, 01, 12, 0, 0, 0, time.UTC),
			BottledDate: time.Date(2019, time.January, 24, 12, 0, 0, 0, time.UTC), VolumeBoiled: 5, VolumeInFermentor: 4.5,
			VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, UserId: u.Id, BrewNotes: "Brew notes",
			TastingNotes: "Taste notes", RecipeURL: "http://example.org/beer"}
		// b := &_b

		if err := b.Save(db); err != nil {
			t.Fatalf("Error inserting batch: %v", err)
		}
		if b.Id == nil {
			t.Fatalf("Save() on new batch did not set an id")
		}

		// I am lazy. Change a few things, but not all of the things.
		b.Name = "Updated"
		b.TastingNotes = "Updated"
		b.BrewNotes = "Updated"
		b.RecipeURL = "https://example.org/updated"

		if err := b.Save(db); err != nil {
			t.Fatalf("Error updating batch: %v", err)
		}

		// if b.UpdatedAt == initialUpdatedAt {
		// TODO: I would like to do this test, but using github.com/DATA-DOG/go-txdb
		// both the initial insert and update are part of the same transaction, so get the same
		// updated_at set.  I could do it in golang code instead of the sql NOW() function.
		// 	t.Errorf("batch.Save() on existing batch did not update UpdatedAt")
		// }

		batchArgs := make(map[string]interface{})
		batchArgs["user_id"] = u.Id
		batchArgs["id"] = b.Id
		found, err := FindBatch(batchArgs, db)
		if err != nil {
			t.Fatalf("Error looking up batch: %s", err)
		}
		// Newly looked up one should have the updates made
		if !cmp.Equal(&b, found) {
			t.Errorf("Expected: - | Got: +\n%v", cmp.Diff(&b, found))
		}
	})
}

func TestBatchSensorAssociations(t *testing.T) {
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
	if err := batch.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	sensor := Sensor{Name: "Test Sensor", CreatedBy: &u}
	if err := sensor.Save(db); err != nil {
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
		association, err := AssociateBatchToSensor(&batch, &sensor, "Testing", nil, db)
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
		aPtr, err := AssociateBatchToSensor(&batch, &sensor, "Testing", nil, db)
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
		expected, err := FindBatchSensorAssociation(map[string]interface{}{"id": updated.Id}, db)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if expected.Description != "Updated" {
			t.Errorf("Expected: %s\nGot: %s. Changes may not have persisted to the database.", spew.Sdump(expected),
				spew.Sdump(updated))
		}
	})
}

func TestFindBatchSensorAssociations(t *testing.T) {
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
	if err := batch.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	batch2 := Batch{Name: "Testing2", UserId: u.Id, BrewedDate: brewedDate, BottledDate: bottledDate, VolumeBoiled: 5,
		VolumeInFermentor: 4.5, VolumeUnits: GALLON, OriginalGravity: 1.060, FinalGravity: 1.020, BrewNotes: "Brew Notes",
		TastingNotes: "Taste Notes", RecipeURL: "http://example.org/beer"}
	if err := batch2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	sensor := Sensor{Name: "Test Sensor", UserId: u.Id}
	if err := sensor.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	sensor2 := Sensor{Name: "Test Sensor2", UserId: u.Id}
	if err := sensor2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	association, err := AssociateBatchToSensor(&batch, &sensor, "Testing", nil, db)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	association2, err := AssociateBatchToSensor(&batch, &sensor2, "Testing 2", nil, db)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	var querytests = []struct {
		name     string
		inputs   map[string]interface{}
		expected []*BatchSensor
	}{
		// basic filters
		{"By batch.Id", map[string]interface{}{"batch_id": *batch.Id}, []*BatchSensor{association, association2}},
		{"By sensor.Id", map[string]interface{}{"sensor_id": *sensor.Id}, []*BatchSensor{association}},
		{"By batch2.Id", map[string]interface{}{"batch_id": *batch2.Id}, []*BatchSensor(nil)},
		{"By sensor2.Id", map[string]interface{}{"sensor_id": *sensor2.Id}, []*BatchSensor{association2}},
		// pagination
		{"Paginated no offset", map[string]interface{}{"limit": 1}, []*BatchSensor{association}},
		{"Paginated with offset", map[string]interface{}{"limit": 1, "offset": 1}, []*BatchSensor{association2}},
	}

	for _, qt := range querytests {
		t.Run(qt.name, func(t *testing.T) {
			associations, err := FindBatchSensorAssociations(qt.inputs, db)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !cmp.Equal(qt.expected, associations) {
				t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(qt.expected, associations))
			}
		})
	}
}

func TestFindTemperatureMeasurements(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	// TODO: must be a good way to shorten this setup model creation... function which takes count of
	// users to create, etc. I suppose.
	u := User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	u2 := User{Email: "user2@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = u2.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	s1 := Sensor{Name: "Test Sensor", UserId: u.Id}
	if err := s1.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	s2 := Sensor{Name: "Test Sensor", UserId: u2.Id}
	if err := s2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	// TODO: make batch, associate with sensor, and test
	b := makeTestBatch(&u, false)
	if err := b.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	m1 := TemperatureMeasurement{UserId: u.Id, SensorId: s1.Id, Temperature: 70.0, Units: FAHRENHEIT,
		RecordedAt: addMinutes(b.BrewedDate, -1)}
	if err := m1.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	m2 := TemperatureMeasurement{UserId: u.Id, SensorId: s1.Id, Temperature: 70.0, Units: FAHRENHEIT,
		RecordedAt: time.Now().Round(time.Microsecond)}
	if err := m2.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	m3 := TemperatureMeasurement{UserId: u2.Id, SensorId: s2.Id, Temperature: 71.0, Units: FAHRENHEIT,
		RecordedAt: time.Now().Round(time.Microsecond)}
	if err := m3.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	_, err = AssociateBatchToSensor(&b, &s1, "", &b.BrewedDate, db)
	if err != nil {
		t.Fatalf("%v", err)
	}

	var testmatrix = []struct {
		name     string
		inputs   map[string]interface{}
		expected []*TemperatureMeasurement
	}{
		// basic filters
		// This is ok for now, but really don't want to write one test per potential filter as those grow
		// will at least add user uuid probably.
		{"Unfiltered", map[string]interface{}{}, []*TemperatureMeasurement{&m1, &m2, &m3}},
		{"By m1.Id", map[string]interface{}{"id": m1.Id}, []*TemperatureMeasurement{&m1}},
		{"By sensor_Id", map[string]interface{}{"sensor_id": *s1.Id}, []*TemperatureMeasurement{&m1, &m2}},
		{"By sensor_uuid", map[string]interface{}{"sensor_uuid": s1.Uuid}, []*TemperatureMeasurement{&m1, &m2}},
		{"By m3.UserId", map[string]interface{}{"user_id": *u2.Id}, []*TemperatureMeasurement{&m3}},
		{"By batch_uuid with active sensor association", map[string]interface{}{"batch_uuid": b.Uuid}, []*TemperatureMeasurement{&m2}},
		// todo: add a batch_uuid test validating if the measurement is AFTER the disassociation
		// pagination
		{"Paginated no offset", map[string]interface{}{"limit": 1}, []*TemperatureMeasurement{&m1}},
		{"Paginated with offset", map[string]interface{}{"limit": 1, "offset": 1}, []*TemperatureMeasurement{&m2}},
	}

	for _, tm := range testmatrix {
		t.Run(tm.name, func(t *testing.T) {
			measurements, err := FindTemperatureMeasurements(tm.inputs, db)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			// Not 100% sure IgnoreUnexported is the best way to go here. Mostly want to ignore m.batch, but this will
			// ignore other unexported things as well if they are added
			cmpOpts := []cmp.Option{
				cmpopts.IgnoreUnexported(TemperatureMeasurement{}),
				// cmp.AllowUnexported(*m.batch),
			}
			if !cmp.Equal(tm.expected, measurements, cmpOpts...) {
				t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(tm.expected, measurements, cmpOpts...))
			}
		})
	}
}
