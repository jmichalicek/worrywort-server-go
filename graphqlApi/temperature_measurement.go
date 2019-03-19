package graphqlApi

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"log"
)

// Resolve a worrywort.TemperatureMeasurement
type temperatureMeasurementResolver struct {
	// m for measurement
	m *worrywort.TemperatureMeasurement
}

func (r *temperatureMeasurementResolver) ID() graphql.ID {
	if r.m == nil {
		log.Printf("temperatureMeasurement with nil id: %v", spew.Sdump(r))
		return graphql.ID("")
	}
	return graphql.ID(r.m.Id)
}
func (r *temperatureMeasurementResolver) CreatedAt() string                    { return dateString(r.m.CreatedAt) }
func (r *temperatureMeasurementResolver) UpdatedAt() string                    { return dateString(r.m.UpdatedAt) }
func (r *temperatureMeasurementResolver) RecordedAt() string                   { return dateString(r.m.RecordedAt) }
func (r *temperatureMeasurementResolver) Temperature() float64                 { return r.m.Temperature }
func (r *temperatureMeasurementResolver) Units() worrywort.TemperatureUnitType { return r.m.Units }
func (r *temperatureMeasurementResolver) Batch(ctx context.Context) *batchResolver {
	// TODO: dataloader, caching, etc.
	// this is not going to scale well like this due to how TemperatureMeasurement.Batch() works.
	var resolved *batchResolver
	db, ok := ctx.Value("db").(*sqlx.DB)
	if !ok {
		log.Printf("No database in context")
		return nil
	}
	b, err := r.m.Batch(db)
	if err != nil {
		log.Printf("%v", err)
		return nil
	}
	resolved = &batchResolver{b: b}
	return resolved
}

func (r *temperatureMeasurementResolver) Sensor(ctx context.Context) *sensorResolver {
	var resolved *sensorResolver
	if r.m.Sensor != nil {
		resolved = &sensorResolver{s: r.m.Sensor}
	} else if r.m.SensorId != nil {
		db, ok := ctx.Value("db").(*sqlx.DB)
		if !ok {
			log.Printf("No database in context")
			return nil
		}
		sensor, err := worrywort.FindSensor(map[string]interface{}{"id": *r.m.SensorId}, db)
		if err != nil {
			log.Printf("%v", err)
			return nil
		}
		resolved = &sensorResolver{s: sensor}
	}

	return resolved
}

func (r *temperatureMeasurementResolver) CreatedBy(ctx context.Context) *userResolver {
	resolved := new(userResolver)
	if r.m.CreatedBy != nil {
		resolved = &userResolver{u: r.m.CreatedBy}
	} else if r.m.UserId != nil {
		db, ok := ctx.Value("db").(*sqlx.DB)
		if !ok {
			log.Printf("No database in context")
			return nil
		}
		user, err := worrywort.FindUser(map[string]interface{}{"id": *r.m.UserId}, db)
		if err != nil {
			log.Printf("%v", err)
			return nil
		}
		resolved = &userResolver{u: user}
	}
	return resolved
}

type temperatureMeasurementEdge struct {
	Cursor string
	Node   *temperatureMeasurementResolver
}

func (r *temperatureMeasurementEdge) CURSOR() string                        { return r.Cursor }
func (r *temperatureMeasurementEdge) NODE() *temperatureMeasurementResolver { return r.Node }

type temperatureMeasurementConnection struct {
	// if dataloader is implemented, this could just store the ids (and do a lighter query for those ids) and use dataloader
	// to get each individual edge or sensor and build the edge in the resolver function
	Edges    *[]*temperatureMeasurementEdge
	PageInfo *pageInfo
}

func (r *temperatureMeasurementConnection) PAGEINFO() pageInfo                    { return *r.PageInfo }
func (r *temperatureMeasurementConnection) EDGES() *[]*temperatureMeasurementEdge { return r.Edges }
