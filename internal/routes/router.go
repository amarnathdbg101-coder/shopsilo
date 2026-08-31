package routes

import (
	"shopMe/internal/handler/products"
	"shopMe/internal/handler/users"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func RouteSetup(db *pgxpool.Pool, logger *zap.Logger) chi.Router {
	ph := products.NewProductHandler(db, logger)
	uh := users.NewUserHandler(db, logger)
	r := chi.NewRouter()

	// Group for product
	r.Route("/products", func(r chi.Router) {
		r.Post("/", ph.AddProduct)
		r.Get("/:id", ph.GetProduct)
		r.Get("/", ph.GetProducts)
		r.Put("/:id", ph.UpdateProduct)
		r.Delete("/:id", ph.DeleteProduct)
	})

	// Group for users
	r.Route("/users", func(r chi.Router) {
		r.Post("/", uh.Register)
		r.Get("/", uh.Login)
		r.Delete("/:id", uh.DeleteAccount)
		r.Put("/:id", uh.ForgetPassword)
	})

	return r
}
