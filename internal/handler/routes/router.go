// Package routes handle routing work .
package routes

import (
	"context"
	"time"

	"shopMe/internal/app"
	"shopMe/internal/middleware"
	"shopMe/internal/utils"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	gohealthcheck "github.com/amarnathdbg101-coder/go/go-healthcheck"
	"go.uber.org/zap"
)

// RouteSetup configures all HTTP routes using components from the centralized Container.
func RouteSetup(c *app.Container) chi.Router {
	uc := c.Controllers.User
	modc := c.Controllers.Moderation
	sc := c.Controllers.Shop
	catc := c.Controllers.Category
	pc := c.Controllers.Product
	upc := c.Controllers.Upload
	resc := c.Controllers.Reservation
	revc := c.Controllers.Review
	invc := c.Controllers.Inventory
	expc := c.Controllers.Expense
	procurementc := c.Controllers.Procurement
	khatac := c.Controllers.Khata
	custKhatac := c.Controllers.CustomerKhata
	ac := c.Controllers.Analytics
	loyc := c.Controllers.Loyalty
	catalogc := c.Controllers.Catalog
	telc := c.Controllers.Telemetry
	aic := c.Controllers.AI
	assistantc := c.Controllers.Assistant
	posc := c.Controllers.POS
	wsc := c.Controllers.WebSocket
	staffc := c.Controllers.Staff
	locationc := c.Controllers.Location
	marginc := c.Controllers.Margin
	drawerc := c.Controllers.CashDrawer
	scanc := c.Controllers.ScanAndGo
	claimc := c.Controllers.DamageClaim
	returnLookupc := c.Controllers.ReturnLookup
	batchc := c.Controllers.Batch
	microPlanc := c.Controllers.MicroPlan
	eodc := c.Controllers.EODReport

	r := chi.NewRouter()

	// Global middlewares
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.TraceMiddleware)
	r.Use(middleware.BotGuard)
	r.Use(middleware.BodySizeGuard(2 << 20))
	r.Use(chimw.Compress(5))
	r.Use(middleware.AdaptiveLimiterMiddleware)
	r.Use(middleware.GlobalRateLimiter.Middleware())
	r.Use(middleware.BanGuard(c.Repos.Moderation))
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.IdempotencyGuard)

	// Production Health, Readiness & Liveness Orchestrator (Render / Kubernetes / Docker)
	// Powered by zero-dependency go-healthcheck with 1s ping caching to protect database from health probes
	healthManager := gohealthcheck.New(gohealthcheck.Config{
		Timeout:  2 * time.Second,
		CacheTTL: 1 * time.Second,
	})
	healthManager.RegisterReadiness(gohealthcheck.NewCheckFunc("postgres", func(ctx context.Context) error {
		return c.DB.Ping(ctx)
	}))
	healthManager.RegisterLiveness(gohealthcheck.NewMemoryChecker("heap_memory", 1024*1024*1024)) // 1GB heap limit

	r.Get("/health", healthManager.ReadyHandler())
	r.Get("/readyz", healthManager.ReadyHandler())
	r.Get("/livez", healthManager.LiveHandler())

	// Centralized Feature Registry & Discovery endpoint
	r.Get("/features", c.Features.ServeHTTP)
	r.Get("/api/features", c.Features.ServeHTTP)

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
	r.Group(func(r chi.Router) {

		r.Get("/sitemap.xml", sc.GenerateDynamicSitemapXML)                // Search Engine Crawlers Dynamic XML Sitemap
		r.Get("/catalog/home-feed", sc.GetHomeFeed)                        // Customer
		r.Get("/catalog/barcode/{code}", catalogc.Lookup)                  // Both - Master Barcode Catalog Lookup
		r.Get("/catalog/lookup", catalogc.Lookup)                          // Both - query param fallback ?barcode=
		r.Get("/catalog/search", catalogc.Search)                          // Both - Master Barcode Catalog Search
		r.Get("/home-feed", sc.GetHomeFeed)                                // Customer
		// Isolated AI Concurrency Bulkhead (max 8 parallel AI queries to prevent CPU starvation)
		r.Group(func(r chi.Router) {
			r.Use(middleware.BulkheadGuard(middleware.AIBulkhead, "AI Studio & Vision Engine"))
			r.Post("/ai/customer-chat", aic.CustomerChat)                      // Customer
			r.Post("/ai/scan-product", aic.ScanProduct)                        // Both
			r.Post("/ai/smart-studio-scan", aic.SmartStudioScan)                // Both - Intelligent Vision + Studio Engine
			r.Post("/ai/parse-parchi", aic.ParseParchi)                        // Both
			r.Post("/ai/semantic-search", aic.SemanticSearch)                  // Customer
			r.Post("/ai/voice-bill", aic.VoiceBill)                            // Shop
			r.Post("/ai/marketing-campaign", aic.GenerateMarketingCampaign)   // Shop
			r.Post("/ai/bargain-assist", aic.BargainAssist)                    // Both
		})
		r.Get("/categories", catc.List)                                    // Both
		r.Get("/shops", sc.List)                                           // Customer
		r.Get("/shops/{id}", sc.GetByID)                                   // Customer
		r.Get("/shops/slug/{slug}", sc.GetBySlug)                          // Customer
		r.Get("/shops/{slug}/qr", sc.GetShopQR)                            // Customer
		r.With(middleware.BulkheadGuard(middleware.PDFBulkhead, "Printable Standee PDF Engine")).Get("/shops/{slug}/standee.pdf", sc.GetShopStandeePDF)           // Customer Printable Standee
		r.Get("/shops/{slug}/products", pc.ListByShop)                     // Customer
		r.Get("/shops/{slug}/live", wsc.ServePublicStorefrontWebSocket)    // Real-Time Storefront Live Channel
		r.Get("/shops/{slug}/reviews", revc.List)                          // Customer
		r.Get("/shops/{slug}/offers", loyc.ListOffers)                     // Customer
		r.Get("/offers", loyc.ListAllOffers)                               // Customer
		r.Get("/deals", loyc.ListAllOffers)                                // Customer
		r.Get("/products/search/suggestions", pc.GetSearchSuggestions)    // Fast Autocomplete Suggestions
		r.Get("/products/nearby", pc.FindNearby)                           // Customer
		r.Get("/products", pc.List)                                        // Customer
		r.Get("/products/{id}", pc.GetByID)                                // Customer
		r.Get("/products/slug/{slug}", pc.GetBySlug)                       // Customer
		r.Get("/products/{productId}/variants", locationc.GetPublicVariants) // Public Customer Variants (Size/Color/Shades)
		r.Get("/products/scan/{code}", loyc.ScanProduct)                   // Customer
		r.Post("/products/{id}/notify-me", invc.SubscribeStockAlert)       // Customer
		r.With(middleware.IdempotencyGuard).Post("/products/{id}/make-offer", pc.MakeOffer)                  // Customer
		r.Get("/receipts/{bill_number}", posc.ViewPublicReceiptPDF)        // Customer
		r.Post("/reports/telemetry-error", telc.LogFrontendError)          // Both
		r.Get("/images/*", upc.ServeImage)                                 // Both
	})

	// 🛍️ Customer Route Group (/api/v1/customer)
	r.Route("/api/v1/customer", func(r chi.Router) {
		r.Use(middleware.OptionalJWTAuth(utils.MustLoad().Jwt))

		r.Get("/products/search/suggestions", pc.GetSearchSuggestions)
		r.Get("/products", pc.ListCustomerProducts)
		r.Get("/products/nearby", pc.FindNearby)
		r.Get("/products/{id}", pc.GetByID)
		r.Get("/products/slug/{slug}", pc.GetBySlug)
		r.Get("/products/{productId}/variants", locationc.GetPublicVariants)
		r.Get("/shops", sc.List)
		r.Get("/shops/{id}", sc.GetByID)
		r.Get("/shops/slug/{slug}", sc.GetBySlug)
		r.Get("/shops/{slug}/offers", loyc.ListOffers)
		r.Get("/shops/{slug}/reviews", revc.List)
		r.Get("/offers", loyc.ListAllOffers)
		r.Get("/receipts/{bill_number}", posc.ViewPublicReceiptPDF)

		// In-Store Scan & Go Self Checkout (Customer Cart)
		r.Post("/scan-go/session", scanc.StartSession)
		r.Post("/scan-go/session/{id}/items", scanc.AddItem)
		r.Put("/scan-go/session/{id}/items/{itemId}", scanc.UpdateItem)
		r.Get("/scan-go/session/{id}", scanc.GetSessionDetails)

		// Customer AI Sathi
		r.Post("/ai/chat", aic.CustomerChat)
		r.Post("/ai/parse-parchi", aic.ParseParchi)
		r.Post("/ai/semantic-search", aic.SemanticSearch)
		r.Post("/ai/scan-product", aic.ScanProduct)
		r.Post("/ai/bargain-assist", aic.BargainAssist)

		// Customer Khata, Loyalty & Item Hold Reservations (JWT Required)
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(utils.MustLoad().Jwt))
			r.Get("/loyalty", loyc.GetUserLoyalty)
			r.Post("/shops/{slug}/reviews", revc.AddOrUpdate)

			r.Get("/khata", custKhatac.GetCustomerKhataSummary)
			r.Get("/khata/{khataId}/transactions", custKhatac.GetCustomerKhataPassbook)
			r.Post("/khata/{khataId}/dispute", custKhatac.DisputeTransaction)
			r.Post("/khata/{khataId}/pay-upi", custKhatac.SubmitUPIPayment)
			r.Get("/khata/{khataId}/statement.pdf", custKhatac.DownloadCustomerPDF)

			r.Post("/reservations", resc.Create)
			r.Get("/reservations", resc.ListUserReservations)
			r.Post("/reservations/{id}/cancel", resc.CancelUserReservation)
			r.Get("/ws", wsc.ServeCustomerWebSocket) // Personal live orders & notifications
		})
	})

	r.Route("/api/v1/merchant/ai-assistant", func(r chi.Router) {
		r.Use(middleware.JWTAuth(utils.MustLoad().Jwt))
		r.Use(middleware.RequireRole("merchant", "shop", "shop_owner", "admin"))

		r.With(middleware.BulkheadGuard(middleware.AIBulkhead, "Merchant AI Assistant")).Post("/chat", assistantc.Chat)
		r.Get("/conversations", assistantc.ListConversations)
		r.Get("/conversations/{conversationID}/messages", assistantc.ListMessages)
	})

	// 🏪 Merchant / Shop Owner Route Group (/api/v1/merchant)
	r.Route("/api/v1/merchant", func(r chi.Router) {
		r.Use(middleware.JWTAuth(utils.MustLoad().Jwt))
		r.Use(middleware.RequireRole("merchant", "shop_owner", "staff", "admin"))

		r.Get("/products", pc.ListMerchantProducts)
		r.Get("/categories", catc.ListMyShopCategories)
		r.Post("/categories", catc.CreateShopCategory)
		r.Put("/categories/{id}", catc.UpdateShopCategory)
		r.Delete("/categories/{id}", catc.DeleteShopCategory)

		// Merchant Procurement & Mandi Checklists
		r.Get("/procurement", procurementc.List)
		r.Post("/procurement", procurementc.Create)
		r.Put("/procurement/{id}", procurementc.Update)
		r.Post("/procurement/toggle/{id}", procurementc.Toggle)
		r.Delete("/procurement/{id}", procurementc.Delete)
		r.Post("/procurement/clear-completed", procurementc.ClearCompleted)

		r.Get("/shop/dashboard", sc.GetMerchantDashboard)
		r.Get("/shop/digest", sc.GetDailyDigest)
		r.Patch("/shop/status", sc.ToggleStatus)

		// Merchant Staff Sub-Accounts
		r.Get("/staff", staffc.ListStaff)
		r.Post("/staff", staffc.CreateStaff)
		r.Put("/staff/{id}", staffc.UpdateStaff)
		r.Delete("/staff/{id}", staffc.DeleteStaff)

		// Merchant Store Analytics
		r.Get("/analytics/profit", ac.GetMonthlyProfit)
		r.Get("/analytics/matrix", ac.GetProductMatrix)

		// Merchant Inventory & Reorder Operations
		r.Get("/inventory/low-stock", invc.GetLowStockAlerts)
		r.Post("/inventory/adjust", invc.AdjustStock)
		r.Get("/inventory/reorder-pdf", invc.DownloadReorderSheetPDF)

		// Multi-Store Godowns & Location-Aware Inventory
		r.Get("/locations", locationc.ListLocations)
		r.Post("/locations", locationc.CreateLocation)
		r.Put("/locations/{id}", locationc.UpdateLocation)
		r.Get("/locations/{id}/inventory", locationc.GetLocationInventory)
		r.Post("/locations/adjust", locationc.AdjustStock)

		// Apparel & Beauty Variant Matrix (Sizes, Colors, Shades, Expiry)
		r.Get("/products/{productId}/variants", locationc.ListVariants)
		r.Post("/products/{productId}/variants", locationc.CreateVariant)
		r.Post("/products/{productId}/variants/matrix", locationc.GenerateVariantMatrix)
		r.Put("/products/variants/{variantId}", locationc.UpdateVariant)
		r.Delete("/products/variants/{variantId}", locationc.DeleteVariant)

		// Stock Transfers State Machine (Godown <-> Store Requisitions)
		r.Get("/transfers", locationc.ListTransfers)
		r.Post("/transfers", locationc.CreateTransfer)
		r.Get("/transfers/{id}", locationc.GetTransfer)
		r.Patch("/transfers/{id}/status", locationc.UpdateTransferStatus)

		// Merchant Offers & In-Store Returns
		r.Get("/offers", loyc.ListShopOffers)
		r.Post("/offers", loyc.CreateOffer)
		r.Put("/offers/{id}", loyc.UpdateOffer)
		r.Patch("/offers/{id}/toggle", loyc.ToggleOffer)
		r.Delete("/offers/{id}", loyc.DeleteOffer)
		r.Get("/offers/history", loyc.GetOfferHistory)
		r.Get("/returns", loyc.ListReturns)
		r.Post("/returns", loyc.ProcessReturn)

		// Merchant In-Store Pickup Holds & Counter POS Billing
		r.Get("/reservations", resc.ListShopReservations)
		r.Post("/reservations/verify", resc.VerifyShopReservation)
		r.Post("/reservations/{id}/cancel", resc.CancelShopReservation)
		r.With(middleware.IdempotencyGuard).Post("/pos/checkout", posc.CreateSale)
		r.Get("/pos/summary", posc.GetDailySummary)

		// POS Financial Shield & Counter Velocity Hotkeys
		r.Post("/pos/margin-check", marginc.CheckMargin)
		r.Get("/pos/hotkeys", marginc.ListHotkeys)
		r.Post("/pos/hotkeys", marginc.CreateHotkey)
		r.Delete("/pos/hotkeys/{id}", marginc.DeleteHotkey)
		r.Post("/pos/hotkeys/auto-populate", marginc.AutoPopulateHotkeys)

		// Cash Drawer Shift Accountability & Drawer Telemetry
		r.Get("/pos/drawer/current", drawerc.GetCurrentStatus)
		r.Post("/pos/drawer/open", drawerc.OpenSession)
		r.Post("/pos/drawer/close", drawerc.CloseSession)
		r.Post("/pos/drawer/safe-drop", drawerc.RecordSafeDrop)
		r.Post("/pos/drawer/expense", drawerc.RecordPettyExpense)
		r.Get("/pos/drawer/history", drawerc.ListHistory)

		// Scan & Go Counter Verification & 1-Tap Checkout
		r.Get("/pos/scan-go/verify/{code}", scanc.VerifyAtCounter)
		r.Post("/pos/scan-go/convert/{code}", scanc.ConvertToBill)

		// Company Damage & Expiry Claim Vault
		r.Post("/claims", claimc.CreateClaim)
		r.Get("/claims", claimc.ListClaims)
		r.Patch("/claims/{id}/status", claimc.UpdateClaimStatus)
		r.Get("/claims/distributor/summary", claimc.GetDistributorStatement)

		// Reverse Return Lookup (Receiptless Returns)
		r.Get("/pos/returns/lookup", returnLookupc.LookupReturn)

		// Batch Expiry Radar & Near-Expiry Auto-Markdown Engine
		r.Post("/batches", batchc.CreateBatch)
		r.Get("/batches/near-expiry", batchc.ListNearExpiry)
		r.Post("/batches/markdown-scan", batchc.RunMarkdownScan)

		// Khata Micro-Installments / Daily Chillar Settle
		r.Post("/khata/micro-plans", microPlanc.CreatePlan)
		r.Post("/khata/micro-plans/{id}/pay", microPlanc.RecordPayment)
		r.Get("/khata/micro-plans/{id}", microPlanc.GetPlanSummary)
		r.Get("/khata/micro-plans", microPlanc.ListPlans)
		r.Put("/khata/{id}/transactions/{txId}", khatac.UpdateTransaction)
		r.Delete("/khata/{id}/transactions/{txId}", khatac.DeleteTransaction)

		// Automated End-of-Day P&L PDF via Pure Go gopdf Engine
		r.With(middleware.BulkheadGuard(middleware.PDFBulkhead, "EndOfDay P&L PDF Engine")).Get("/pos/day-close/eod-report.pdf", eodc.DownloadEndOfDayPLPDF)
		r.With(middleware.BulkheadGuard(middleware.PDFBulkhead, "EndOfDay P&L PDF Engine")).Get("/shops/me/pos/eod-report.pdf", eodc.DownloadEndOfDayPLPDF)

		// Merchant AI Copilot & Vision Engines
		r.Post("/ai/copilot", aic.MerchantCopilot)
		r.Post("/ai/scan-product", aic.ScanProduct)
		r.Post("/ai/smart-studio-scan", aic.SmartStudioScan)
		r.Post("/ai/voice-bill", aic.VoiceBill)
		r.Post("/api/ai/voice-bill", aic.VoiceBill)
		r.Post("/ai/marketing-campaign", aic.GenerateMarketingCampaign)
		r.Post("/api/ai/marketing-campaign", aic.GenerateMarketingCampaign)
		r.Post("/ai/supplier-invoice", aic.ParseSupplierInvoice)
		r.Post("/api/ai/supplier-invoice", aic.ParseSupplierInvoice)
		r.Post("/ai/cross-sell", aic.GetCrossSellRecommendations)
		r.Post("/api/ai/cross-sell", aic.GetCrossSellRecommendations)
		r.Get("/ai/dead-stock-bundles", aic.GetDeadStockBundles)
		r.Get("/api/ai/dead-stock-bundles", aic.GetDeadStockBundles)
		r.Post("/ai/recovery-nudge", aic.GenerateSmartRecoveryNudge)
		r.Post("/api/ai/recovery-nudge", aic.GenerateSmartRecoveryNudge)

		// Merchant Udhar Ledger
		r.Get("/khata/summary", khatac.GetSummary)
		r.Get("/khata/customers", khatac.ListCustomers)
		r.Get("/khata/customers/{mobile}", khatac.GetCustomerHistory)
		r.With(middleware.IdempotencyGuard).Post("/khata/credit", khatac.RecordCredit)
		r.With(middleware.IdempotencyGuard).Post("/khata/payment/{mobile}", khatac.RecordPayment)
		r.Get("/khata/reminder/{mobile}", khatac.GetPaymentReminder)
	})

	// 🌐 Direct /api Aliases for POS, Drawer, Claims, Batches, Khata & Scan-Go
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.JWTAuth(utils.MustLoad().Jwt))

		// 1. Live Margin Floor Check & Touch Counter Hotkeys
		r.Post("/pos/margin-check", marginc.CheckMargin)
		r.Get("/pos/hotkeys", marginc.ListHotkeys)
		r.Post("/pos/hotkeys", marginc.CreateHotkey)
		r.Delete("/pos/hotkeys/{id}", marginc.DeleteHotkey)
		r.Post("/pos/hotkeys/auto-populate", marginc.AutoPopulateHotkeys)

		// 2. Cash Drawer Shift Accountability & Safe Drops
		r.Get("/pos/drawer/current", drawerc.GetCurrentStatus)
		r.Post("/pos/drawer/open", drawerc.OpenSession)
		r.Post("/pos/drawer/close", drawerc.CloseSession)
		r.Post("/pos/drawer/safe-drop", drawerc.RecordSafeDrop)
		r.Post("/pos/drawer/expense", drawerc.RecordPettyExpense)
		r.Get("/pos/drawer/history", drawerc.ListHistory)

		// 3. Scan & Go Counter Verification & 1-Tap Checkout
		r.Get("/pos/scan-go/verify/{code}", scanc.VerifyAtCounter)
		r.Post("/pos/scan-go/convert/{code}", scanc.ConvertToBill)

		// 4. Company Damage Claim Vault & Receiptless Return Lookup
		r.Post("/claims", claimc.CreateClaim)
		r.Get("/claims", claimc.ListClaims)
		r.Patch("/claims/{id}/status", claimc.UpdateClaimStatus)
		r.Get("/claims/distributor/summary", claimc.GetDistributorStatement)
		r.Get("/pos/returns/lookup", returnLookupc.LookupReturn)

		// 5. Batch Expiry Radar & Near-Expiry Auto-Markdown Engine
		r.Post("/batches", batchc.CreateBatch)
		r.Get("/batches/near-expiry", batchc.ListNearExpiry)
		r.Post("/batches/markdown-scan", batchc.RunMarkdownScan)

		// 6. Khata Daily Micro-Installments
		r.Post("/khata/micro-plans", microPlanc.CreatePlan)
		r.Post("/khata/micro-plans/{id}/pay", microPlanc.RecordPayment)
		r.Get("/khata/micro-plans/{id}", microPlanc.GetPlanSummary)
		r.Get("/khata/micro-plans", microPlanc.ListPlans)

		// 7. Automated End-of-Day P&L PDF (Pure Go)
		r.With(middleware.BulkheadGuard(middleware.PDFBulkhead, "EndOfDay P&L PDF Engine")).Get("/pos/day-close/eod-report.pdf", eodc.DownloadEndOfDayPLPDF)

		// 8. 🧠 AI Layer Engines
		r.Post("/ai/supplier-invoice", aic.ParseSupplierInvoice)
		r.Post("/api/ai/supplier-invoice", aic.ParseSupplierInvoice)
		r.Post("/ai/cross-sell", aic.GetCrossSellRecommendations)
		r.Post("/api/ai/cross-sell", aic.GetCrossSellRecommendations)
		r.Get("/ai/dead-stock-bundles", aic.GetDeadStockBundles)
		r.Get("/api/ai/dead-stock-bundles", aic.GetDeadStockBundles)
		r.Post("/ai/recovery-nudge", aic.GenerateSmartRecoveryNudge)
		r.Post("/api/ai/recovery-nudge", aic.GenerateSmartRecoveryNudge)
		r.Post("/ai/voice-bill", aic.VoiceBill)
		r.Post("/api/ai/voice-bill", aic.VoiceBill)
		r.Post("/ai/marketing-campaign", aic.GenerateMarketingCampaign)
		r.Post("/api/ai/marketing-campaign", aic.GenerateMarketingCampaign)
	})

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
		r.Get("/shops/me/branches", sc.ListMyBranches)                                          // Shop
		r.Get("/shops/me/dashboard", sc.GetMerchantDashboard)                                  // Shop
		r.Get("/shops/me/qr", sc.GetMyShopQR)                                                   // Shop
		r.Get("/shops/me/standee.pdf", sc.GetMyShopStandeePDF)                                     // Shop Printable Standee
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

		// Shop Product & Category management
		r.Get("/shops/me/categories", catc.ListMyShopCategories)                                           // Shop
		r.Post("/shops/me/categories", catc.CreateShopCategory)                                            // Shop
		r.Put("/shops/me/categories/{id}", catc.UpdateShopCategory)                                        // Shop
		r.Delete("/shops/me/categories/{id}", catc.DeleteShopCategory)                                     // Shop
		r.Get("/shops/me/products", pc.ListMyShopProducts)                                                 // Shop
		r.With(middleware.MutationRateLimiter.Middleware()).Post("/products", pc.Create)                  // Shop
		r.Put("/products/{id}", pc.Update)                                                                 // Shop
		r.Delete("/products/{id}", pc.Delete)                                                              // Shop
		r.Delete("/shops/me/products/batch", pc.BatchDelete)                                               // Shop
		r.Patch("/shops/me/products/batch-status", pc.BatchToggleStatus)                                   // Shop
		r.With(middleware.UploadRateLimiter.Middleware()).Post("/products/images", upc.UploadProductImages) // Shop
		r.Post("/shops/me/products/{id}/markdown", pc.ApplyClearanceMarkdown)                              // Shop
		r.Post("/shops/me/products/bulk-import", pc.BulkImport)                                            // Shop
		r.Get("/shops/me/products/import-template.csv", pc.DownloadImportTemplate)                         // Shop
		r.Get("/shops/me/products/import-template.xlsx", pc.DownloadImportTemplate)                        // Shop
		r.Get("/shops/me/products/import-template", pc.DownloadImportTemplate)                             // Shop                         // Shop

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
		r.Post("/shops/me/pos/sale", posc.CreateSale)
		r.Get("/shops/me/pos/sales", posc.ListSales)                                 // Shop                               // Shop
		r.Post("/shops/me/pos/sales/{billNumber}/cancel", posc.CancelBill)          // Shop
		r.Get("/shops/me/pos/customer-check", posc.GetCustomerCreditCheck)          // Shop
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

		// Shop Procurement & Mandi Purchase Checklists
		r.Get("/shops/me/procurement", procurementc.List)
		r.Post("/shops/me/procurement", procurementc.Create)
		r.Put("/shops/me/procurement/{id}", procurementc.Update)
		r.Post("/shops/me/procurement/toggle/{id}", procurementc.Toggle)
		r.Delete("/shops/me/procurement/{id}", procurementc.Delete)
		r.Post("/shops/me/procurement/clear-completed", procurementc.ClearCompleted)

		// Shop Customer Khata (Udhar & settlement passbook)
		r.Get("/shops/me/khata/summary", khatac.GetSummary)                      // Shop
		r.Get("/shops/me/khata/aging", khatac.GetAgingReport)                    // Shop
		r.Get("/shops/me/khata", khatac.ListCustomers)                           // Shop
		r.Delete("/shops/me/khata/{khataId}", khatac.DeleteCustomer)              // Shop
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
		r.Put("/shops/me/khata/{id}/transactions/{txId}", khatac.UpdateTransaction)            // Shop
		r.Delete("/shops/me/khata/{id}/transactions/{txId}", khatac.DeleteTransaction)        // Shop
		r.Patch("/shops/me/khata/transactions/{id}/notes", khatac.UpdateTransactionNotes) // Shop
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
		r.Get("/shops/me/offers", loyc.ListShopOffers)          // Shop
		r.Post("/shops/me/offers", loyc.CreateOffer)            // Shop
		r.Put("/shops/me/offers/{id}", loyc.UpdateOffer)        // Shop
		r.Patch("/shops/me/offers/{id}/toggle", loyc.ToggleOffer) // Shop
		r.Delete("/shops/me/offers/{id}", loyc.DeleteOffer)     // Shop
		r.Get("/shops/me/offers/history", loyc.GetOfferHistory)  // Shop
		r.Post("/shops/me/returns", loyc.ProcessReturn)          // Shop
		r.Get("/shops/me/returns", loyc.ListReturns)            // Shop

		// In-Store Item Reservations (Customer)
		r.With(middleware.IdempotencyGuard).Post("/reservations", resc.Create)                   // Customer
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

// RouteSetupWithDB initializes a new centralized Container and returns the configured router.
// Retained for backward compatibility.
func RouteSetupWithDB(db *pgxpool.Pool, logger *zap.Logger) chi.Router {
	return RouteSetup(app.NewContainer(db, logger))
}

