// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"
	"strings"
)

type InventoryService struct {
	productRepo *repository.ProductRepo
	shopRepo    *repository.ShopRepo
}

func NewInventoryService(productRepo *repository.ProductRepo, shopRepo *repository.ShopRepo) *InventoryService {
	return &InventoryService{
		productRepo: productRepo,
		shopRepo:    shopRepo,
	}
}

// AdjustStock allows shop owner to record wholesale stock arrival (+qty) or audited adjustments.
func (s *InventoryService) AdjustStock(ctx context.Context, shopOwnerUserID string, input dto.AdjustStockRequest) (*model.Inventory, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	inv, err := s.productRepo.AdjustInventoryStock(ctx, shop.ID, input.ProductID, input.Adjustment, input.LowStockThreshold)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, errors.New("product not found in your shop")
		}
		return nil, err
	}

	return inv, nil
}

// GetLowStockAlerts returns all products that are at or below the low stock threshold.
func (s *InventoryService) GetLowStockAlerts(ctx context.Context, shopOwnerUserID string) (*dto.LowStockResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	items, err := s.productRepo.GetLowStockProducts(ctx, shop.ID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*dto.LowStockProduct{}
	}

	return &dto.LowStockResponse{
		TotalLowStockItems: len(items),
		Items:              items,
	}, nil
}

// GenerateReorderSheetPDF creates a printable wholesale re-order sheet in PDF.
func (s *InventoryService) GenerateReorderSheetPDF(ctx context.Context, shopOwnerUserID string) ([]byte, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	items, err := s.productRepo.GetLowStockProducts(ctx, shop.ID)
	if err != nil {
		return nil, err
	}

	return utils.GenerateWholesaleReorderPDF(shop, items)
}

// GenerateSupplierReorderWhatsApp creates a WhatsApp click-to-chat purchase order message with low stock items.
func (s *InventoryService) GenerateSupplierReorderWhatsApp(ctx context.Context, shopOwnerUserID, supplierPhone string) (*dto.SupplierReorderWhatsAppResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	items, err := s.productRepo.GetLowStockProducts(ctx, shop.ID)
	if err != nil {
		return nil, err
	}

	cleanSupplierPhone := strings.TrimSpace(supplierPhone)
	var digits strings.Builder
	for _, r := range cleanSupplierPhone {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	phone := digits.String()
	if len(phone) == 10 {
		phone = "91" + phone
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Namaste! Wholesale Purchase Order from %s:\n\n", shop.Name))

	if len(items) == 0 {
		sb.WriteString("Sabhi items currently stock me hain. Koi reorder required nahi hai.")
	} else {
		for i, item := range items {
			sb.WriteString(fmt.Sprintf("%d. %s (SKU: %s)\n   Required Qty: %d\n", i+1, item.Name, item.SKU, item.SuggestedReorderQty))
		}
		sb.WriteString("\nKripya bill aur delivery time confirm karein. Dhanyawad!")
	}

	msg := sb.String()
	var waURL string
	if phone != "" {
		waURL = fmt.Sprintf("https://wa.me/%s?text=%s", phone, url.QueryEscape(msg))
	} else {
		waURL = fmt.Sprintf("https://api.whatsapp.com/send?text=%s", url.QueryEscape(msg))
	}

	return &dto.SupplierReorderWhatsAppResponse{
		SupplierPhone: cleanSupplierPhone,
		ItemsCount:    len(items),
		OrderMessage:  msg,
		WhatsAppURL:   waURL,
	}, nil
}

// SubscribeStockAlert allows shoppers to get notified when an out-of-stock item is restocked.
func (s *InventoryService) SubscribeStockAlert(ctx context.Context, productID string, input dto.CreateStockAlertRequest) error {
	prod, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return ErrProductNotFound
		}
		return err
	}

	phone := strings.TrimSpace(input.CustomerPhone)
	name := strings.TrimSpace(input.CustomerName)
	if name == "" {
		name = "Shopper"
	}

	return s.productRepo.CreateStockAlert(ctx, prod.ID, prod.ShopID, phone, name)
}

// GetDemandWatchlist aggregates unnotified customer interest for out-of-stock and low-stock items.
func (s *InventoryService) GetDemandWatchlist(ctx context.Context, shopOwnerUserID string) (*dto.DemandWatchlistResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	items, err := s.productRepo.GetDemandWatchlist(ctx, shop.ID)
	if err != nil {
		return nil, err
	}

	totalWaiting := 0
	for _, it := range items {
		totalWaiting += it.WaitingCustomersCount
		broadcastCopy := fmt.Sprintf(
			"Namaste! Aapne %s ke liye alert lagaya tha. Yeh item ab hamari dukan %s par restock ho chuka hai! Jaldi karein, limited stock available hai.",
			it.ProductName, shop.Name,
		)
		it.WhatsAppBroadcastCopy = broadcastCopy
		it.WhatsAppBroadcastURL = fmt.Sprintf("https://wa.me/?text=%s", url.QueryEscape(broadcastCopy))
	}

	return &dto.DemandWatchlistResponse{
		TotalDemandItems:      len(items),
		TotalWaitingCustomers: totalWaiting,
		Items:                 items,
	}, nil
}

