package graphqlApi

import (
	"context"
	"database/sql"
	"github.com/davecgh/go-spew/spew"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"reflect"
	"testing"
	"time"
)

func setUpTestDb() (*sqlx.DB, error) {
	_db, err := sql.Open("txdb", "one")
	if err != nil {
		return nil, err
	}

	db := sqlx.NewDb(_db, "postgres")
	if err != nil {
		return nil, err
	}

	return db, nil
}

// utility to add a given number of minutes to a time.Time and round to match
// what postgres returns
func addMinutes(d time.Time, increment int) time.Time {
	return d.Add(time.Duration(increment) * time.Minute).Round(time.Microsecond)
}

// Make a standard, generic batch for testing
// optionally attach the user
func makeTestBatch(u worrywort.User, attachUser bool) worrywort.Batch {
	b := worrywort.Batch{Name: "Testing", BrewedDate: addMinutes(time.Now(), 1), BottledDate: addMinutes(time.Now(), 10), VolumeBoiled: 5,
		VolumeInFermenter: 4.5, VolumeUnits: worrywort.GALLON, OriginalGravity: 1.060, FinalGravity: 1.020,
		UserId: sql.NullInt64{Int64: int64(u.Id), Valid: true}, BrewNotes: "Brew notes",
		TastingNotes: "Taste notes", RecipeURL: "http://example.org/beer"}
	if attachUser {
		b.CreatedBy = &u
	}
	return b
}

func TestUserResolver(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	createdAt := time.Now()
	updatedAt := time.Now()
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", createdAt, updatedAt)
	r := userResolver{u: &u}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = r.ID()
		expected := graphql.ID("1")
		if ID != expected {
			t.Errorf("Expected: %v, got: %v", expected, ID)
		}
	})

	t.Run("FirstName()", func(t *testing.T) {
		var firstName string = r.FirstName()
		expected := "Justin"
		if firstName != expected {
			t.Errorf("Expected: %v, got: %v", expected, firstName)
		}
	})

	t.Run("LastName()", func(t *testing.T) {
		var lastName string = r.LastName()
		expected := "Michalicek"
		if lastName != expected {
			t.Errorf("Expected: %v, got: %v", expected, lastName)
		}
	})

	t.Run("Email()", func(t *testing.T) {
		var email string = r.Email()
		expected := "user@example.com"
		if email != expected {
			t.Errorf("Expected: %v, got: %v", expected, email)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var dt string = r.CreatedAt()
		expected := u.CreatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt string = r.UpdatedAt()
		expected := u.UpdatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("Expected: %v, got %v", expected, dt)
		}
	})
}

func TestBatchResolver(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	u, err := worrywort.SaveUser(db, worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"})
	brewed := makeTestBatch(u, true)
	brewed.Id = 1
	unbrewed := makeTestBatch(u, true)
	unbrewed.BrewedDate = time.Time{}
	unbrewed.BottledDate = time.Time{}

	br := batchResolver{b: &brewed}
	unbr := batchResolver{b: &unbrewed}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = br.ID()
		expected := graphql.ID("1")
		if ID != expected {
			t.Errorf("Expected: %v, got: %v", expected, ID)
		}
	})

	t.Run("Name()", func(t *testing.T) {
		var name string = br.Name()
		expected := "Testing"
		if name != expected {
			t.Errorf("Expected: %v, got: %v", expected, name)
		}
	})

	t.Run("BrewNotes()", func(t *testing.T) {
		var notes string = br.BrewNotes()
		expected := "Brew notes"
		if notes != expected {
			t.Errorf("Expected: %v, got: %v", expected, notes)
		}
	})

	t.Run("TastingNotes()", func(t *testing.T) {
		var notes string = br.TastingNotes()
		expected := "Taste notes"
		if notes != expected {
			t.Errorf("Expected: %v, got: %v", expected, notes)
		}
	})

	t.Run("BrewedDate()", func(t *testing.T) {
		var dt *string = br.BrewedDate()
		expected := brewed.BrewedDate.Format(time.RFC3339)
		if *dt != expected {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}

		unbrewedDate := unbr.BrewedDate()
		if unbrewedDate != nil {
			t.Errorf("Expected: nil but got %v", unbrewedDate)
		}
	})

	t.Run("BottledDate()", func(t *testing.T) {
		var dt *string = br.BottledDate()
		expected := brewed.BottledDate.Format(time.RFC3339)
		if *dt != expected {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}

		unbrewedDate := unbr.BottledDate()
		if unbrewedDate != nil {
			t.Errorf("Expected: nil but got %v", unbrewedDate)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var dt string = br.CreatedAt()
		expected := brewed.CreatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt string = br.UpdatedAt()
		expected := brewed.UpdatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("Expected: %v, got %v", expected, dt)
		}
	})

	t.Run("VolumeBoiled()", func(t *testing.T) {
		var actual *float64 = br.VolumeBoiled()
		expected := brewed.VolumeBoiled
		// direct comparison seems to be ok, probably since no math is happening
		// but may be better to do like this:
		// if math.Abs(*boiled - expected) > .0000000001 {
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("VolumeInFermenter()", func(t *testing.T) {
		var actual *float64 = br.VolumeInFermenter()
		expected := brewed.VolumeInFermenter
		// direct comparison seems to be ok, probably since no math is happening
		// but may be better to do like this:
		// if math.Abs(*boiled - expected) > .0000000001 {
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("OriginalGravity()", func(t *testing.T) {
		var actual *float64 = br.OriginalGravity()
		expected := brewed.OriginalGravity
		// direct comparison seems to be ok, probably since no math is happening
		// but may be better to do like this:
		// if math.Abs(*boiled - expected) > .0000000001 {
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("FinalGravity()", func(t *testing.T) {
		var actual *float64 = br.FinalGravity()
		expected := brewed.FinalGravity
		// direct comparison seems to be ok, probably since no math is happening
		// but may be better to do like this:
		// if math.Abs(*boiled - expected) > .0000000001 {
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("VolumeUnits()", func(t *testing.T) {
		var actual worrywort.VolumeUnitType = br.VolumeUnits()
		expected := brewed.VolumeUnits

		if actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("RecipeURL()", func(t *testing.T) {
		var actual string = br.RecipeURL()
		expected := brewed.RecipeURL

		if actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("CreatedBy() with User struct populated", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "db", db)

		actual, err := br.CreatedBy(ctx)
		if err != nil {
			t.Errorf("%v", err)
		}
		expected := userResolver{u: brewed.CreatedBy}
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("CreatedBy() without User populated", func(t *testing.T) {
		batchNoUser := makeTestBatch(u, false)
		batchNoUser, err = worrywort.SaveBatch(db, batchNoUser)
		if err != nil {
			t.Fatalf("%v", err)
		}
		ctx := context.Background()
		ctx = context.WithValue(ctx, "db", db)
		r := batchResolver{b: &batchNoUser}
		actual, err := r.CreatedBy(ctx)
		if err != nil {
			t.Errorf("%v", err)
		}
		expected := &userResolver{u: &u}

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})
}

func TestFermenterResolver(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)

	u, err := worrywort.SaveUser(db, worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"})
	userId := sql.NullInt64{Valid: true, Int64: int64(u.Id)}
	f := worrywort.Fermenter{Id: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: "Ferm", Description: "A Fermenter", Volume: 5.0, VolumeUnits: worrywort.GALLON,
		FermenterType: worrywort.BUCKET, IsActive: true, IsAvailable: true, CreatedBy: &u, UserId: userId}
	r := fermenterResolver{f: &f}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = r.ID()
		expected := graphql.ID("1")
		if ID != expected {
			t.Errorf("Expected: %v, got: %v", expected, ID)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var dt string = r.CreatedAt()
		expected := f.CreatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt string = r.UpdatedAt()
		expected := f.UpdatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("Expected: %v, got %v", expected, dt)
		}
	})

	t.Run("CreatedBy()", func(t *testing.T) {

		var actual *userResolver = r.CreatedBy(ctx)
		expected := userResolver{u: f.CreatedBy}
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("CreatedBy() without User populated", func(t *testing.T) {
		var f2 worrywort.Fermenter = f
		f2.CreatedBy = nil
		r := fermenterResolver{f: &f2}
		actual := r.CreatedBy(ctx)
		expected := &userResolver{u: &u}

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})
}

func TestTemperatureSensorResolver(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)

	u, err := worrywort.SaveUser(db, worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"})
	userId := sql.NullInt64{Valid: true, Int64: int64(u.Id)}
	sensor := worrywort.NewTemperatureSensor(1, "Therm1", &u, time.Now(), time.Now())
	sensor.UserId = userId
	r := temperatureSensorResolver{t: &sensor}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = r.ID()
		expected := graphql.ID("1")
		if ID != expected {
			t.Errorf("Expected: %v, got: %v", expected, ID)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var dt string = r.CreatedAt()
		expected := sensor.CreatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt string = r.UpdatedAt()
		expected := sensor.UpdatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("Expected: %v, got %v", expected, dt)
		}
	})

	t.Run("CreatedBy()", func(t *testing.T) {
		var actual *userResolver = r.CreatedBy(ctx)
		expected := userResolver{u: sensor.CreatedBy}
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})

	t.Run("CreatedBy() without User populated", func(t *testing.T) {
		var s2 worrywort.TemperatureSensor = sensor
		s2.CreatedBy = nil
		s2.Id = 0
		s2, err = worrywort.SaveTemperatureSensor(db, s2)
		if err != nil {
			t.Fatalf("%v", err)
		}
		r2 := temperatureSensorResolver{t: &s2}
		actual := r2.CreatedBy(ctx)
		expected := &userResolver{u: &u}

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("Expected: %s\nGot: %s", spew.Sdump(expected), spew.Sdump(actual))
		}
	})
}

func TestTemperatureMeasurementResolver(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)

	u, err := worrywort.SaveUser(db, worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"})
	userId := sql.NullInt64{Valid: true, Int64: int64(u.Id)}
	sensor := worrywort.NewTemperatureSensor(1, "Therm1", &u, time.Now(), time.Now())
	batch := makeTestBatch(u, true)
	fermenter := worrywort.NewFermenter(1, "Ferm", "A Fermenter", 5.0, worrywort.GALLON, worrywort.BUCKET, true, true, u,
		time.Now(), time.Now())
	timeRecorded := time.Now().Add(time.Hour * time.Duration(-1))
	measurement := worrywort.TemperatureMeasurement{Id: "shouldbeauuid", Temperature: 64.26, Units: worrywort.FAHRENHEIT, RecordedAt: timeRecorded,
		Batch: &batch, TemperatureSensor: &sensor, Fermenter: &fermenter, CreatedBy: &u, UserId: userId, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	resolver := temperatureMeasurementResolver{m: &measurement}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = resolver.ID()
		expected := graphql.ID(measurement.Id)
		if ID != expected {
			t.Errorf("\nExpected: %v\ngot: %v", expected, ID)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var dt string = resolver.CreatedAt()
		expected := measurement.CreatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("\nExpected: %v\ngot: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt string = resolver.UpdatedAt()
		expected := measurement.UpdatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("\nExpected: %v\ngot %v", expected, dt)
		}
	})

	t.Run("Temperature()", func(t *testing.T) {
		temp := resolver.Temperature()
		if measurement.Temperature != temp {
			t.Errorf("\nExpected: %v\ngot: %v", measurement.Temperature, temp)
		}
	})

	t.Run("Units()", func(t *testing.T) {
		units := resolver.Units()
		if measurement.Units != units {
			t.Errorf("\nExpected: %v\ngot: %v", measurement.Units, units)
		}
	})

	t.Run("Batch()", func(t *testing.T) {

		b := resolver.Batch(ctx)
		expected := batchResolver{b: measurement.Batch}
		if expected != *b {
			t.Errorf("\nExpected: %v\ngot: %v", expected, *b)
		}
	})

	t.Run("Fermenter()", func(t *testing.T) {
		f := resolver.Fermenter(ctx)
		expected := fermenterResolver{f: measurement.Fermenter}
		if expected != *f {
			t.Errorf("\nExpected: %v\ngot: %v", expected, *f)
		}
	})

	t.Run("TemperatureSensor()", func(t *testing.T) {
		ts := resolver.TemperatureSensor(ctx)
		expected := temperatureSensorResolver{t: measurement.TemperatureSensor}
		if expected != *ts {
			t.Errorf("\nExpected: %v\ngot: %v", expected, ts)
		}
	})

	t.Run("CreatedBy() with User attached", func(t *testing.T) {
		// TODO: This test with user not already populated
		var actual *userResolver = resolver.CreatedBy(ctx)
		expected := userResolver{u: measurement.CreatedBy}
		if *actual != expected {
			t.Errorf("\nExpected: %v\ngot %v", expected, actual)
		}
	})

}

func TestAuthTokenResolver(t *testing.T) {
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	token := worrywort.NewToken("tokenid", "token", u, worrywort.TOKEN_SCOPE_ALL)
	r := authTokenResolver{t: token}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = r.ID()
		expected := graphql.ID(token.ForAuthenticationHeader())
		if ID != expected {
			t.Errorf("\nExpected: %v\ngot: %v", expected, ID)
		}
	})

	t.Run("Token()", func(t *testing.T) {
		var tokenStr string = r.Token()
		expected := token.ForAuthenticationHeader()
		if tokenStr != expected {
			t.Errorf("\nExpected: %v\ngot: %v", expected, tokenStr)
		}
	})
}
