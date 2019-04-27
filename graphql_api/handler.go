package graphql_api

import (
	"context"
	"github.com/graph-gophers/graphql-go/relay"
	"github.com/jmoiron/sqlx"
	"net/http"
)

type Handler struct {
	Db *sqlx.DB
	*relay.Handler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), "db", h.Db)
	h.Handler.ServeHTTP(w, r.WithContext(ctx))
}
