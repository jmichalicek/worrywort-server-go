package graphqlApi

import (
	"context"
	"github.com/davecgh/go-spew/spew"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jmichalicek/worrywort-server-go/worrywort"
	"github.com/jmoiron/sqlx"
	"log"
	"strconv"
)

// Resolve a worrywort.Sensor
type sensorResolver struct {
	s *worrywort.Sensor
}

func (r *sensorResolver) ID() graphql.ID {
	if r.s == nil && r.s.Id == nil {
		log.Printf("sensor resolver with nil id: %s", spew.Sdump(r))
		return graphql.ID("")
	}
	return graphql.ID(strconv.Itoa(int(*r.s.Id)))
}
func (r *sensorResolver) CreatedAt() string { return dateString(r.s.CreatedAt) }
func (r *sensorResolver) UpdatedAt() string { return dateString(r.s.UpdatedAt) }

// TODO: Make this return an actual nil if there is no createdBy, such as for a deleted user?
func (r *sensorResolver) CreatedBy(ctx context.Context) *userResolver {
	var resolved *userResolver
	sensor := r.s
	// Not sure these parens are necessary, but vs code complains without them
	// because it seems to think I am referring to this function
	if (sensor.CreatedBy) != nil {
		resolved = &userResolver{u: r.s.CreatedBy}
	} else if sensor.UserId != nil {
		db, ok := ctx.Value("db").(*sqlx.DB)
		if !ok {
			log.Printf("No database in context")
			return nil
		}
		user, err := worrywort.LookupUser(*r.s.UserId, db)
		if err != nil {
			log.Printf("%v", err)
			return nil
		}
		resolved = &userResolver{u: user}
	}
	return resolved
}

func (r *sensorResolver) Name() string { return r.s.Name }

type sensorEdge struct {
	Cursor string
	Node   *sensorResolver
}

func (r *sensorEdge) CURSOR() string        { return r.Cursor }
func (r *sensorEdge) NODE() *sensorResolver { return r.Node }

// Going full relay, I suppose
// the graphql lib needs case-insensitive match of names on the methods
// so the resolver functions are just named all caps... alternately the
// struct members could be named as such to avoid a collision
// idea from https://github.com/deltaskelta/graphql-go-pets-example/blob/ab169fb644b1a00998208e7feede5975214d60da/users.go#L156
type sensorConnection struct {
	// if dataloader is implemented, this could just store the ids (and do a lighter query for those ids) and use dataloader
	// to get each individual edge or sensor and build the edge in the resolver function
	Edges    *[]*sensorEdge
	PageInfo *pageInfo
}

func (r *sensorConnection) PAGEINFO() pageInfo    { return *r.PageInfo }
func (r *sensorConnection) EDGES() *[]*sensorEdge { return r.Edges }
