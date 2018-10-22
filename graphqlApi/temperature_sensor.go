package graphqlApi

import (
	"context"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"log"
	"strconv"
)

// Resolve a worrywort.TemperatureSensor
type temperatureSensorResolver struct {
	t *worrywort.TemperatureSensor
}

func (r *temperatureSensorResolver) ID() graphql.ID    { return graphql.ID(strconv.Itoa(r.t.Id)) }
func (r *temperatureSensorResolver) CreatedAt() string { return dateString(r.t.CreatedAt) }
func (r *temperatureSensorResolver) UpdatedAt() string { return dateString(r.t.UpdatedAt) }

// TODO: Make this return an actual nil if there is no createdBy, such as for a deleted user?
func (r *temperatureSensorResolver) CreatedBy(ctx context.Context) *userResolver {
	var resolved *userResolver
	sensor := r.t
	// Not sure these parens are necessary, but vs code complains without them
	// because it seems to think I am referring to this function
	if (sensor.CreatedBy) != nil {
		resolved = &userResolver{u: r.t.CreatedBy}
	} else if sensor.UserId.Valid {
		db, ok := ctx.Value("db").(*sqlx.DB)
		if !ok {
			log.Printf("No database in context")
			return nil
		}
		user, err := worrywort.LookupUser(int(r.t.UserId.Int64), db)
		if err != nil {
			log.Printf("%v", err)
			return nil
		}
		resolved = &userResolver{u: user}
	}
	return resolved
}

func (r *temperatureSensorResolver) Name() string { return r.t.Name }
