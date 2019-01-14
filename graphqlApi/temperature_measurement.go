package graphqlApi

import (
	"context"
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

func (r *temperatureMeasurementResolver) ID() graphql.ID                       { return graphql.ID(r.m.Id) }
func (r *temperatureMeasurementResolver) CreatedAt() string                    { return dateString(r.m.CreatedAt) }
func (r *temperatureMeasurementResolver) UpdatedAt() string                    { return dateString(r.m.UpdatedAt) }
func (r *temperatureMeasurementResolver) RecordedAt() string                   { return dateString(r.m.RecordedAt) }
func (r *temperatureMeasurementResolver) Temperature() float64                 { return r.m.Temperature }
func (r *temperatureMeasurementResolver) Units() worrywort.TemperatureUnitType { return r.m.Units }
func (r *temperatureMeasurementResolver) Batch(ctx context.Context) *batchResolver {
	// TODO: dataloader, caching, etc.
	var resolved *batchResolver
	if r.m.Batch != nil {
		resolved = &batchResolver{b: r.m.Batch}
	} else if r.m.BatchId.Valid {
		db, ok := ctx.Value("db").(*sqlx.DB)
		if !ok {
			log.Printf("No database in context")
			return nil
		}
		batch, err := worrywort.FindBatch(map[string]interface{}{"id": r.m.BatchId}, db)
		if err != nil {
			log.Printf("%v", err)
			return nil
		}
		resolved = &batchResolver{b: batch}
	}
	return resolved
}

func (r *temperatureMeasurementResolver) Sensor(ctx context.Context) *sensorResolver {
	// TODO: lookup sensor if not already populated
	var resolved *sensorResolver
	if r.m.Sensor != nil {
		resolved = &sensorResolver{t: r.m.Sensor}
	} else if r.m.SensorId.Valid {
		db, ok := ctx.Value("db").(*sqlx.DB)
		if !ok {
			log.Printf("No database in context")
			return nil
		}
		sensor, err := worrywort.FindSensor(map[string]interface{}{"id": r.m.SensorId}, db)
		if err != nil {
			log.Printf("%v", err)
			return nil
		}
		resolved = &sensorResolver{t: sensor}
	}

	return resolved
}

func (r *temperatureMeasurementResolver) Fermentor(ctx context.Context) *fermentorResolver {
	var resolved *fermentorResolver
	if r.m.Fermentor != nil {
		resolved = &fermentorResolver{f: r.m.Fermentor}
	} else if r.m.FermentorId.Valid {
		db, ok := ctx.Value("db").(*sqlx.DB)
		if !ok {
			log.Printf("No database in context")
			return nil
		}
		fermentor, err := worrywort.FindFermentor(map[string]interface{}{"id": r.m.FermentorId}, db)
		if err != nil {
			log.Printf("%v", err)
			return nil
		}
		resolved = &fermentorResolver{f: fermentor}
	}

	return resolved
}

// TODO: Make this return an actual nil if there is no createdBy, such as for a deleted user?
func (r *temperatureMeasurementResolver) CreatedBy(ctx context.Context) *userResolver {
	// TODO: lookup user if not already populated
	var resolved *userResolver
	if r.m.CreatedBy != nil {
		// TODO: this will probably go to taking a pointer to the User
		resolved = &userResolver{u: r.m.CreatedBy}
	} else if r.m.UserId.Valid {
		// Looking at https://github.com/OscarYuen/go-graphql-starter/blob/f8ff416af2213ef93ef5f459904d6a403ab25843/service/user_service.go#L23
		// and https://github.com/OscarYuen/go-graphql-starter/blob/f8ff416af2213ef93ef5f459904d6a403ab25843/server.go#L20
		// I will just want to put the db in my context even though it seems like many things say do not do that.
		// Not sure I like this at all, but I also do not want to have to attach the db from resolver to every other
		// resolver/type struct I create.
		db, ok := ctx.Value("db").(*sqlx.DB)
		if !ok {
			log.Printf("No database in context")
			return nil
		}
		user, err := worrywort.LookupUser(int(r.m.UserId.Int64), db)
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
