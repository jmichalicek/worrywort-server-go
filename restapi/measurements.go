package restapi

import (
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"net/http"
	// "github.com/davecgh/go-spew/spew"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jmichalicek/worrywort-server-go/authMiddleware"
	"github.com/jmoiron/sqlx"
	"log"
	"strconv"
	"time"
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
	h.InsertMeasurement(w, r)
}

func (h *MeasurementHandler) InsertMeasurement(w http.ResponseWriter, r *http.Request) {
	// TODO: put this log config somewhere central. really need something to encapsulate
	// all routing for setting up graphql plus the new REST stuff, etc.
	// Kind of started in server.go, but that really belongs elsewhere
	log.SetFlags(log.LstdFlags | log.Llongfile)

	// TODO: better error handling - send back as proper json response with all field errors, not one at a time
	switch r.Method {
	case "POST":
		ctx := r.Context()
		user, _ := authMiddleware.UserFromContext(ctx)
		db := h.Db
		if err := r.ParseForm(); err != nil {
			http.Error(w, fmt.Sprintf("%s", err), http.StatusBadRequest)
			return
		}
		// fmt.Printf("%s", spew.Sdump(r.Form))
		sensorUUID := r.FormValue("sensor_id")
		metric := r.FormValue("metric")
		val := r.FormValue("value")
		units := r.FormValue("units")
		timestamp := r.FormValue("recorded_at")

		formErrors := map[string][]string{}
		// have to change up testing for metric to fit in with the error handling nicely.
		switch metric {
		case "temperature":

			recordedAt, err := time.Parse(time.RFC3339, timestamp)
			if err != nil {
				_, ok := formErrors["recorded_at"]
				if !ok {
					formErrors["recorded_at"] = []string{}
				}
				formErrors["recorded_at"] = append(formErrors["recorded_at"], "Error parsing recorded_at time")
				http.Error(w, "Error parsing recorded_at time", http.StatusBadRequest)
				return
			}
			sensor, err := worrywort.FindSensor(map[string]interface{}{"uuid": sensorUUID, "user_id": *user.Id}, db)
			if err != nil {
				if err != sql.ErrNoRows {
					log.Printf("%v", err)
				}
				http.Error(w, "Error looking up sensor", http.StatusBadRequest)
				return
			}
			temperature, err := strconv.ParseFloat(val, 64)
			if err != nil {
				http.Error(w, "Bad temperature value", http.StatusBadRequest)
				return
			}

			if len(formErrors) == 0 {
				unitType := temperatureUnits[units]
				m := &worrywort.TemperatureMeasurement{
					Temperature: float64(temperature),
					Units:       unitType,
					SensorId:    sensor.Id,
					Sensor:      sensor,
					CreatedBy:   user,
					UserId:      user.Id,
					RecordedAt:  recordedAt,
				}
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
		default:
			http.Error(w, "Unsupported metric", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}
