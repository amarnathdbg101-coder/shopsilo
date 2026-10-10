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
	phoneVerificationRepo := repository.NewPhoneVerificationRepo(db, logger)
	userService := services.NewUserService(userRepo)
	userService.SetPhoneVerificationRepo(phoneVerificationRepo)
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

	// Wire repos to userService for consolidated bootstrap initial state
	userService.SetShopRepo(shopRepo)
	userService.SetCategoryRepo(categoryRepo)

	// Product layer
	productRepo := repository.NewProductRepo(db, logger)
	productService := services.NewProductService(productRepo, shopRepo, categoryRepo)
	pc := controller.NewProductController(productService, shopService)

	// Upload layer (Cloudflare R2 image management)
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
	khataService := services.NewKhataService(khataRepo, shopRepo, userRepo)
	userService.SetKhataRepo(khataRepo)
	khatac := controller.NewKhataController(khataService)
	custKhatac := controller.NewCustomerKhataController(khataService)

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

	// Real-Time WebSocket Layer
	wsc := controller.NewWebSocketController(shopService, logger)

	// Shop Staff & Cashier sub-account layer
	staffRepo := repository.NewStaffRepo(db, logger)
	staffService := services.NewStaffService(staffRepo, shopRepo)
	staffc := controller.NewStaffController(staffService)

	// Wire aggregated repos to shopService for unified batch endpoints
	shopService.SetAggregatedRepos(categoryRepo, productRepo, posRepo, loyaltyRepo)

	r := chi.NewRouter()

	// Global middlewares
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.BotGuard)
	r.Use(middleware.BodySizeGuard(2 << 20))
	r.Use(chimw.Compress(5))
	r.Use(middleware.GlobalRateLimiter.Middleware())
	r.Use(middleware.BanGuard(modRepo))
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS)

	// Liveness & Readiness health check for production orchestrators
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { // Both
		if err := db.Ping(r.Context()); err != nil {
			reuse.Error(w, http.StatusServiceUnavailable, "database ping failed")
			return
		}
		reuse.Success(w, "ShopMe API is healthy and operational", map[string]string{
			"status": "healthy",
		})
	})

	// Public Grievance & Content Safety Report endpoint (IT Rules 2021 compliance)
	r.Post("/reports", modc.SubmitReport) // Both

	// Public Auth routes (Rate limited to 15 attempts/min per IP to prevent brute force)
	r.Route("/auth", func(r chi.Router) {
		r.Use(middleware.AuthRateLimiter.Middleware())
		r.With(middleware.OTPRateLimiter.Middleware()).Post("/send-otp", uc.SendRegistrationOTP)         // Both
		r.Post("/verify-otp", uc.VerifyRegistrationOTP)     // Both
		r.Post("/register", uc.Register)                   // Both
		r.Post("/login", uc.Login)                         // Both
		r.Post("/staff-login", staffc.StaffLogin)           // Shop
		r.Post("/google", uc.GoogleLogin)                   // Both
		r.Post("/forgot-password", uc.ForgotPassword)       // Both
		r.Post("/forget-password", uc.ForgotPassword)       // Both
		r.Post("/reset-password", uc.ResetPassword)         // Both
		r.Post("/refresh", uc.RefreshToken)                 // Both
	})

	// Public Browsing routes (customers & visitors)
	r.Get("/catalog/home-feed", sc.GetHomeFeed)                        // Customer
	r.Get("/home-feed", sc.GetHomeFeed)                                // Customer
	r.Post("/ai/customer-chat", aic.CustomerChat)                      // Customer
	r.Post("/ai/scan-product", aic.ScanProduct)                        // Both
	r.Post("/ai/parse-parchi", aic.ParseParchi)                        // Both
	r.Post("/ai/semantic-search", aic.SemanticSearch)                  // Customer
	r.Post("/ai/voice-bill", aic.VoiceBill)                            // Shop
	r.Post("/ai/marketing-campaign", aic.GenerateMarketingCampaign)   // Shop
	r.Post("/ai/bargain-assist", aic.BargainAssist)                    // Both
	r.Get("/categories", catc.List)                                    // Both
	r.Get("/shops", sc.List)                                           // Customer
	r.Get("/shops/{id}", sc.GetByID)                                   // Customer
	r.Get("/shops/slug/{slug}", sc.GetBySlug)                          // Customer
	r.Get("/shops/{slug}/qr", sc.GetShopQR)                            // Customer
	r.Get("/shops/{slug}/products", pc.ListByShop)                     // Customer
	r.Get("/shops/{slug}/reviews", revc.List)                          // Customer
	r.Get("/shops/{slug}/offers", loyc.ListOffers)                     // Customer
	r.Get("/offers", loyc.ListAllOffers)                               // Customer
	r.Get("/deals", loyc.ListAllOffers)                                // Customer
	r.Get("/products/nearby", pc.FindNearby)                           // Customer
	r.Get("/products", pc.List)                                        // Customer
	r.Get("/products/{id}", pc.GetByID)                                // Customer
	r.Get("/products/slug/{slug}", pc.GetBySlug)                       // Customer
	r.Get("/products/scan/{code}", loyc.ScanProduct)                   // Customer
	r.Post("/products/{id}/notify-me", invc.SubscribeStockAlert)       // Customer
	r.Post("/products/{id}/make-offer", pc.MakeOffer)                  // Customer
	r.Get("/receipts/{bill_number}", posc.ViewPublicReceiptPDF)        // Customer
	r.Post("/reports/telemetry-error", telc.LogFrontendError)          // Both
	r.Get("/images/*", upc.ServeImage)                                 // Both

	// Protected routes (JWT authentication required)
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAuth(utils.MustLoad().Jwt))

		// User profile actions
		r.Get("/user/me", uc.GetProfile)                                                             // Both
		r.Get("/user/bootstrap", uc.GetBootstrapData)                                               // Both
		r.Get("/user/loyalty", loyc.GetUserLoyalty)                                                 // Customer
		r.Put("/user/profile", uc.UpdateProfile)                                                     // Both
		r.With(middleware.UploadRateLimiter.Middleware()).Post("/user/avatar", upc.UploadUserAvatar) // Both

		// Customer Khata & Dual-Entry Udhar Passbook
		r.Get("/customer/khata", custKhatac.GetCustomerKhataSummary)                     // Customer
		r.Get("/customer/khata/{khataId}/transactions", custKhatac.GetCustomerKhataPassbook) // Customer
		r.Post("/customer/khata/{khataId}/dispute", custKhatac.DisputeTransaction)       // Customer
		r.Post("/customer/khata/{khataId}/pay-upi", custKhatac.SubmitUPIPayment)          // Customer
		r.Get("/customer/khata/{khataId}/statement.pdf", custKhatac.DownloadCustomerPDF) // Customer

		// Shop Owner management
		r.Post("/shops", sc.Create)                                                             // Shop
		r.Get("/shops/me", sc.GetMyShop)                                                         // Shop
		r.Get("/shops/me/dashboard", sc.GetMerchantDashboard)                                  // Shop
		r.Get("/shops/me/qr", sc.GetMyShopQR)                                                   // Shop
		r.Get("/shops/me/digest", sc.GetDailyDigest)                                            // Shop
		r.Put("/shops/me", sc.UpdateMyShop)                                                     // Shop
		r.Patch("/shops/me/status", sc.ToggleStatus)                                            // Shop
		r.Delete("/shops/me", sc.DeleteMyShop)                                                  // Shop
		r.Post("/shops/me/restore", sc.RestoreMyShop)                                           // Shop
		r.Get("/shops/me/ws", wsc.ServeShopWebSocket)                                            // Shop
		r.With(middleware.UploadRateLimiter.Middleware()).Post("/shops/me/images", upc.UploadShopImages) // Shop

		// Authenticated merchant intelligence
		r.Post("/ai/merchant-copilot", aic.MerchantCopilot) // Shop

		// Shop Staff & Cashier sub-accounts
		r.Get("/shops/me/staff", staffc.ListStaff)             // Shop
		r.Post("/shops/me/staff", staffc.CreateStaff)           // Shop
		r.Put("/shops/me/staff/{id}", staffc.UpdateStaff)       // Shop
		r.Delete("/shops/me/staff/{id}", staffc.DeleteStaff)    // Shop

		// Shop Product management
		r.Get("/shops/me/products", pc.ListMyShopProducts)                                                 // Shop
		r.With(middleware.MutationRateLimiter.Middleware()).Post("/products", pc.Create)                  // Shop
		r.Put("/products/{id}", pc.Update)                                                                 // Shop
		r.Delete("/products/{id}", pc.Delete)                                                              // Shop
		r.With(middleware.UploadRateLimiter.Middleware()).Post("/products/images", upc.UploadProductImages) // Shop
		r.Post("/shops/me/products/{id}/markdown", pc.ApplyClearanceMarkdown)                              // Shop
		r.Post("/shops/me/products/bulk-import", pc.BulkImport)                                            // Shop
		r.Get("/shops/me/products/import-template.csv", pc.DownloadImportTemplate)                         // Shop

		// Shop Inventory & Wholesale Restock
		r.Post("/shops/me/inventory/adjust", invc.AdjustStock)                           // Shop
		r.Get("/shops/me/inventory/low-stock", invc.GetLowStockAlerts)                     // Shop
		r.Post("/shops/me/inventory/alerts/{id}/dismiss", invc.DismissStockAlert)          // Shop
		r.Get("/shops/me/inventory/demand-watchlist", invc.GetDemandWatchlist)             // Shop
		r.Get("/shops/me/inventory/reorder-sheet.pdf", invc.DownloadReorderSheetPDF)       // Shop
		r.Post("/shops/me/inventory/procurement-pdf", invc.GenerateCustomProcurementPDF)   // Shop
		r.Get("/shops/me/inventory/reorder/whatsapp", invc.GetSupplierReorderWhatsApp)     // Shop

		// Shop Analytics & Profit Intelligence
		r.Get("/shops/me/analytics/profit", ac.GetMonthlyProfit)     // Shop
		r.Get("/shops/me/analytics/products", ac.GetProductMatrix)   // Shop

		// Shop Counter POS & Digital Receipts
		r.Post("/shops/me/pos/sale", posc.CreateSale)                               // Shop
		r.Post("/shops/me/pos/sales/{billNumber}/cancel", posc.CancelBill)          // Shop
		r.Get("/shops/me/pos/daily-summary", posc.GetDailySummary)                  // Shop
		r.Get("/shops/me/pos/summary", posc.GetDailySummary)                        // Shop
		r.Get("/shops/me/pos/scan/{sku}", posc.ScanBarcode)                         // Shop
		r.Get("/shops/me/pos/receipts/{bill_number}", posc.DownloadReceiptPDF)      // Shop
		r.Get("/shops/me/pos/receipts/{bill_number}/share", posc.ShareBill)         // Shop
		r.Get("/shops/me/pos/day-close", posc.GetDailyCloseReport)                  // Shop
		r.Get("/shops/me/pos/gst-report", posc.GetMonthlyGSTReport)                // Shop
		r.Post("/shops/me/pos/park", posc.ParkBill)                                 // Shop
		r.Get("/shops/me/pos/park", posc.ListParkedBills)                           // Shop
		r.Get("/shops/me/pos/park/{id}", posc.GetParkedBill)                        // Shop
		r.Delete("/shops/me/pos/park/{id}", posc.DeleteParkedBill)                  // Shop
		r.Get("/shops/me/pos/bargain-assist", pc.GetPOSBargainAssist)               // Shop
		r.Post("/shops/me/pos/parse-parchi", aic.ParseParchi)                      // Shop
		r.Get("/shops/me/pos/weekly-scorecard", posc.GetWeeklyScorecard)           // Shop
		r.Get("/shops/me/pos/customers/{phone}/recent-basket", posc.GetCustomerRecentBasket) // Shop
		r.Post("/shops/me/pos/returns", posc.ProcessPOSReturn)                     // Shop

		// Shop Expenses (Dukan ke Roz ke Kharche)
		r.Post("/shops/me/expenses", expc.CreateExpense)      // Shop
		r.Get("/shops/me/expenses", expc.ListExpenses)        // Shop
		r.Delete("/shops/me/expenses/{id}", expc.DeleteExpense) // Shop

		// Shop Customer Khata (Udhar & settlement passbook)
		r.Get("/shops/me/khata/summary", khatac.GetSummary)                      // Shop
		r.Get("/shops/me/khata/aging", khatac.GetAgingReport)                    // Shop
		r.Get("/shops/me/khata", khatac.ListCustomers)                           // Shop
		r.Get("/shops/me/khata/{mobile}", khatac.GetCustomerHistory)             // Shop
		r.Get("/shops/me/khata/{mobile}/statement", khatac.GetCustomerHistory)   // Shop
		r.Get("/shops/me/khata/{mobile}/reminder", khatac.GetPaymentReminder)     // Shop
		r.Get("/shops/me/khata/{mobile}/statement.pdf", khatac.DownloadStatementPDF) // Shop
		r.Get("/shops/me/khata/{mobile}/statement/share", khatac.GetStatementShare) // Shop
		r.Put("/shops/me/khata/{mobile}/credit-limit", khatac.UpdateCreditLimit) // Shop
		r.Post("/shops/me/khata", khatac.RecordCredit)                           // Shop
		r.Post("/shops/me/khata/{mobile}/payment", khatac.RecordPayment)         // Shop
		r.Post("/shops/me/khata/{id}/request-closure", khatac.RequestClosure)   // Shop
		r.Post("/shops/me/khata/{id}/verify-closure-otp", khatac.VerifyClosureOTP) // Shop
		r.Post("/shops/me/khata/{id}/transactions/{txId}/reverse", khatac.ReverseTransaction) // Shop
		r.Post("/shops/me/khata/{id}/dispute/{txId}/resolve", khatac.ResolveDispute) // Shop
		r.Post("/shops/me/khata/{id}/promise-date", khatac.SetPromiseToPay)      // Shop
		r.Get("/shops/me/khata/{mobile}/trust-score", khatac.GetCustomerTrustScore) // Shop

		// Customer Digital Khata & Udhar Passbook (Dual-Entry Ledger)
		r.Get("/customer/khata", custKhatac.GetCustomerKhataSummary)                     // Customer
		r.Get("/customer/khata/{khataId}/transactions", custKhatac.GetCustomerKhataPassbook) // Customer
		r.Post("/customer/khata/{khataId}/dispute", custKhatac.DisputeTransaction)       // Customer
		r.Post("/customer/khata/{khataId}/pay-upi", custKhatac.SubmitUPIPayment)          // Customer
		r.Get("/customer/khata/{khataId}/statement.pdf", custKhatac.DownloadCustomerPDF) // Customer
		r.Post("/customer/khata/{khataId}/request-closure", custKhatac.RequestClosure)   // Customer
		r.Post("/customer/khata/{khataId}/verify-closure-otp", custKhatac.VerifyClosureOTP) // Customer
		r.Put("/customer/khata/{khataId}/otp-protection", custKhatac.SetCreditOTPProtection) // Customer
		r.Post("/customer/khata/{khataId}/promise-date", custKhatac.SetPromiseToPay)      // Customer

		// Shop Returns & VIP Offers
		r.Post("/shops/me/returns", loyc.ProcessReturn)          // Shop
		r.Get("/shops/me/returns", loyc.ListReturns)            // Shop
		r.Post("/shops/me/offers", loyc.CreateOffer)            // Shop
		r.Put("/shops/me/offers/{id}", loyc.UpdateOffer)        // Shop
		r.Delete("/shops/me/offers/{id}", loyc.DeleteOffer)     // Shop
		r.Get("/shops/me/offers/history", loyc.GetOfferHistory)  // Shop

		// In-Store Item Reservations (Customer)
		r.Post("/reservations", resc.Create)                   // Customer
		r.Get("/reservations", resc.ListUserReservations)      // Customer
		r.Get("/reservations/{id}", resc.GetByID)              // Customer
		r.Post("/reservations/{id}/cancel", resc.CancelUserReservation) // Customer

		// In-Store Reservations (Shopkeeper counter management)
		r.Get("/shops/me/reservations", resc.ListShopReservations)         // Shop
		r.Post("/shops/me/reservations/verify", resc.VerifyShopReservation) // Shop
		r.Post("/shops/me/reservations/{id}/cancel", resc.CancelShopReservation) // Shop

		// Shop Reviews (Customer)
		r.Post("/shops/{slug}/reviews", revc.AddOrUpdate) // Customer
		r.Delete("/shops/{slug}/reviews", revc.Delete)     // Customer
	})

	// Admin protected routes (Role: admin required)
	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.JWTAuth(utils.MustLoad().Jwt))
		r.Use(middleware.RequireRole("admin"))

		r.Get("/stats", modc.GetStats)                           // Admin
		r.Get("/shops", modc.ListAdminShops)                     // Admin
		r.Patch("/shops/{id}/status", modc.UpdateAdminShopStatus) // Admin
		r.Post("/shops/{id}/ban", modc.BanShop)                   // Admin

		r.Get("/reports", modc.ListReports)                      // Admin
		r.Post("/reports/{id}/resolve", modc.ResolveReport)      // Admin

		r.Get("/banned-entities", modc.ListBannedEntities)       // Admin
		r.Post("/banned-entities", modc.AddBannedEntity)         // Admin
		r.Delete("/banned-entities/{id}", modc.UnbanEntity)      // Admin

		// Category management
		r.Get("/categories", catc.AdminList)                     // Admin
		r.Post("/categories", catc.AdminCreate)                   // Admin
		r.Put("/categories/{id}", catc.AdminUpdate)               // Admin
		r.Delete("/categories/{id}", catc.AdminDelete)            // Admin

		// User & Merchant management
		r.Get("/users", uc.AdminListUsers)                       // Admin
		r.Patch("/users/{id}/status", uc.AdminUpdateUserStatus)   // Admin

		// 24-Hour Developer Error Telemetry Vault & Live Performance Metrics
		r.Get("/errors", telc.GetAdminErrors)                    // Admin
		r.Delete("/errors/clear", telc.ClearAdminErrors)          // Admin
		r.Get("/performance-metrics", telc.GetLivePerformanceMetrics) // Admin
	})

	return r
}

