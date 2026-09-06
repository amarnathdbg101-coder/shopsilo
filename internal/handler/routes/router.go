package routes

import (
	"shopMe/internal/handler/controller"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func RouteSetup(db *pgxpool.Pool, logger *zap.Logger) chi.Router {
	// User layer
	userRepo := repository.NewUserRepo(db, logger)
	userService := services.NewUserService(userRepo)
	uc := controller.NewUserController(userService)

	r := chi.NewRouter()

	// Public routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", uc.Register)
		r.Post("/login", uc.Login)
	})

	return r
}
