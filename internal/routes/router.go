package routes

import (
	handler "shopMe/internal/handler/user"
	"shopMe/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RouteSetup(db *pgxpool.Pool) chi.Router {
	r := chi.NewRouter()
	uh := handler.NewUserHandler(db)
	
	r.Post("/register", uh.Register)
	r.Get("/login", uh.Login)

	// Protected routes
	r.Group(func(pr chi.Router) {
		pr.Use(middleware.AuthMiddleware)
		pr.Get("/profile", uh.Profile)
		pr.Delete("/delete-account", uh.DeleteAccount)
	})
	return r
}
