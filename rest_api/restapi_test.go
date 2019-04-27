package rest_api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	txdb "github.com/DATA-DOG/go-txdb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	// "github.com/davecgh/go-spew/spew"
	"github.com/jmichalicek/worrywort-server-go/middleware"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	// "log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// Could use the same struct as I serialize with, but this is honestly simpler for now
type successResponse struct {
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
	Units       string    `json:"units"`
	SensorId    string    `json:"sensor_id"`
	UserId      string    `json:"user_id"`
	Id          string    `json:"id"`
	Temperature float64   `json:"temperature"`
	RecordedAt  time.Time `json:"recorded_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type errorResponse struct {
}

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

func TestMeasurementHandler(t *testing.T) {
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

	handler := MeasurementHandler{Db: db}

	t.Run("GET", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "", nil)
		ctx := req.Context()
		ctx = context.WithValue(ctx, middleware.DefaultUserKey, &user)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("/measurements page didn't return %v. Returned %v", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("POST valid", func(t *testing.T) {
		form := url.Values{}
		form.Add("value", "65.2")
		form.Add("metric", "temperature")
		form.Add("sensor_id", sensor.UUID)
		form.Add("units", "FAHRENHEIT")
		form.Add("recorded_at", "2019-04-21T11:30:33.32838Z")

		// TODO: something here is not working, it is not adding the values.
		req, _ := http.NewRequest("POST", "", strings.NewReader(form.Encode()))
		req.PostForm = form
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		ctx := req.Context()
		ctx = context.WithValue(ctx, middleware.DefaultUserKey, &user)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("Home page didn't return 201, returned %v", w.Code)
		}

		// TODO: make sure it saved, validate the response
		cmpOpts := []cmp.Option{
			// TODO: write proper custom validators for these which make sure they at least have data
			cmpopts.IgnoreFields(successResponse{}, "CreatedAt", "UpdatedAt", "Id"),
		}
		target := &successResponse{}
		recordedAtResponse, _ := time.Parse(time.RFC3339, "2019-04-21T11:30:33.32838Z")
		expectedResponse := &successResponse{Temperature: 65.2, SensorId: sensor.UUID, Units: "FAHRENHEIT",
			RecordedAt: recordedAtResponse, UserId: user.UUID}
		json.NewDecoder(w.Body).Decode(target)
		if !cmp.Equal(expectedResponse, target, cmpOpts...) {
			t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(expectedResponse, target, cmpOpts...))
		}

		// TODO: FindTemperatureMeasurement will eventually add the Sensor and CreatedBy onto the response for
		// FindTemperatureMeasurement at which point this test needs a quickie update
		expectedMeasurement := &worrywort.TemperatureMeasurement{Id: target.Id, SensorId: sensor.Id, UserId: user.Id,
			Temperature: 65.2, Units: worrywort.FAHRENHEIT, RecordedAt: target.RecordedAt, UpdatedAt: target.UpdatedAt,
			CreatedAt: target.CreatedAt}
		if m, err := worrywort.FindTemperatureMeasurement(
			map[string]interface{}{"uuid": target.Id, "sensor_id": *sensor.Id, "user_id": *user.Id}, db); err != nil {
			t.Fatalf("Expected TemperatureMeasurement not found in database: %v", err)
		} else {
			cmpOpts := []cmp.Option{
				// TODO: write proper custom validators for these which make sure they at least have data
				// IgnoreUnexported is for `batch`
				cmpopts.IgnoreFields(worrywort.TemperatureMeasurement{}, "Id"),
				cmpopts.IgnoreUnexported(worrywort.TemperatureMeasurement{}),
			}
			if !cmp.Equal(expectedMeasurement, m, cmpOpts...) {
				t.Errorf("Expected: - | Got: +\n%s", cmp.Diff(expectedMeasurement, m, cmpOpts...))
			}
		}
	})

	t.Run("POST unauthenticated", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("Home page didn't return 401, returned %v", w.Code)
		}
	})

	t.Run("POST errors", func(t *testing.T) {
		// for testing post see
		// http://markjberger.com/testing-web-apps-in-golang/
		// req, _ := http.NewRequest("POST", "", nil)
		form := url.Values{}
		form.Add("value", "asdf")
		form.Add("metric", "foobar")
		form.Add("sensor_id", "")
		// TODO: add user to the request context
		req, _ := http.NewRequest("POST", "", strings.NewReader(form.Encode()))
		req.PostForm = form
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		ctx := req.Context()
		ctx = context.WithValue(ctx, middleware.DefaultUserKey, &user)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Home page didn't return 201, returned %v", w.Code)
		}

		// TODO: make sure it did not save, validate the response errors
	})
}
