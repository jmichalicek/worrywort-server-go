package rest_api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	// "github.com/davecgh/go-spew/spew"
	"github.com/jmichalicek/worrywort-server-go/middleware"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	// "github.com/google/uuid"
)

// TODO: Not sure this is how I want to organize the code, but it'll work for now.
// doesn't feel very versioning friendly, etc. though.

var temperatureUnits = map[string]worrywort.TemperatureUnitType{
	"FAHRENHEIT": worrywort.FAHRENHEIT,
	"CELSIUS":    worrywort.CELSIUS,
}

type TemperatureMeasurementSerializer struct {
	*worrywort.TemperatureMeasurement
}

func (t *TemperatureMeasurementSerializer) UnitString() string {
	for k, v := range temperatureUnits {
		if v == t.Units {
			return k
		}
	}
	return ""
}

type TemperatureMeasurementForm struct {
	valid bool

	// Store good values - will have to rethink this once I add in other measurement types, such as hydrometer readings
	// might be able to make a reasonable interface or something.
	// May be smart to make this private and use a receiver to get at it
	CleanedMeasurement *worrywort.TemperatureMeasurement `json:"-"`

	// Store Errors
	UnitsErrors      []string `json:"units"`
	MetricErrors     []string `json:"metric"`
	SensorIdErrors   []string `json:"sensor_id"`
	RecordedAtErrors []string `json:"recorded_at"`
	ValueErrors      []string `json:"value"`

	// not a big fan of this, but for form validation...
	// would prefer a nice way of putting the valid values or how to query for them into the form
	// but that would require a bunch of extra nonsense which is not worthwhile for a specific use case
	// ie. pass in a sqrl SELECT/set as a struct member and then execute it in the validation
	user *worrywort.User
	db   *sqlx.DB
}

func (f *TemperatureMeasurementForm) IsValid() bool {
	// I may want to change this up and avoid having to set f.valid manually everywhere
	// so use this function. Plus keeping it private avoids other things mucking with it and doing dumb stuff.
	return f.valid
}

// Validate values submitted on the form.
// May make db part of the struct instead...
// may also just put the values right on the struct, not pass in here
func (f *TemperatureMeasurementForm) Validate(values url.Values) {
	// Should I make this case insensitive in some manner?
	sensorUUID := values.Get("sensor_id")
	metric := strings.ToLower(values.Get("metric"))
	val := values.Get("value")
	units := strings.ToUpper(values.Get("units"))
	timestamp := values.Get("recorded_at")

	// Should probably break this up, but whatever for now
	isValid := true
	if metric != "temperature" {
		isValid = false
		f.MetricErrors = append(f.MetricErrors, fmt.Sprintf("%s is not a known metric", metric))
	}

	if recordedAt, err := time.Parse(time.RFC3339, timestamp); err == nil {
		f.CleanedMeasurement.RecordedAt = recordedAt
	} else {
		isValid = false
		f.RecordedAtErrors = append(f.RecordedAtErrors, "recorded_at must be a valid RFC3339 timestamp")
	}

	if sensor, err := worrywort.FindSensor(map[string]interface{}{"uuid": sensorUUID, "user_id": *f.user.Id}, f.db); err == nil {
		f.CleanedMeasurement.Sensor = sensor
		f.CleanedMeasurement.SensorId = sensor.Id
	} else {
		isValid = false
		if err != sql.ErrNoRows {
			log.Printf("%v", err)
		}
		f.SensorIdErrors = append(f.SensorIdErrors, "Invalid sensor_id")
	}

	if temperature, err := strconv.ParseFloat(val, 64); err == nil {
		f.CleanedMeasurement.Temperature = temperature
	} else {
		// This error could be better, as there are things which are technically numbers which would not work
		f.ValueErrors = append(f.ValueErrors, "Temperature must be a number")
		isValid = false
	}

	if unitType, ok := temperatureUnits[units]; ok {
		f.CleanedMeasurement.Units = unitType
	} else {
		isValid = false
		f.UnitsErrors = append(f.UnitsErrors, "%s is not a valid unit")
	}
	f.valid = isValid
}

func (t *TemperatureMeasurementSerializer) MarshalJSON() ([]byte, error) {
	// type Copy TemperatureMeasurementRest
	return json.Marshal(&struct {
		Units    string `json:"units"`
		SensorId string `json:"sensor_id"`
		UserId   string `db:"user_id" json:"user_id"`
		*worrywort.TemperatureMeasurement
		// *Copy // not sure this is necessary for me. comes from https://ashleyd.ws/custom-json-marshalling-in-golang/index.html
	}{
		Units:                  t.UnitString(),
		SensorId:               t.Sensor.UUID, // a problem if we do not actually have it but this is currently used in a place where we always do
		UserId:                 t.CreatedBy.UUID,
		TemperatureMeasurement: t.TemperatureMeasurement,
		// Copy:      (*Copy)(t),
	})
}

type MeasurementHandler struct {
	Db *sqlx.DB
}

func (h *MeasurementHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: put this log config somewhere central. really need something to encapsulate
	// all routing for setting up graphql plus the new REST stuff, etc.
	// Kind of started in server.go, but that really belongs elsewhere
	log.SetFlags(log.LstdFlags | log.Llongfile)
	ctx := r.Context()
	u, err := middleware.UserFromContext(ctx)
	if u == nil || err != nil {
		// This is actually handled by some middleware before we ever get here, but playing it safe.
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case "POST":
		h.InsertMeasurement(w, r, u)
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

}

func (h *MeasurementHandler) InsertMeasurement(w http.ResponseWriter, r *http.Request, user *worrywort.User) {
	db := h.Db
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("%s", err), http.StatusBadRequest)
		return
	}

	form := TemperatureMeasurementForm{CleanedMeasurement: &worrywort.TemperatureMeasurement{CreatedBy: user, UserId: user.Id}, db: db, user: user}
	form.Validate(r.Form)
	if !form.IsValid() {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(form); err != nil {
			panic(err)
		}
		return
	}

	m := form.CleanedMeasurement
	if err := m.Save(db); err != nil {
		log.Printf("%v", err)
		http.Error(w, "Error saving measurement", http.StatusInternalServerError)
		return
	}
	serializer := &TemperatureMeasurementSerializer{m}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(serializer); err != nil {
		panic(err)
	}
	return
}
