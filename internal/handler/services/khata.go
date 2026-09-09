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
	"shopMe/internal/reuse"
	"shopMe/internal/utils"
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
		if errors.Is(err, repository.ErrCreditLimitExceeded) {
			return nil, ErrCreditLimitExceeded
		}
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

// UpdateCreditLimit updates the credit cap for a customer.
func (s *KhataService) UpdateCreditLimit(ctx context.Context, shopOwnerUserID, customerMobile string, limit float64) error {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return ErrShopNotFound
		}
		return err
	}

	err = s.khataRepo.UpdateCreditLimit(ctx, shop.ID, customerMobile, limit)
	if err != nil {
		if errors.Is(err, repository.ErrKhataNotFound) {
			return ErrKhataCustomerNotFound
		}
		return err
	}
	return nil
}

// GetAgingReport returns debt aging buckets and overdue customer accounts.
func (s *KhataService) GetAgingReport(ctx context.Context, shopOwnerUserID string) (*dto.KhataAgingReport, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.khataRepo.GetAgingReport(ctx, shop.ID)
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
			CreditLimit:    c.CreditLimit,
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

// GeneratePaymentReminder creates a personalized, polite WhatsApp reminder link and text for an udhar customer.
func (s *KhataService) GeneratePaymentReminder(ctx context.Context, shopOwnerUserID, customerMobile string) (*dto.KhataReminderResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	mobile := strings.TrimSpace(customerMobile)
	customer, _, err := s.khataRepo.GetCustomerHistory(ctx, shop.ID, mobile)
	if err != nil {
		if errors.Is(err, repository.ErrKhataNotFound) {
			return nil, ErrKhataCustomerNotFound
		}
		return nil, err
	}

	var digits strings.Builder
	for _, r := range mobile {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	phone := digits.String()
	if len(phone) == 10 {
		phone = "91" + phone
	}

	var reminderText string
	if customer.CurrentBalance > 0 {
		reminderText = fmt.Sprintf(
			"Namaste %s ji, aapka dukaan %s par kul Rs.%.2f ka hisaab (udhar) baaki hai. Kripya samay par chukta karne ka kasht karein. Dhanyawad!",
			customer.CustomerName, shop.Name, customer.CurrentBalance,
		)
	} else {
		reminderText = fmt.Sprintf(
			"Namaste %s ji, aapka dukaan %s par koi hisaab (udhar) baaki nahi hai. ShopMe par humare sath jude rehne ke liye dhanyawad!",
			customer.CustomerName, shop.Name,
		)
	}

	waURL := fmt.Sprintf("https://wa.me/%s?text=%s", phone, url.QueryEscape(reminderText))

	return &dto.KhataReminderResponse{
		CustomerName:   customer.CustomerName,
		CustomerMobile: customer.CustomerMobile,
		MaskedMobile:   reuse.MaskPhoneNumber(customer.CustomerMobile),
		DueAmount:      customer.CurrentBalance,
		ReminderText:   reminderText,
		WhatsAppURL:    waURL,
	}, nil
}

// GenerateStatementPDF generates the official itemized account statement PDF for a customer.
func (s *KhataService) GenerateStatementPDF(ctx context.Context, shopOwnerUserID, customerMobile string) ([]byte, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	mobile := strings.TrimSpace(customerMobile)
	customer, transactions, err := s.khataRepo.GetCustomerHistory(ctx, shop.ID, mobile)
	if err != nil {
		if errors.Is(err, repository.ErrKhataNotFound) {
			return nil, ErrKhataCustomerNotFound
		}
		return nil, err
	}

	return utils.GenerateKhataStatementPDF(customer, transactions, shop)
}

// GetStatementShareLink generates a WhatsApp message with an itemized statement summary and PDF link.
func (s *KhataService) GetStatementShareLink(ctx context.Context, shopOwnerUserID, customerMobile, baseURL string) (*dto.KhataStatementShareResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	mobile := strings.TrimSpace(customerMobile)
	customer, transactions, err := s.khataRepo.GetCustomerHistory(ctx, shop.ID, mobile)
	if err != nil {
		if errors.Is(err, repository.ErrKhataNotFound) {
			return nil, ErrKhataCustomerNotFound
		}
		return nil, err
	}

	var digits strings.Builder
	for _, r := range mobile {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	phone := digits.String()
	if len(phone) == 10 {
		phone = "91" + phone
	}

	statementURL := fmt.Sprintf("%s/shops/me/khata/%s/statement.pdf", baseURL, mobile)

	shareMessage := fmt.Sprintf(
		"Namaste %s ji, aapka dukaan %s par kul baaki hisaab Rs.%.2f hai (%d transactions). Kripya apna poora passbook statement yahan dekhein: %s\nDhanyawad!",
		customer.CustomerName, shop.Name, customer.CurrentBalance, len(transactions), statementURL,
	)

	waURL := fmt.Sprintf("https://wa.me/%s?text=%s", phone, url.QueryEscape(shareMessage))

	return &dto.KhataStatementShareResponse{
		CustomerName:      customer.CustomerName,
		CustomerMobile:    customer.CustomerMobile,
		MaskedMobile:      reuse.MaskPhoneNumber(customer.CustomerMobile),
		CurrentBalance:    customer.CurrentBalance,
		TotalTransactions: len(transactions),
		StatementURL:      statementURL,
		WhatsAppShareURL:  waURL,
	}, nil
}
