package restapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	txdb "github.com/DATA-DOG/go-txdb"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	dbUser, _ := os.LookupEnv("DATABASE_USER")
	dbPassword, _ := os.LookupEnv("DATABASE_PASSWORD")
	dbHost, _ := os.LookupEnv("DATABASE_HOST")
	// we register an sql driver txdb
	connString := fmt.Sprintf("host=%s port=5432 user=%s dbname=worrywort_test sslmode=disable", dbHost,
		dbUser)
	if dbPassword != "" {
		connString += fmt.Sprintf(" password=%s", dbPassword)
	}
	txdb.Register("txdb", "postgres", connString)
	retCode := m.Run()
	os.Exit(retCode)
}

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

func TestMeasurementSerializer(t *testing.T) {
	db, err := setUpTestDb()
	if err != nil {
		t.Fatalf("Got error setting up database: %s", err)
	}
	defer db.Close()

	// TODO: must be a good way to shorten this setup model creation... function which takes count of
	// users to create, etc. I suppose.
	user := worrywort.User{Email: "user@example.com", FirstName: "Justin", LastName: "Michalicek"}
	err = user.Save(db)
	if err != nil {
		t.Fatalf("failed to insert user: %s", err)
	}

	sensor := worrywort.Sensor{Name: "Test Sensor", UserId: user.Id}
	if err := sensor.Save(db); err != nil {
		t.Fatalf("%v", err)
	}

	t.Run("MarshalJSON()", func(t *testing.T) {
		recordedAt := time.Now().Add(time.Duration(-1) * time.Minute).Round(time.Microsecond)
		m := &worrywort.TemperatureMeasurement{
			Temperature: 65.2,
			Units:       worrywort.FAHRENHEIT,
			SensorId:    sensor.Id,
			Sensor:      &sensor,
			CreatedBy:   &user,
			UserId:      user.Id,
			RecordedAt:  recordedAt,
		}
		m.Save(db)
		s := TemperatureMeasurementSerializer{TemperatureMeasurement: m}
		jsonStr, err := s.MarshalJSON()
		if err != nil {
			t.Fatalf("%v", err)
		}
		// {
		//   "units": "FAHRENHEIT",
		//   "sensor_id": "bc69f698-de5a-46fe-8847-78007aa73b05",
		//   "user_id": "970aaedc-1bef-487f-92dc-425557fe68a3",
		//   "id": "35d46438-0fe3-4134-86a3-3899d449ef6a",
		//   "temperature": 65.2,
		//   "recorded_at": "2019-04-21T18:54:33.32838Z",
		//   "created_at": "2019-04-22T01:09:38.416851Z",
		//   "updated_at": "2019-04-22T01:09:38.416851Z"
		// }

		type expected struct {
			Units       string    `json:"units"`
			Temperature float64   `json:"temperature"`
			SensorId    string    `json:"sensor_id"`
			UserId      string    `json:"user_id"`
			Id          string    `json:"id"`
			RecordedAt  time.Time `json:"recorded_at"`
			CreatedAt   time.Time `json:"created_at"`
			UpdatedAt   time.Time `json:"updated_at"`
		}
		unmarshalled := new(expected)
		err = json.Unmarshal(jsonStr, unmarshalled)
		if err != nil {
			t.Fatalf("%v", err)
		}
	})

	t.Run("UnitString()", func(t *testing.T) {
		recordedAt := time.Now().Add(time.Duration(-1) * time.Minute).Round(time.Microsecond)
		m := &worrywort.TemperatureMeasurement{
			Temperature: 65.2,
			Units:       worrywort.FAHRENHEIT,
			SensorId:    sensor.Id,
			Sensor:      &sensor,
			CreatedBy:   &user,
			UserId:      user.Id,
			RecordedAt:  recordedAt,
		}
		s := TemperatureMeasurementSerializer{TemperatureMeasurement: m}

		testmatrix := []struct {
			unitType worrywort.TemperatureUnitType
			expected string
		}{
			{worrywort.FAHRENHEIT, "FAHRENHEIT"},
			{worrywort.CELSIUS, "CELSIUS"},
		}
		for _, tm := range testmatrix {
			t.Run(tm.expected, func(t *testing.T) {
				s.Units = tm.unitType
				actual := s.UnitString()
				if actual != tm.expected {
					t.Errorf("Expected: %s\nGot: %s\n", tm.expected, actual)
				}
			})
		}
	})
}
