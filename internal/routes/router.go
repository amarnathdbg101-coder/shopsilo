package routes

import (
	

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RouteSetup(db *pgxpool.Pool) chi.Router {
	r := chi.NewRouter()
	return r
}
