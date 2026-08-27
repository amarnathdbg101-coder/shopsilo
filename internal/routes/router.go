package routes

import (
	"shopMe/internal/handler/user"
	"shopMe/internal/middleware"

	"firebase.google.com/go/v4/auth"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RouteSetup(db *pgxpool.Pool, firebaseAuth *auth.Client) chi.Router {
	r := chi.NewRouter()
	uh := user.NewUserHandler(db)
	
	// Protected routes (Requires Firebase ID Token)
	r.Group(func(pr chi.Router) {
		pr.Use(middleware.AuthMiddleware(firebaseAuth))

		// User sync and profile
		pr.Post("/sync-user", uh.SyncUser)
		pr.Get("/profile", uh.Profile)
		pr.Delete("/delete-account", uh.DeleteAccount)
	})

	return r
}
