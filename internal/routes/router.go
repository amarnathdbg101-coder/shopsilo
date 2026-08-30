package routes

import (
	"shopMe/internal/handler"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RouteSetup(db *pgxpool.Pool) chi.Router {
	r := chi.NewRouter()
	r.Post("/product",handler.Create)
	return r
}
