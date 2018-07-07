package graphqlApi

import (
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"testing"
	"time"
)

// Test that the schema parses, the same as is done at runtime when starting worrywortd.
// Any issues here would probably also be caught by integration tests on worrywortd ensuring
// http routing, responses, etc.
func TestUserResolver(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", createdAt, updatedAt)
	r := userResolver{u: u}

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
	createdAt := time.Now()
	updatedAt := time.Now()
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", createdAt, updatedAt)
	brewed := worrywort.NewBatch(1, "Testing", time.Now(), time.Now(), 5, 4.5, worrywort.GALLON, 1.060, 1.020, u,
		createdAt, updatedAt, "Brew notes", "Taste notes", "http://example.org/beer")
	unbrewed := worrywort.NewBatch(2, "Testing", time.Time{}, time.Time{}, 5, 4.5, worrywort.GALLON, 1.060, 1.020,
		worrywort.User{}, createdAt, updatedAt, "Brew notes", "Taste notes", "http://example.org/beer")
	br := batchResolver{b: brewed}
	unbr := batchResolver{b: unbrewed}

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

	t.Run("CreatedBy()", func(t *testing.T) {
		var actual *userResolver = br.CreatedBy()
		expected := userResolver{u: brewed.CreatedBy}
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})
}

func TestFermenterResolver(t *testing.T) {
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	f := worrywort.NewFermenter(1, "Ferm", "A Fermenter", 5.0, worrywort.GALLON, worrywort.BUCKET, true, true, u,
		time.Now(), time.Now())
	r := fermenterResolver{f: f}

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
		var actual *userResolver = r.CreatedBy()
		expected := userResolver{u: f.CreatedBy}
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})
}

func TestTemperatureSensorResolver(t *testing.T) {
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	therm := worrywort.NewTemperatureSensor(1, "Therm1", u, time.Now(), time.Now())
	r := temperatureSensorResolver{t: therm}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = r.ID()
		expected := graphql.ID("1")
		if ID != expected {
			t.Errorf("Expected: %v, got: %v", expected, ID)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var dt string = r.CreatedAt()
		expected := therm.CreatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("Expected: %v, got: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt string = r.UpdatedAt()
		expected := therm.UpdatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("Expected: %v, got %v", expected, dt)
		}
	})

	t.Run("CreatedBy()", func(t *testing.T) {
		var actual *userResolver = r.CreatedBy()
		expected := userResolver{u: therm.CreatedBy}
		if *actual != expected {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	})
}

func TestTemperatureMeasurementResolver(t *testing.T) {
	u := worrywort.NewUser(1, "user@example.com", "Justin", "Michalicek", time.Now(), time.Now())
	therm := worrywort.NewTemperatureSensor(1, "Therm1", u, time.Now(), time.Now())
	b := worrywort.NewBatch(1, "Testing", time.Now(), time.Now(), 5, 4.5, worrywort.GALLON, 1.060, 1.020, u,
		time.Now(), time.Now(), "Brew notes", "Taste notes", "http://example.org/beer")
	f := worrywort.NewFermenter(1, "Ferm", "A Fermenter", 5.0, worrywort.GALLON, worrywort.BUCKET, true, true, u,
		time.Now(), time.Now())
	timeRecorded := time.Now().Add(time.Hour * time.Duration(-1))
	m := worrywort.NewTemperatureMeasurement(
		"shouldbeauuid", 64.26, worrywort.FAHRENHEIT, b, therm, f, timeRecorded, time.Now(), time.Now(), u)
	r := temperatureMeasurementResolver{m: m}

	t.Run("ID()", func(t *testing.T) {
		var ID graphql.ID = r.ID()
		expected := graphql.ID(m.Id)
		if ID != expected {
			t.Errorf("\nExpected: %v\ngot: %v", expected, ID)
		}
	})

	t.Run("CreatedAt()", func(t *testing.T) {
		var dt string = r.CreatedAt()
		expected := m.CreatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("\nExpected: %v\ngot: %v", expected, dt)
		}
	})

	t.Run("UpdatedAt()", func(t *testing.T) {
		var dt string = r.UpdatedAt()
		expected := m.UpdatedAt.Format(time.RFC3339)
		if dt != expected {
			t.Errorf("\nExpected: %v\ngot %v", expected, dt)
		}
	})

	t.Run("CreatedBy()", func(t *testing.T) {
		var actual *userResolver = r.CreatedBy()
		expected := userResolver{u: m.CreatedBy}
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
