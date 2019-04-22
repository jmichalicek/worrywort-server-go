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

var temperatureUnits = map[string]worrywort.TemperatureUnitType{
	"FAHRENHEIT": worrywort.FAHRENHEIT,
	"CELSIUS":    worrywort.CELSIUS,
}

type TemperatureMeasurementRestSerializer struct {
	*worrywort.TemperatureMeasurement
}

func (t *TemperatureMeasurementRestSerializer) UnitString() string {
	for k, v := range temperatureUnits {
		if v == t.Units {
			return k
		}
	}
	return ""
}

func (t *TemperatureMeasurementRestSerializer) MarshalJSON() ([]byte, error) {
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
		// db, ok := ctx.Value("db").(*sqlx.DB)
		// if !ok {
		// 	// TODO: logging with stack info?
		// 	log.Printf("No database in context")
		// 	http.Error(w, "Server Error", http.StatusInternalServerError)
		// }
		db := h.Db
		if err := r.ParseForm(); err != nil {
			// fmt.Fprintf(w, "ParseForm() err: %v", err)
			http.Error(w, fmt.Sprintf("%s", err), http.StatusBadRequest)
			return
		}
		// fmt.Printf("%s", spew.Sdump(r.Form))
		sensorUUID := r.FormValue("sensor_id")
		metric := r.FormValue("metric")
		val := r.FormValue("value")
		units := r.FormValue("units")
		timestamp := r.FormValue("recorded_at")
		switch metric {
		case "temperature":

			recordedAt, err := time.Parse(time.RFC3339, timestamp)
			if err != nil {
				// TODO: See what the actual error types are and try to return friendlier errors which are not golang specific messaging
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

			serializer := &TemperatureMeasurementRestSerializer{m}

			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(serializer); err != nil {
				panic(err)
			}
			return
		default:
			http.Error(w, "Unsupported metric", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}
