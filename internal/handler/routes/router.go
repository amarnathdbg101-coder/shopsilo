// Package routes handle routing work .
package routes

import (
	"net/http"
	"shopMe/internal/handler/controller"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
	"shopMe/internal/middleware"
	"shopMe/internal/reuse"
	"shopMe/internal/utils"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func RouteSetup(db *pgxpool.Pool, logger *zap.Logger) chi.Router {
	// User layer
	userRepo := repository.NewUserRepo(db, logger)
	userService := services.NewUserService(userRepo)
	uc := controller.NewUserController(userService)

	// Moderation & Anti-Abuse layer
	modRepo := repository.NewModerationRepo(db, logger)
	modService := services.NewModerationService(modRepo, nil, userRepo) // shopRepo injected below
	modc := controller.NewModerationController(modService)

	// Shop layer
	shopRepo := repository.NewShopRepo(db, logger)
	shopService := services.NewShopService(shopRepo, userRepo, modRepo)
	sc := controller.NewShopController(shopService)

	// Wire shopRepo to modService
	modService = services.NewModerationService(modRepo, shopRepo, userRepo)
	modc = controller.NewModerationController(modService)

	// Category layer
	categoryRepo := repository.NewCategoryRepo(db, logger)
	categoryService := services.NewCategoryService(categoryRepo)
	catc := controller.NewCategoryController(categoryService)

	// Product layer
	productRepo := repository.NewProductRepo(db, logger)
	productService := services.NewProductService(productRepo, shopRepo, categoryRepo)
	pc := controller.NewProductController(productService, shopService)

	// Upload layer (Cloudflare R2 cost-efficient image management with perceptual safety check)
	uploadService := services.NewUploadService(userRepo, shopRepo, modRepo)
	upc := controller.NewUploadController(uploadService)

	// Reservation layer (In-Store item hold & counter pickup verification)
	resRepo := repository.NewReservationRepo(db, logger)
	resService := services.NewReservationService(resRepo, shopRepo, productRepo)
	resc := controller.NewReservationController(resService)

	// Review layer (Shop reviews & trust ratings)
	reviewRepo := repository.NewReviewRepo(db, logger)
	reviewService := services.NewReviewService(reviewRepo, shopRepo)
	revc := controller.NewReviewController(reviewService)

	// Inventory layer (Stock management, low stock alerts, wholesale reorder PDF)
	inventoryService := services.NewInventoryService(productRepo, shopRepo)
	invc := controller.NewInventoryController(inventoryService)

	// Expense layer (Daily store operational expenses)
	expenseRepo := repository.NewExpenseRepo(db, logger)
	expenseService := services.NewExpenseService(expenseRepo, shopRepo)
	expc := controller.NewExpenseController(expenseService)

	// Khata layer (Customer credit & udhar book)
	khataRepo := repository.NewKhataRepo(db, logger)
	khataService := services.NewKhataService(khataRepo, shopRepo)
	khatac := controller.NewKhataController(khataService)

	// Analytics layer (Monthly profit, Best/Worst/Old/New product matrix, Net Pocket Profit)
	analyticsService := services.NewAnalyticsService(productRepo, shopRepo, expenseRepo)
	ac := controller.NewAnalyticsController(analyticsService)

	// Loyalty & Returns layer (Customer loyalty points, in-store returns, VIP offers, barcode scan)
	loyaltyRepo := repository.NewLoyaltyRepo(db, logger)
	loyaltyService := services.NewLoyaltyService(loyaltyRepo, shopRepo, productRepo)
	loyc := controller.NewLoyaltyController(loyaltyService)

	// Telemetry & Error Audit layer
	telemetryRepo := repository.NewTelemetryRepo(db, logger)
	telemetryService := services.NewTelemetryService(telemetryRepo)
	telc := controller.NewTelemetryController(telemetryService)

	// AI Layer (Gemini Flash - Customer Shopping Sathi & Merchant Copilot)
	aiService := services.NewAIService(shopRepo)
	aic := controller.NewAIController(aiService)

	// POS layer (Counter billing POS, daily summary, digital receipts, credit integration)
	posRepo := repository.NewPOSRepo(db, logger)
	posService := services.NewPOSService(posRepo, shopRepo, productRepo, khataRepo)
	posc := controller.NewPOSController(posService)

	// Wire aggregated repos to shopService for unified batch endpoints
	shopService.SetAggregatedRepos(categoryRepo, productRepo, posRepo, loyaltyRepo)

	r := chi.NewRouter()

	// Global middlewares
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Compress(5))
	r.Use(middleware.GlobalRateLimiter.Middleware())
	r.Use(middleware.BanGuard(modRepo))
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS)

	// Liveness & Readiness health check for production orchestrators
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			reuse.Error(w, http.StatusServiceUnavailable, "database ping failed")
			return
		}
		reuse.Success(w, "ShopMe API is healthy and operational", map[string]string{
			"status": "healthy",
		})
	})

	// Public Grievance & Content Safety Report endpoint (IT Rules 2021 compliance)
	r.Post("/reports", modc.SubmitReport)

	// Public Auth routes (Rate limited to 10 attempts/min per IP to prevent brute force)
	r.Route("/auth", func(r chi.Router) {
		r.Use(middleware.AuthRateLimiter.Middleware())
		r.Post("/register", uc.Register)
		r.Post("/login", uc.Login)
		r.Post("/google", uc.GoogleLogin)
		r.Post("/forgot-password", uc.ForgotPassword)
		r.Post("/forget-password", uc.ForgotPassword) // alias for convenience
		r.Post("/reset-password", uc.ResetPassword)
		r.Post("/refresh", uc.RefreshToken)
	})

	// Public Browsing routes (customers & visitors)
	r.Get("/catalog/home-feed", sc.GetHomeFeed) // Consolidated customer explore feed
	r.Post("/ai/customer-chat", aic.CustomerChat)
	r.Post("/ai/scan-product", aic.ScanProduct)
	r.Post("/ai/parse-parchi", aic.ParseParchi)
	r.Post("/ai/semantic-search", aic.SemanticSearch)
	r.Post("/ai/voice-bill", aic.VoiceBill)
	r.Post("/ai/marketing-campaign", aic.GenerateMarketingCampaign)
	r.Post("/ai/bargain-assist", aic.BargainAssist)
	r.Get("/categories", catc.List)
	r.Get("/shops", sc.List)
	r.Get("/shops/{id}", sc.GetByID)
	r.Get("/shops/slug/{slug}", sc.GetBySlug)
	r.Get("/shops/{slug}/qr", sc.GetShopQR)
	r.Get("/shops/{slug}/products", pc.ListByShop)
	r.Get("/shops/{slug}/reviews", revc.List)
	r.Get("/shops/{slug}/offers", loyc.ListOffers)
	r.Get("/offers", loyc.ListAllOffers)
	r.Get("/deals", loyc.ListAllOffers)
	r.Get("/products/nearby", pc.FindNearby)
	r.Get("/products", pc.List)
	r.Get("/products/{id}", pc.GetByID)
	r.Get("/products/slug/{slug}", pc.GetBySlug)
	r.Get("/products/scan/{code}", loyc.ScanProduct)
	r.Post("/products/{id}/notify-me", invc.SubscribeStockAlert)
	r.Post("/products/{id}/make-offer", pc.MakeOffer)
	r.Get("/receipts/{bill_number}", posc.ViewPublicReceiptPDF)
	r.Post("/reports/telemetry-error", telc.LogFrontendError)
	r.Get("/images/*", upc.ServeImage) // Public Cloudflare R2 image streaming proxy

	// Protected routes (JWT authentication required)
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth(utils.MustLoad().Jwt))

		// User profile actions
		r.Get("/user/me", uc.GetProfile)
		r.Put("/user/profile", uc.UpdateProfile)
		r.With(middleware.UploadRateLimiter.Middleware()).Post("/user/avatar", upc.UploadUserAvatar)

		// Shop Owner management
		r.Post("/shops", sc.Create)
		r.Get("/shops/me", sc.GetMyShop)
		r.Get("/shops/me/dashboard", sc.GetMerchantDashboard) // Consolidated merchant dashboard
		r.Get("/shops/me/qr", sc.GetMyShopQR)
		r.Get("/shops/me/digest", sc.GetDailyDigest)
		r.Put("/shops/me", sc.UpdateMyShop)
		r.Patch("/shops/me/status", sc.ToggleStatus)
		r.Delete("/shops/me", sc.DeleteMyShop)
		r.Post("/shops/me/restore", sc.RestoreMyShop)
		r.With(middleware.UploadRateLimiter.Middleware()).Post("/shops/me/images", upc.UploadShopImages)

		// Shop Product management
		r.Get("/shops/me/products", pc.ListMyShopProducts)
		r.Post("/products", pc.Create)
		r.Put("/products/{id}", pc.Update)
		r.Delete("/products/{id}", pc.Delete)
		r.With(middleware.UploadRateLimiter.Middleware()).Post("/products/images", upc.UploadProductImages)
		r.Post("/shops/me/products/{id}/markdown", pc.ApplyClearanceMarkdown)

		// Shop Inventory & Wholesale Restock
		r.Post("/shops/me/inventory/adjust", invc.AdjustStock)
		r.Get("/shops/me/inventory/low-stock", invc.GetLowStockAlerts)
		r.Post("/shops/me/inventory/alerts/{id}/dismiss", invc.DismissStockAlert)
		r.Get("/shops/me/inventory/demand-watchlist", invc.GetDemandWatchlist)
		r.Get("/shops/me/inventory/reorder-sheet.pdf", invc.DownloadReorderSheetPDF)
		r.Get("/shops/me/inventory/reorder/whatsapp", invc.GetSupplierReorderWhatsApp)

		// Shop Analytics & Profit Intelligence
		r.Get("/shops/me/analytics/profit", ac.GetMonthlyProfit)
		r.Get("/shops/me/analytics/products", ac.GetProductMatrix)

		// Shop Counter POS & Digital Receipts
		r.Post("/shops/me/pos/sale", posc.CreateSale)
		r.Post("/shops/me/pos/sales/{billNumber}/cancel", posc.CancelBill)
		r.Get("/shops/me/pos/daily-summary", posc.GetDailySummary)
		r.Get("/shops/me/pos/summary", posc.GetDailySummary) // frontend alias
		r.Get("/shops/me/pos/scan/{sku}", posc.ScanBarcode)
		r.Get("/shops/me/pos/receipts/{bill_number}", posc.DownloadReceiptPDF)
		r.Get("/shops/me/pos/receipts/{bill_number}/share", posc.ShareBill)
		r.Get("/shops/me/pos/day-close", posc.GetDailyCloseReport)
		r.Get("/shops/me/pos/gst-report", posc.GetMonthlyGSTReport)
		r.Post("/shops/me/pos/park", posc.ParkBill)
		r.Get("/shops/me/pos/park", posc.ListParkedBills)
		r.Get("/shops/me/pos/park/{id}", posc.GetParkedBill)
		r.Delete("/shops/me/pos/park/{id}", posc.DeleteParkedBill)
		r.Get("/shops/me/pos/bargain-assist", pc.GetPOSBargainAssist)
		r.Post("/shops/me/pos/parse-parchi", aic.ParseParchi)
		r.Get("/shops/me/pos/weekly-scorecard", posc.GetWeeklyScorecard)
		r.Get("/shops/me/pos/customers/{phone}/recent-basket", posc.GetCustomerRecentBasket)
		r.Post("/shops/me/pos/returns", posc.ProcessPOSReturn)

		// Shop Expenses (Dukan ke Roz ke Kharche)
		r.Post("/shops/me/expenses", expc.CreateExpense)
		r.Get("/shops/me/expenses", expc.ListExpenses)
		r.Delete("/shops/me/expenses/{id}", expc.DeleteExpense)

		// Shop Customer Khata (Udhar & settlement passbook)
		r.Get("/shops/me/khata/summary", khatac.GetSummary)
		r.Get("/shops/me/khata/aging", khatac.GetAgingReport)
		r.Get("/shops/me/khata", khatac.ListCustomers)
		r.Get("/shops/me/khata/{mobile}", khatac.GetCustomerHistory)
		r.Get("/shops/me/khata/{mobile}/statement", khatac.GetCustomerHistory) // frontend JSON alias
		r.Get("/shops/me/khata/{mobile}/reminder", khatac.GetPaymentReminder)
		r.Get("/shops/me/khata/{mobile}/statement.pdf", khatac.DownloadStatementPDF)
		r.Get("/shops/me/khata/{mobile}/statement/share", khatac.GetStatementShare)
		r.Put("/shops/me/khata/{mobile}/credit-limit", khatac.UpdateCreditLimit)
		r.Post("/shops/me/khata", khatac.RecordCredit)
		r.Post("/shops/me/khata/{mobile}/payment", khatac.RecordPayment)

		// Shop Returns & VIP Offers
		r.Post("/shops/me/returns", loyc.ProcessReturn)
		r.Get("/shops/me/returns", loyc.ListReturns)
		r.Post("/shops/me/offers", loyc.CreateOffer)
		r.Put("/shops/me/offers/{id}", loyc.UpdateOffer)
		r.Delete("/shops/me/offers/{id}", loyc.DeleteOffer)
		r.Get("/shops/me/offers/history", loyc.GetOfferHistory)

		// In-Store Item Reservations (Customer)
		r.Post("/reservations", resc.Create)
		r.Get("/reservations", resc.ListUserReservations)
		r.Get("/reservations/{id}", resc.GetByID)
		r.Post("/reservations/{id}/cancel", resc.CancelUserReservation)

		// In-Store Reservations (Shopkeeper counter management)
		r.Get("/shops/me/reservations", resc.ListShopReservations)
		r.Post("/shops/me/reservations/verify", resc.VerifyShopReservation)
		r.Post("/shops/me/reservations/{id}/cancel", resc.CancelShopReservation)

		// Shop Reviews (Customer)
		r.Post("/shops/{slug}/reviews", revc.AddOrUpdate)
		r.Delete("/shops/{slug}/reviews", revc.Delete)
	})

	// Admin protected routes (Role: admin required)
	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.JWTAuth(utils.MustLoad().Jwt))
		r.Use(middleware.RequireRole("admin"))

		r.Get("/stats", modc.GetStats)
		r.Get("/shops", modc.ListAdminShops)
		r.Patch("/shops/{id}/status", modc.UpdateAdminShopStatus)
		r.Post("/shops/{id}/ban", modc.BanShop)

		r.Get("/reports", modc.ListReports)
		r.Post("/reports/{id}/resolve", modc.ResolveReport)

		r.Get("/banned-entities", modc.ListBannedEntities)
		r.Post("/banned-entities", modc.AddBannedEntity)
		r.Delete("/banned-entities/{id}", modc.UnbanEntity)

		// Category management
		r.Get("/categories", catc.AdminList)
		r.Post("/categories", catc.AdminCreate)
		r.Put("/categories/{id}", catc.AdminUpdate)
		r.Delete("/categories/{id}", catc.AdminDelete)

		// User & Merchant management
		r.Get("/users", uc.AdminListUsers)
		r.Patch("/users/{id}/status", uc.AdminUpdateUserStatus)

		// 24-Hour Developer Error Telemetry Vault
		r.Get("/errors", telc.GetAdminErrors)
		r.Delete("/errors/clear", telc.ClearAdminErrors)
	})

	return r
}
