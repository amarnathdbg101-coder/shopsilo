// Package app provides centralized application state, dependency container, feature registry, and worker coordination.
package app

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"shopMe/internal/handler/controller"
	"shopMe/internal/handler/repository"
	"shopMe/internal/handler/services"
)

// Repositories holds all centralized data access repository instances.
type Repositories struct {
	User         *repository.UserRepo
	PhoneVerify  *repository.PhoneVerificationRepo
	Moderation   *repository.ModerationRepo
	Shop         *repository.ShopRepo
	Category     *repository.CategoryRepo
	Product      *repository.ProductRepo
	Reservation  *repository.ReservationRepo
	Review       *repository.ReviewRepo
	Expense      *repository.ExpenseRepo
	Procurement  *repository.ProcurementRepo
	Khata        *repository.KhataRepo
	Loyalty      *repository.LoyaltyRepo
	Catalog      *repository.CatalogRepo
	Telemetry    *repository.TelemetryRepo
	POS          *repository.POSRepo
	Staff        *repository.StaffRepo
	Location     *repository.LocationRepo
	Hotkey       *repository.HotkeyRepo
	CashDrawer   *repository.CashDrawerRepo
	ScanAndGo    *repository.ScanAndGoRepo
	DamageClaim  *repository.DamageClaimRepo
	ReturnLookup *repository.ReturnLookupRepo
	Batch        *repository.BatchRepo
	MicroPlan    *repository.MicroPlanRepo
	AIAssistant  *repository.AIAssistantRepo
}

// Services holds all centralized business logic service instances.
type Services struct {
	User         *services.UserService
	Moderation   *services.ModerationService
	Shop         *services.ShopService
	Category     *services.CategoryService
	Product      *services.ProductService
	Upload       *services.UploadService
	Reservation  *services.ReservationService
	Review       *services.ReviewService
	Inventory    *services.InventoryService
	Expense      *services.ExpenseService
	Procurement  *services.ProcurementService
	Khata        *services.KhataService
	Analytics    *services.AnalyticsService
	Loyalty      *services.LoyaltyService
	Catalog      *services.CatalogService
	Telemetry    *services.TelemetryService
	AI           services.AIService
	POS          *services.POSService
	Staff        *services.StaffService
	Location     *services.LocationService
	Margin       *services.MarginService
	CashDrawer   *services.CashDrawerService
	ScanAndGo    *services.ScanAndGoService
	DamageClaim  *services.DamageClaimService
	ReturnLookup *services.ReturnLookupService
	Batch        *services.BatchService
	MicroPlan    *services.MicroPlanService
	EODPDF       *services.EODPDFService
	Assistant    *services.MerchantAssistantService
}

// Controllers holds all centralized HTTP controller instances.
type Controllers struct {
	User          *controller.UserController
	Moderation    *controller.ModerationController
	Shop          *controller.ShopController
	Category      *controller.CategoryController
	Product       *controller.ProductController
	Upload        *controller.UploadController
	Reservation   *controller.ReservationController
	Review        *controller.ReviewController
	Inventory     *controller.InventoryController
	Expense       *controller.ExpenseController
	Procurement   *controller.ProcurementController
	Khata         *controller.KhataController
	CustomerKhata *controller.CustomerKhataController
	Analytics     *controller.AnalyticsController
	Loyalty       *controller.LoyaltyController
	Catalog       *controller.CatalogController
	Telemetry     *controller.TelemetryController
	AI            controller.AIController
	POS           *controller.POSController
	WebSocket     *controller.WebSocketController
	Staff         *controller.StaffController
	Location      *controller.LocationController
	Margin        *controller.MarginController
	CashDrawer    *controller.CashDrawerController
	ScanAndGo     *controller.ScanAndGoController
	DamageClaim   *controller.DamageClaimController
	ReturnLookup  *controller.ReturnLookupController
	Batch         *controller.BatchController
	MicroPlan     *controller.MicroPlanController
	EODReport     *controller.EODReportController
	Assistant    *controller.MerchantAssistantController
}

// Container is the single source of truth application registry holding all centralized components.
type Container struct {
	DB          *pgxpool.Pool
	Logger      *zap.Logger
	Features    *FeatureRegistry
	Repos       *Repositories
	Services    *Services
	Controllers *Controllers
}

// NewContainer initializes and wires the complete dependency injection graph in one centralized location.
func NewContainer(db *pgxpool.Pool, logger *zap.Logger) *Container {
	// 1. Initialize Repositories (Data Access Layer)
	repos := &Repositories{
		User:         repository.NewUserRepo(db, logger),
		PhoneVerify:  repository.NewPhoneVerificationRepo(db, logger),
		Moderation:   repository.NewModerationRepo(db, logger),
		Shop:         repository.NewShopRepo(db, logger),
		Category:     repository.NewCategoryRepo(db, logger),
		Product:      repository.NewProductRepo(db, logger),
		Reservation:  repository.NewReservationRepo(db, logger),
		Review:       repository.NewReviewRepo(db, logger),
		Expense:      repository.NewExpenseRepo(db, logger),
		Procurement:  repository.NewProcurementRepo(db, logger),
		Khata:        repository.NewKhataRepo(db, logger),
		Loyalty:      repository.NewLoyaltyRepo(db, logger),
		Catalog:      repository.NewCatalogRepo(db, logger),
		Telemetry:    repository.NewTelemetryRepo(db, logger),
		POS:          repository.NewPOSRepo(db, logger),
		Staff:        repository.NewStaffRepo(db, logger),
		Location:     repository.NewLocationRepo(db, logger),
		Hotkey:       repository.NewHotkeyRepo(db, logger),
		CashDrawer:   repository.NewCashDrawerRepo(db, logger),
		ScanAndGo:    repository.NewScanAndGoRepo(db, logger),
		DamageClaim:  repository.NewDamageClaimRepo(db, logger),
		ReturnLookup: repository.NewReturnLookupRepo(db, logger),
		Batch:        repository.NewBatchRepo(db, logger),
		MicroPlan:    repository.NewMicroPlanRepo(db, logger),
		AIAssistant:  repository.NewAIAssistantRepo(db, logger),
	}

	// 2. Initialize Services (Domain Business Logic Layer)
	posService := services.NewPOSService(repos.POS, repos.Shop, repos.Product, repos.Khata)
	posService.SetDrawerRepo(repos.CashDrawer)

	svcs := &Services{
		User:         services.NewUserService(repos.User),
		Moderation:   services.NewModerationService(repos.Moderation, repos.Shop, repos.User),
		Shop:         services.NewShopService(repos.Shop, repos.User, repos.Moderation),
		Category:     services.NewCategoryService(repos.Category, repos.Shop),
		Product:      services.NewProductService(repos.Product, repos.Shop, repos.Category),
		Upload:       services.NewUploadService(repos.User, repos.Shop, repos.Moderation),
		Reservation:  services.NewReservationService(repos.Reservation, repos.Shop, repos.Product),
		Review:       services.NewReviewService(repos.Review, repos.Shop),
		Inventory:    services.NewInventoryService(repos.Product, repos.Shop),
		Expense:      services.NewExpenseService(repos.Expense, repos.Shop),
		Procurement:  services.NewProcurementService(repos.Procurement, repos.Shop),
		Khata:        services.NewKhataService(repos.Khata, repos.Shop, repos.User),
		Analytics:    services.NewAnalyticsService(repos.Product, repos.Shop, repos.Expense),
		Loyalty:      services.NewLoyaltyService(repos.Loyalty, repos.Shop, repos.Product),
		Catalog:      services.NewCatalogService(repos.Catalog),
		Telemetry:    services.NewTelemetryService(repos.Telemetry),
		AI:           services.NewAIService(repos.Shop),
		POS:          posService,
		Staff:        services.NewStaffService(repos.Staff, repos.Shop),
		Location:     services.NewLocationService(repos.Location, repos.Shop, repos.Product),
		Margin:       services.NewMarginService(repos.Shop, repos.Hotkey, logger),
		CashDrawer:   services.NewCashDrawerService(repos.Shop, repos.CashDrawer, logger),
		ScanAndGo:    services.NewScanAndGoService(repos.ScanAndGo, repos.Shop, repos.Product, posService, logger),
		DamageClaim:  services.NewDamageClaimService(repos.DamageClaim, repos.Shop, logger),
		ReturnLookup: services.NewReturnLookupService(repos.ReturnLookup, repos.Shop, logger),
		Batch:        services.NewBatchService(repos.Batch, repos.Shop, repos.Product, logger),
		MicroPlan:    services.NewMicroPlanService(repos.MicroPlan, repos.Shop, repos.User, repos.Khata, logger),
		EODPDF:       services.NewEODPDFService(repos.Shop, repos.POS, repos.CashDrawer, logger),
	}

	// 3. Centralized Inter-Service Wiring
	svcs.User.SetPhoneVerificationRepo(repos.PhoneVerify)
	svcs.User.SetShopRepo(repos.Shop)
	svcs.User.SetCategoryRepo(repos.Category)
	svcs.User.SetKhataRepo(repos.Khata)

	svcs.Reservation.SetUserRepo(repos.User)
	svcs.Product.SetCatalogService(svcs.Catalog)
	svcs.Shop.SetAggregatedRepos(repos.Category, repos.Product, repos.POS, repos.Loyalty)
	svcs.Assistant = services.NewMerchantAssistantService(
		repos.AIAssistant,
		svcs.Shop,
		services.NewMerchantAssistantToolExecutor(svcs.Shop, svcs.Inventory, svcs.Product),
		services.NewGeminiMerchantAssistantModel(nil),
	)

	// 4. Initialize Controllers (HTTP Presentation Layer)
	ctrls := &Controllers{
		User:          controller.NewUserController(svcs.User),
		Moderation:    controller.NewModerationController(svcs.Moderation),
		Shop:          controller.NewShopController(svcs.Shop),
		Category:      controller.NewCategoryController(svcs.Category),
		Product:       controller.NewProductController(svcs.Product, svcs.Shop),
		Upload:        controller.NewUploadController(svcs.Upload),
		Reservation:   controller.NewReservationController(svcs.Reservation),
		Review:        controller.NewReviewController(svcs.Review),
		Inventory:     controller.NewInventoryController(svcs.Inventory),
		Expense:       controller.NewExpenseController(svcs.Expense),
		Procurement:   controller.NewProcurementController(svcs.Procurement),
		Khata:         controller.NewKhataController(svcs.Khata),
		CustomerKhata: controller.NewCustomerKhataController(svcs.Khata),
		Analytics:     controller.NewAnalyticsController(svcs.Analytics),
		Loyalty:       controller.NewLoyaltyController(svcs.Loyalty),
		Catalog:       controller.NewCatalogController(svcs.Catalog),
		Telemetry:     controller.NewTelemetryController(svcs.Telemetry),
		AI:            controller.NewAIController(svcs.AI),
		POS:           controller.NewPOSController(svcs.POS),
		WebSocket:     controller.NewWebSocketController(svcs.Shop, logger),
		Staff:         controller.NewStaffController(svcs.Staff),
		Location:      controller.NewLocationController(svcs.Location),
		Margin:        controller.NewMarginController(svcs.Margin),
		CashDrawer:    controller.NewCashDrawerController(svcs.CashDrawer),
		ScanAndGo:     controller.NewScanAndGoController(svcs.ScanAndGo),
		DamageClaim:   controller.NewDamageClaimController(svcs.DamageClaim),
		ReturnLookup:  controller.NewReturnLookupController(svcs.ReturnLookup),
		Batch:         controller.NewBatchController(svcs.Batch),
		MicroPlan:     controller.NewMicroPlanController(svcs.MicroPlan),
		EODReport:     controller.NewEODReportController(svcs.EODPDF),
		Assistant:     controller.NewMerchantAssistantController(svcs.Assistant),
	}

	return &Container{
		DB:          db,
		Logger:      logger,
		Features:    NewFeatureRegistry(),
		Repos:       repos,
		Services:    svcs,
		Controllers: ctrls,
	}
}
