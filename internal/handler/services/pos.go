// Package services handle business logic.
package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"
	"strings"
	"sync"
	"time"
)

type POSService struct {
	posRepo     *repository.POSRepo
	shopRepo    *repository.ShopRepo
	productRepo *repository.ProductRepo
	khataRepo   *repository.KhataRepo
}

func NewPOSService(posRepo *repository.POSRepo, shopRepo *repository.ShopRepo, productRepo *repository.ProductRepo, khataRepo *repository.KhataRepo) *POSService {
	return &POSService{
		posRepo:     posRepo,
		shopRepo:    shopRepo,
		productRepo: productRepo,
		khataRepo:   khataRepo,
	}
}

func generateBillNumber() string {
	n, err := rand.Int(rand.Reader, big.NewInt(9000))
	randPart := int64(1000)
	if err == nil {
		randPart = n.Int64() + 1000
	}
	return fmt.Sprintf("BIL-%s-%04d", time.Now().Format("060102"), randPart)
}

// CreateSale creates a fast walk-in counter sale, deducts stock, credits loyalty points, and generates bill.
func (s *POSService) CreateSale(ctx context.Context, shopOwnerUserID string, input dto.CreatePOSSaleRequest) (*dto.POSSaleResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	if input.PaymentMethod == "credit" && strings.TrimSpace(input.CustomerPhone) == "" {
		return nil, errors.New("customer phone number is required when billing on credit (udhar)")
	}

	if len(input.Items) == 0 {
		return nil, errors.New("at least one item is required to create a bill")
	}

	var billItems []*model.POSBillItem
	var subtotal float64
	var totalCost float64

	// Concurrently fetch all products in parallel using goroutines
	type productFetchResult struct {
		index int
		prod  *model.Product
		err   error
	}

	fetchChan := make(chan productFetchResult, len(input.Items))
	var wg sync.WaitGroup

	for i, it := range input.Items {
		wg.Add(1)
		go func(idx int, productID string) {
			defer wg.Done()
			p, err := s.productRepo.FindByID(ctx, productID)
			fetchChan <- productFetchResult{index: idx, prod: p, err: err}
		}(i, it.ProductID)
	}

	wg.Wait()
	close(fetchChan)

	prods := make([]*model.Product, len(input.Items))
	for r := range fetchChan {
		if r.err != nil {
			return nil, fmt.Errorf("product %s not found", input.Items[r.index].ProductID)
		}
		prods[r.index] = r.prod
	}

	// Validate each product belongs to shop and has stock
	for i, it := range input.Items {
		prod := prods[i]
		if prod.ShopID != shop.ID {
			return nil, fmt.Errorf("product '%s' does not belong to your shop", prod.Name)
		}

		if prod.Inventory != nil {
			avail := prod.Inventory.Quantity - prod.Inventory.ReservedQuantity
			if avail < it.Quantity {
				return nil, fmt.Errorf("insufficient stock for '%s' (available: %d, requested: %d)", prod.Name, avail, it.Quantity)
			}
		}

		unitPrice := prod.Price
		if it.CustomPrice != nil && *it.CustomPrice >= 0 {
			unitPrice = *it.CustomPrice
		}
		lineTotal := unitPrice * float64(it.Quantity)
		lineCost := prod.CostPrice * float64(it.Quantity)

		subtotal += lineTotal
		totalCost += lineCost

		billItems = append(billItems, &model.POSBillItem{
			ProductID:   prod.ID,
			ProductName: prod.Name,
			ProductSKU:  prod.SKU,
			Quantity:    it.Quantity,
			UnitPrice:   unitPrice,
			UnitCost:    prod.CostPrice,
			TotalPrice:  lineTotal,
		})
	}

	discount := input.DiscountAmount
	if discount < 0 {
		discount = 0
	}
	totalAmount := subtotal - discount
	if totalAmount < 0 {
		totalAmount = 0
	}

	bill := &model.POSBill{
		ShopID:         shop.ID,
		BillNumber:     generateBillNumber(),
		CustomerPhone:  strings.TrimSpace(input.CustomerPhone),
		Subtotal:       subtotal,
		DiscountAmount: discount,
		TotalAmount:    totalAmount,
		TotalCost:      totalCost,
		PaymentMethod:  input.PaymentMethod,
		Items:          billItems,
		Shop:           shop,
	}

	tx, err := s.posRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	createdBill, pointsAwarded, err := s.posRepo.CreateSaleWithTx(ctx, tx, bill)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit sale")
	}

	if createdBill.PaymentMethod == "credit" && s.khataRepo != nil {
		_, _ = s.khataRepo.RecordTransaction(
			ctx,
			shop.ID,
			createdBill.CustomerPhone,
			"Walk-in Customer",
			model.KhataTxTypeGiveCredit,
			createdBill.TotalAmount,
			fmt.Sprintf("POS Bill %s", createdBill.BillNumber),
			createdBill.BillNumber,
			"",
		)
	}

	receiptURL := fmt.Sprintf("/shops/me/pos/receipts/%s.pdf", createdBill.BillNumber)

	return &dto.POSSaleResponse{
		Bill:                  createdBill,
		ReceiptURL:            receiptURL,
		LoyaltyPointsCredited: pointsAwarded,
	}, nil
}

// GetDailySummary returns today's counter cash and UPI register numbers.
func (s *POSService) GetDailySummary(ctx context.Context, shopOwnerUserID string) (*model.DailySalesSummary, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.posRepo.GetDailySummary(ctx, shop.ID, time.Now())
}

// GenerateReceiptPDF generates the PDF bytes for a digital bill receipt.
func (s *POSService) GenerateReceiptPDF(ctx context.Context, billNumber string) ([]byte, error) {
	bill, err := s.posRepo.GetBillByNumber(ctx, billNumber)
	if err != nil {
		if errors.Is(err, repository.ErrBillNotFound) {
			return nil, repository.ErrBillNotFound
		}
		return nil, err
	}

	return utils.GeneratePOSReceiptPDF(bill)
}
