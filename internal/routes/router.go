package routes

import (
	"shopMe/internal/handler/products"
	"shopMe/internal/handler/shop"
	"shopMe/internal/handler/users"
	"shopMe/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func RouteSetup(db *pgxpool.Pool, logger *zap.Logger) chi.Router {
	ph := products.NewProductHandler(db, logger)
	uh := users.NewUserHandler(db, logger)
	sh := shop.NewShopHandler(db, logger)

	r := chi.NewRouter()

	// Public routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", uh.Register)
		r.Post("/login", uh.Login) // login should be POST
	})

	// Protected routes (JWT required)
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuthMiddleware)

		// Profile
		r.Route("/profile", func(r chi.Router) {
			r.Get("/", uh.GetProfile)
			r.Put("/", uh.UpdateProfile)
			r.Put("/password", uh.ChangePassword)
			r.Post("/images",uh.UploadProfileImage)
		})

		// Shop
		r.Route("/shops", func(r chi.Router) {
			r.Post("/", sh.CreateShop)
			r.Delete("/", sh.DeleteShop)
			r.Put("/", sh.UpdateShop)
			r.Post("/images", sh.UploadImage)

		})

		// Product
		r.Route("/products", func(r chi.Router) {
			r.Post("/", ph.AddProduct)
			r.Get("/{id}", ph.GetProduct)
			r.Get("/", ph.GetProducts)
			r.Put("/{id}", ph.UpdateProduct)
			r.Delete("/{id}", ph.DeleteProduct)
			r.Post("/images", ph.UploadImage)
		})
	})

	return r
}
