// Package services implements business logic for shop management and operations.
package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"

	"go.uber.org/zap"
)

var (
	ErrProductNotFoundInShop = errors.New("product not found in this shop catalog")
	ErrCartSessionEmpty      = errors.New("cannot convert empty scan and go cart into a bill")
)

// ScanAndGoService handles customer self-checkout carts and queue busting at the counter.
type ScanAndGoService struct {
	scanRepo    *repository.ScanAndGoRepo
	shopRepo    *repository.ShopRepo
	productRepo *repository.ProductRepo
	posService  *POSService
	logger      *zap.Logger
}

// NewScanAndGoService initializes ScanAndGoService.
func NewScanAndGoService(
	scanRepo *repository.ScanAndGoRepo,
	shopRepo *repository.ShopRepo,
	productRepo *repository.ProductRepo,
	posService *POSService,
	logger *zap.Logger,
) *ScanAndGoService {
	return &ScanAndGoService{
		scanRepo:    scanRepo,
		shopRepo:    shopRepo,
		productRepo: productRepo,
		posService:  posService,
		logger:      logger,
	}
}

// StartSession creates a new mobile cart session when a customer enters the shop.
func (s *ScanAndGoService) StartSession(ctx context.Context, customerUserID *string, req dto.StartScanSessionRequest) (*dto.ScanSessionResponse, error) {
	shop, err := s.shopRepo.FindByID(ctx, req.ShopID)
	if err != nil {
		return nil, err
	}
	if !shop.IsActive {
		return nil, errors.New("this shop is currently inactive")
	}

	custName := strings.TrimSpace(req.CustomerName)
	if custName == "" {
		custName = "In-Store Shopper"
	}

	sessionModel := &model.ScanAndGoSession{
		ShopID:         req.ShopID,
		CustomerUserID: customerUserID,
		CustomerName:   custName,
		CustomerPhone:  strings.TrimSpace(req.CustomerPhone),
	}

	created, err := s.scanRepo.CreateSession(ctx, sessionModel)
	if err != nil {
		return nil, err
	}

	return &dto.ScanSessionResponse{
		SessionID:     created.ID,
		ShopID:        created.ShopID,
		CustomerName:  created.CustomerName,
		CustomerPhone: created.CustomerPhone,
		SessionCode:   created.SessionCode,
		QRToken:       created.QRToken,
		Status:        created.Status,
		TotalItems:    0,
		Subtotal:      0.00,
		ExpiresAt:     created.ExpiresAt,
		Items:         []*model.ScanAndGoItem{},
	}, nil
}

// AddItemToCart adds a scanned product (by Barcode/SKU or ProductID) into the customer's cart.
func (s *ScanAndGoService) AddItemToCart(ctx context.Context, sessionID string, req dto.AddItemToCartRequest) (*dto.ScanSessionResponse, error) {
	session, err := s.scanRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.Status != model.ScanAndGoActive {
		return nil, repository.ErrScanSessionInactive
	}

	var product *model.Product
	if req.ProductID != "" {
		product, err = s.productRepo.FindByID(ctx, req.ProductID)
	} else if req.Barcode != "" {
		product, err = s.productRepo.FindBySKU(ctx, session.ShopID, req.Barcode)
	} else {
		return nil, errors.New("either product_id or barcode is required")
	}

	if err != nil || product == nil || product.ShopID != session.ShopID {
		return nil, ErrProductNotFoundInShop
	}

	qty := req.Quantity
	if qty <= 0 {
		qty = 1
	}

	item := &model.ScanAndGoItem{
		SessionID:   session.ID,
		ProductID:   product.ID,
		VariantID:   req.VariantID,
		ProductName: product.Name,
		Barcode:     product.SKU,
		Quantity:    qty,
		UnitPrice:   product.Price,
		TotalPrice:  float64(qty) * product.Price,
	}

	if err := s.scanRepo.AddOrUpdateItem(ctx, session.ID, item); err != nil {
		return nil, err
	}

	return s.GetSessionDetails(ctx, session.ID)
}

// UpdateItemQuantity modifies the count or deletes an item from the cart.
func (s *ScanAndGoService) UpdateItemQuantity(ctx context.Context, sessionID, itemID string, quantity int) (*dto.ScanSessionResponse, error) {
	if err := s.scanRepo.UpdateItemQuantity(ctx, sessionID, itemID, quantity); err != nil {
		return nil, err
	}
	return s.GetSessionDetails(ctx, sessionID)
}

// GetSessionDetails retrieves the full cart state and all line items.
func (s *ScanAndGoService) GetSessionDetails(ctx context.Context, sessionID string) (*dto.ScanSessionResponse, error) {
	session, err := s.scanRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	items, err := s.scanRepo.ListItems(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return &dto.ScanSessionResponse{
		SessionID:       session.ID,
		ShopID:          session.ShopID,
		CustomerName:    session.CustomerName,
		CustomerPhone:   session.CustomerPhone,
		SessionCode:     session.SessionCode,
		QRToken:         session.QRToken,
		Status:          session.Status,
		TotalItems:      session.TotalItems,
		Subtotal:        session.Subtotal,
		ExpiresAt:       session.ExpiresAt,
		ConvertedBillID: session.ConvertedBillID,
		Items:           items,
	}, nil
}

// VerifySessionAtCounter allows cashier to peek customer cart using 6-digit code or QR token.
func (s *ScanAndGoService) VerifySessionAtCounter(ctx context.Context, shopOwnerUserID, sessionCodeOrToken string) (*dto.ScanSessionResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	session, err := s.scanRepo.GetSessionByCode(ctx, shop.ID, sessionCodeOrToken)
	if err != nil {
		return nil, err
	}

	return s.GetSessionDetails(ctx, session.ID)
}

// ConvertToBill turns a customer's verified Scan & Go cart into an official POS sale with 1-tap checkout.
func (s *ScanAndGoService) ConvertToBill(
	ctx context.Context,
	shopOwnerUserID string,
	sessionCodeOrToken string,
	req dto.ConvertToBillRequest,
) (*dto.POSSaleResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		return nil, err
	}

	session, err := s.scanRepo.GetSessionByCode(ctx, shop.ID, sessionCodeOrToken)
	if err != nil {
		return nil, err
	}

	if session.Status != model.ScanAndGoActive {
		return nil, fmt.Errorf("session cannot be converted: status is %s", session.Status)
	}

	items, err := s.scanRepo.ListItems(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrCartSessionEmpty
	}

	// Prepare POS Sale payload
	posItems := make([]dto.POSSaleItemRequest, 0, len(items))
	for _, it := range items {
		unitPrice := it.UnitPrice
		posItems = append(posItems, dto.POSSaleItemRequest{
			ProductID:   it.ProductID,
			Name:        it.ProductName,
			Quantity:    it.Quantity,
			CustomPrice: &unitPrice,
		})
	}

	custPhone := req.CustomerPhone
	if custPhone == "" {
		custPhone = session.CustomerPhone
	}
	custName := req.CustomerName
	if custName == "" {
		custName = session.CustomerName
	}

	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "cash"
	}

	saleReq := dto.CreatePOSSaleRequest{
		Items:          posItems,
		PaymentMethod:  paymentMethod,
		CustomerPhone:  custPhone,
		CustomerName:   custName,
		DiscountAmount: 0.00,
	}

	if paymentMethod == "split" {
		saleReq.SplitPayments = &dto.POSSplitPayment{
			CashAmount:   req.SplitCash,
			OnlineAmount: req.SplitUPI,
		}
	}

	// Execute through the audited POS sales pipeline (stock deduction, drawer cash telemetry, khata)
	saleResp, err := s.posService.CreateSale(ctx, shopOwnerUserID, saleReq)
	if err != nil {
		return nil, fmt.Errorf("failed to finalize POS sale: %w", err)
	}

	// Mark Scan & Go session as successfully converted
	if saleResp != nil && saleResp.Bill != nil {
		if err := s.scanRepo.MarkConverted(ctx, session.ID, saleResp.Bill.ID); err != nil {
			s.logger.Warn("failed to mark scan session converted", zap.Error(err), zap.String("session_id", session.ID))
		}
	}

	return saleResp, nil
}
