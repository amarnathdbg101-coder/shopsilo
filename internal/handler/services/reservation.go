// Package services handle business logic.
package services

import (
	"context"
	"crypto/rand"
	"regexp"
	"errors"
	"fmt"
	"math"
	"math/big"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"
	"strings"
	"time"
)

type ReservationService struct {
	resRepo     *repository.ReservationRepo
	shopRepo    *repository.ShopRepo
	productRepo *repository.ProductRepo
	userRepo    *repository.UserRepo
}

func (s *ReservationService) SetUserRepo(userRepo *repository.UserRepo) {
	s.userRepo = userRepo
}

func NewReservationService(
	resRepo *repository.ReservationRepo,
	shopRepo *repository.ShopRepo,
	productRepo *repository.ProductRepo,
) *ReservationService {
	return &ReservationService{
		resRepo:     resRepo,
		shopRepo:    shopRepo,
		productRepo: productRepo,
	}
}

func generatePickupCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

func generateReservationNumber() string {
	n, err := rand.Int(rand.Reader, big.NewInt(9000))
	randPart := int64(1000)
	if err == nil {
		randPart = n.Int64() + 1000
	}
	return fmt.Sprintf("HLD-%d-%04d", time.Now().Unix()%100000, randPart)
}

// CreateReservation lets a customer place an in-store hold on a product.
func (s *ReservationService) CreateReservation(ctx context.Context, userID string, input dto.CreateReservationRequest) (*model.Reservation, error) {
	if input.Quantity <= 0 {
		input.Quantity = 1
	}

	// 0. Verify customer account exists & is active
	if s.userRepo != nil {
		user, err := s.userRepo.FindByID(ctx, userID)
		if err != nil || user == nil {
			return nil, ErrUserNotFound
		}
		if !user.IsActive {
			return nil, ErrAccountInactive
		}
	}

	// 1. Fetch product
	prod, err := s.productRepo.FindByID(ctx, input.ProductID)
	if err != nil {
		return nil, errors.New("product not found or unavailable")
	}

	// 2. Fetch and verify shop
	shop, err := s.shopRepo.FindByID(ctx, prod.ShopID)
	if err != nil {
		return nil, ErrShopNotFound
	}
	if !shop.IsActive {
		return nil, errors.New("shop is currently inactive")
	}

	// 3. Default hold duration: 4 hours (bounds: 1-24)
	holdHours := input.HoldHours
	if holdHours < 1 || holdHours > 24 {
		holdHours = 4
	}

	reservation := &model.Reservation{
		ReservationNumber: generateReservationNumber(),
		UserID:            userID,
		ShopID:            prod.ShopID,
		ProductID:         prod.ID,
		Quantity:          input.Quantity,
		PickupCode:        generatePickupCode(),
		Status:            "active",
		ExpiresAt:         time.Now().Add(time.Duration(holdHours) * time.Hour),
		Notes:             strings.TrimSpace(input.Notes),
	}

	tx, err := s.resRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to start database transaction")
	}
	defer tx.Rollback(ctx)

	created, err := s.resRepo.CreateWithTx(ctx, tx, reservation)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficientStock) {
			return nil, ErrProductOutOfStock
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit reservation")
	}

	// Return enriched reservation
	fullRes, err := s.resRepo.FindByID(ctx, created.ID)
	if err != nil {
		return created, nil
	}

	enrichShop(fullRes.Shop)

	productImg := ""
	if len(prod.Images) > 0 {
		productImg = prod.Images[0]
	}

	// Broadcast Complete Real-Time Order & Pickup Alert to Counter in 0ms!
	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShop(prod.ShopID, "NEW_RESERVATION", map[string]interface{}{
		"id":                 fullRes.ID,
		"reservation_number": fullRes.ReservationNumber,
		"pickup_code":        fullRes.PickupCode,
		"product_id":         prod.ID,
		"product_name":       prod.Name,
		"product_image":      productImg,
		"quantity":           fullRes.Quantity,
		"unit_price":         prod.Price,
		"total_price":        prod.Price * float64(fullRes.Quantity),
		"customer_name":      fullRes.CustomerName,
		"customer_phone":     fullRes.CustomerPhone,
		"notes":              fullRes.Notes,
		"status":             fullRes.Status,
		"created_at":         fullRes.CreatedAt.In(utils.IndianLocation).Format(time.RFC3339),
		"expires_at":         fullRes.ExpiresAt.In(utils.IndianLocation).Format(time.RFC3339),
		"created_at_display": utils.IndianNow().Format("02-Jan 03:04 PM"),
		"expires_at_display": fullRes.ExpiresAt.In(utils.IndianLocation).Format("03:04 PM"),
	})

	return fullRes, nil
}

func (s *ReservationService) ListUserReservations(ctx context.Context, userID string, filter dto.ReservationFilter) (*dto.ReservationPaginationResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 50 {
		filter.Limit = 10
	}

	list, totalCount, err := s.resRepo.FindByUserID(ctx, userID, filter)
	if err != nil {
		return nil, err
	}

	for _, r := range list {
		enrichShop(r.Shop)
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(filter.Limit)))
	}

	return &dto.ReservationPaginationResponse{
		Reservations: list,
		TotalCount:   totalCount,
		Page:         filter.Page,
		Limit:        filter.Limit,
		TotalPages:   totalPages,
	}, nil
}

func (s *ReservationService) GetReservationByID(ctx context.Context, id, userID string) (*model.Reservation, error) {
	res, err := s.resRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrReservationNotFound) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}

	if res.UserID != userID {
		return nil, ErrReservationNotFound
	}

	enrichShop(res.Shop)
	return res, nil
}

func (s *ReservationService) CancelUserReservation(ctx context.Context, id, userID string) (*model.Reservation, error) {
	tx, err := s.resRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to start transaction")
	}
	defer tx.Rollback(ctx)

	res, err := s.resRepo.CancelWithTx(ctx, tx, id, userID, "")
	if err != nil {
		if errors.Is(err, repository.ErrReservationNotFound) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit cancellation")
	}

	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToShop(res.ShopID, "RESERVATION_CANCELLED", map[string]interface{}{
		"id":                 res.ID,
		"reservation_number": res.ReservationNumber,
		"product_id":         res.ProductID,
	})

	return res, nil
}

// ListShopReservations lists incoming customer reservations for the authenticated shop owner.
func (s *ReservationService) ListShopReservations(ctx context.Context, shopOwnerUserID string, filter dto.ReservationFilter) (*dto.ReservationPaginationResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 50 {
		filter.Limit = 10
	}

	list, totalCount, err := s.resRepo.FindByShopID(ctx, shop.ID, filter)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(filter.Limit)))
	}

	return &dto.ReservationPaginationResponse{
		Reservations: list,
		TotalCount:   totalCount,
		Page:         filter.Page,
		Limit:        filter.Limit,
		TotalPages:   totalPages,
	}, nil
}

// VerifyShopReservation validates a customer's 6-digit pickup code at the store counter and completes the hold.
func (s *ReservationService) VerifyShopReservation(ctx context.Context, shopOwnerUserID string, input dto.VerifyReservationRequest) (*model.Reservation, error) {
	if strings.TrimSpace(input.PickupCode) == "" && strings.TrimSpace(input.ReservationNumber) == "" {
		return nil, errors.New("either pickup_code or reservation_number is required")
	}

	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	tx, err := s.resRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to start transaction")
	}
	defer tx.Rollback(ctx)

	code := strings.TrimSpace(input.PickupCode)
	resNum := strings.TrimSpace(input.ReservationNumber)
	if code != "" && resNum == "" {
		resNum = code
	} else if resNum != "" && code == "" {
		code = resNum
	}

	completed, err := s.resRepo.VerifyAndCompleteWithTx(ctx, tx, shop.ID, code, resNum)
	if err != nil {
		if errors.Is(err, repository.ErrReservationNotFound) {
			return nil, ErrInvalidVerificationCode
		}
		return nil, err
	}

	// Record completed pickup sale into pos_bills so Daily Revenue, Sales Count, and Receipts sync seamlessly
	var prodName, prodSKU string
	var prodPrice, prodCost float64
	if completed.ProductID != "" {
		_ = tx.QueryRow(ctx, `SELECT name, sku, price, cost_price FROM products WHERE id = $1`, completed.ProductID).Scan(&prodName, &prodSKU, &prodPrice, &prodCost)
	}
	if prodName == "" {
		prodName = "Store Pickup Order"
	}
	unitPrice := prodPrice
	if unitPrice <= 0 {
		unitPrice = 0
	}
	totalAmount := unitPrice * float64(completed.Quantity)
	totalCost := prodCost * float64(completed.Quantity)

	var customerPhone string
	if completed.UserID != "" {
		_ = tx.QueryRow(ctx, `SELECT phone FROM users WHERE id = $1`, completed.UserID).Scan(&customerPhone)
	}
	if customerPhone == "" && completed.Notes != "" {
		re := regexp.MustCompile(`[6-9]\d{9}`)
		if m := re.FindString(completed.Notes); m != "" {
			customerPhone = m
		}
	}

	billNum := fmt.Sprintf("PICKUP-%s", completed.ReservationNumber)
	var billID string
	billQuery := `
		INSERT INTO pos_bills (
			shop_id, bill_number, customer_phone, subtotal, discount_amount, total_amount, total_cost, payment_method, cash_amount, online_amount, khata_amount, created_at
		)
		VALUES ($1, $2, $3, $4, 0, $4, $5, 'pickup_counter', $4, 0, 0, NOW())
		RETURNING id
	`
	if err := tx.QueryRow(ctx, billQuery, shop.ID, billNum, customerPhone, totalAmount, totalCost).Scan(&billID); err == nil && billID != "" {
		var pIDParam interface{} = nil
		if completed.ProductID != "" {
			pIDParam = completed.ProductID
		}
		_, _ = tx.Exec(ctx, `
			INSERT INTO pos_bill_items (bill_id, product_id, product_name, product_sku, quantity, unit_price, unit_cost, total_price)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, billID, pIDParam, prodName, prodSKU, completed.Quantity, unitPrice, prodCost, totalAmount)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit verification")
	}

	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToCustomer(completed.UserID, "RESERVATION_STATUS_MUTATED", map[string]interface{}{
		"id":                 completed.ID,
		"reservation_number": completed.ReservationNumber,
		"status":             "completed",
		"message":            "Your pickup order has been completed at the store counter!",
	})
	wsHub.BroadcastToShop(shop.ID, "POS_SALE_COMPLETED", map[string]interface{}{
		"bill_number":    billNum,
		"total_amount":   totalAmount,
		"payment_method": "pickup_counter",
	})

	return completed, nil
}

// CancelShopReservation allows the shopkeeper to cancel an in-store reservation (e.g. damaged or customer no-show).
func (s *ReservationService) CancelShopReservation(ctx context.Context, id, shopOwnerUserID string) (*model.Reservation, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	tx, err := s.resRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to start transaction")
	}
	defer tx.Rollback(ctx)

	cancelled, err := s.resRepo.CancelWithTx(ctx, tx, id, "", shop.ID)
	if err != nil {
		if errors.Is(err, repository.ErrReservationNotFound) {
			return nil, ErrReservationNotFound
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit cancellation")
	}

	wsHub := GetWebSocketHub(nil)
	wsHub.BroadcastToCustomer(cancelled.UserID, "RESERVATION_STATUS_MUTATED", map[string]interface{}{
		"id":                 cancelled.ID,
		"reservation_number": cancelled.ReservationNumber,
		"status":             "cancelled",
		"message":            "Your pickup order was cancelled by the store.",
	})
	wsHub.BroadcastToShop(shop.ID, "STOCK_CHANGED", map[string]interface{}{
		"product_id": cancelled.ProductID,
		"shop_id":    shop.ID,
	})

	return cancelled, nil
}
