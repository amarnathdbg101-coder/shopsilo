// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"fmt"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"strings"
)

var (
	ErrKhataCustomerNotFound = errors.New("khata customer not found")
	ErrInvalidPaymentAmount  = errors.New("payment amount must be greater than zero")
)

type KhataService struct {
	khataRepo *repository.KhataRepo
	shopRepo  *repository.ShopRepo
}

func NewKhataService(khataRepo *repository.KhataRepo, shopRepo *repository.ShopRepo) *KhataService {
	return &KhataService{
		khataRepo: khataRepo,
		shopRepo:  shopRepo,
	}
}

// RecordCredit gives udhar to a customer and increases their balance.
func (s *KhataService) RecordCredit(ctx context.Context, shopOwnerUserID string, input dto.RecordCreditTransactionRequest) (*model.KhataTransaction, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	mobile := strings.TrimSpace(input.CustomerMobile)
	name := strings.TrimSpace(input.CustomerName)
	if name == "" {
		name = fmt.Sprintf("Customer %s", mobile)
	}

	tx, err := s.khataRepo.RecordTransaction(
		ctx,
		shop.ID,
		mobile,
		name,
		model.KhataTxTypeGiveCredit,
		input.Amount,
		strings.TrimSpace(input.Notes),
		strings.TrimSpace(input.BillNumber),
		"",
	)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// RecordPayment records payment received from customer (settlement) and decreases balance.
func (s *KhataService) RecordPayment(ctx context.Context, shopOwnerUserID, customerMobile string, input dto.RecordPaymentRequest) (*model.KhataTransaction, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	mobile := strings.TrimSpace(customerMobile)
	name := strings.TrimSpace(input.CustomerName)

	tx, err := s.khataRepo.RecordTransaction(
		ctx,
		shop.ID,
		mobile,
		name,
		model.KhataTxTypeReceivePayment,
		input.Amount,
		strings.TrimSpace(input.Notes),
		"",
		strings.TrimSpace(input.PaymentMode),
	)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// ListCustomers returns all customers with their balances.
func (s *KhataService) ListCustomers(ctx context.Context, shopOwnerUserID, search string, onlyWithBalance bool) ([]dto.KhataCustomerResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	customers, err := s.khataRepo.ListCustomers(ctx, shop.ID, search, onlyWithBalance)
	if err != nil {
		return nil, err
	}

	res := make([]dto.KhataCustomerResponse, 0, len(customers))
	for _, c := range customers {
		res = append(res, dto.KhataCustomerResponse{
			ID:             c.ID,
			CustomerName:   c.CustomerName,
			CustomerMobile: c.CustomerMobile,
			CurrentBalance: c.CurrentBalance,
			LastActivityAt: c.UpdatedAt.Format("2006-01-02 15:04"),
		})
	}

	return res, nil
}

// GetCustomerHistory returns customer details and full transaction passbook.
func (s *KhataService) GetCustomerHistory(ctx context.Context, shopOwnerUserID, customerMobile string) (*dto.KhataDetailResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	customer, transactions, err := s.khataRepo.GetCustomerHistory(ctx, shop.ID, strings.TrimSpace(customerMobile))
	if err != nil {
		if errors.Is(err, repository.ErrKhataNotFound) {
			return nil, ErrKhataCustomerNotFound
		}
		return nil, err
	}

	return &dto.KhataDetailResponse{
		Customer:     *customer,
		Transactions: transactions,
	}, nil
}

// GetSummary returns total market udhar and customer count.
func (s *KhataService) GetSummary(ctx context.Context, shopOwnerUserID string) (*model.KhataSummary, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.khataRepo.GetSummary(ctx, shop.ID)
}
