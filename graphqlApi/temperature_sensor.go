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

type temperatureSensorEdge struct {
	Cursor string
	Node   *temperatureSensorResolver
}

func (r *temperatureSensorEdge) CURSOR() string                   { return r.Cursor }
func (r *temperatureSensorEdge) NODE() *temperatureSensorResolver { return r.Node }

// Going full relay, I suppose
// the graphql lib needs case-insensitive match of names on the methods
// so the resolver functions are just named all caps... alternately the
// struct members could be named as such to avoid a collision
// idea from https://github.com/deltaskelta/graphql-go-pets-example/blob/ab169fb644b1a00998208e7feede5975214d60da/users.go#L156
type temperatureSensorConnection struct {
	// if dataloader is implemented, this could just store the ids (and do a lighter query for those ids) and use dataloader
	// to get each individual edge or sensor and build the edge in the resolver function
	Edges    *[]*temperatureSensorEdge
	PageInfo *pageInfo
}

func (r *temperatureSensorConnection) PAGEINFO() pageInfo               { return *r.PageInfo }
func (r *temperatureSensorConnection) EDGES() *[]*temperatureSensorEdge { return r.Edges }
