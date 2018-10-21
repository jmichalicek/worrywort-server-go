package graphqlApi

import (
	"context"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"strconv"
)

// Resolve a worrywort.TemperatureSensor
type temperatureSensorResolver struct {
	t *worrywort.TemperatureSensor
}

func (r temperatureSensorResolver) ID() graphql.ID    { return graphql.ID(strconv.Itoa(r.t.Id)) }
func (r temperatureSensorResolver) CreatedAt() string { return dateString(r.t.CreatedAt) }
func (r temperatureSensorResolver) UpdatedAt() string { return dateString(r.t.UpdatedAt) }

// TODO: Make this return an actual nil if there is no createdBy, such as for a deleted user?
func (r temperatureSensorResolver) CreatedBy(ctx context.Context) *userResolver {
	// TODO: lookup user if not already populated
	// TODO: Really implement this!!!!
	if r.t.CreatedBy == nil {
		return nil
	}
	return &userResolver{u: r.t.CreatedBy}
}
func (r temperatureSensorResolver) Name() string { return r.t.Name }
