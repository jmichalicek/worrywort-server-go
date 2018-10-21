package graphqlApi

import (
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
)

// Resolve a worrywort.TemperatureMeasurement
type temperatureMeasurementResolver struct {
	// m for measurement
	m worrywort.TemperatureMeasurement
}

func (r *temperatureMeasurementResolver) ID() graphql.ID                       { return graphql.ID(r.m.Id) }
func (r *temperatureMeasurementResolver) CreatedAt() string                    { return dateString(r.m.CreatedAt) }
func (r *temperatureMeasurementResolver) UpdatedAt() string                    { return dateString(r.m.UpdatedAt) }
func (r *temperatureMeasurementResolver) RecordedAt() string                   { return dateString(r.m.RecordedAt) }
func (r *temperatureMeasurementResolver) Temperature() float64                 { return r.m.Temperature }
func (r *temperatureMeasurementResolver) Units() worrywort.TemperatureUnitType { return r.m.Units }
func (r *temperatureMeasurementResolver) Batch() *batchResolver {
	if r.m.Batch != nil {
		return &batchResolver{b: r.m.Batch}
	}
	return nil
}

func (r *temperatureMeasurementResolver) TemperatureSensor() *temperatureSensorResolver {
	return &temperatureSensorResolver{t: *(r.m.TemperatureSensor)}
}

func (r *temperatureMeasurementResolver) Fermenter() *fermenterResolver {
	return &fermenterResolver{f: *(r.m.Fermenter)}
}

// TODO: Make this return an actual nil if there is no createdBy, such as for a deleted user?
func (r *temperatureMeasurementResolver) CreatedBy() *userResolver {
	return &userResolver{u: *r.m.CreatedBy}
}
