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
	"time"
)

var (
	ErrKhataCustomerNotFound = errors.New("khata customer not found")
	ErrInvalidPaymentAmount  = errors.New("payment amount must be greater than zero")
)

type KhataService struct {
	khataRepo *repository.KhataRepo
	shopRepo  *repository.ShopRepo
	userRepo  *repository.UserRepo
}

func NewKhataService(khataRepo *repository.KhataRepo, shopRepo *repository.ShopRepo, userRepo *repository.UserRepo) *KhataService {
	return &KhataService{
		khataRepo: khataRepo,
		shopRepo:  shopRepo,
		userRepo:  userRepo,
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
		strings.TrimSpace(input.ParchiImageURL),
		strings.TrimSpace(input.ItemsSummary),
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
		"",
		"",
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
		closureReqBy := ""
		if c.ClosureRequestedBy != nil {
			closureReqBy = *c.ClosureRequestedBy
		}
		closureOTP := ""
		if c.ClosureRequestedBy != nil && *c.ClosureRequestedBy == "CUSTOMER" && c.ClosureOTP != nil {
			closureOTP = *c.ClosureOTP
		}

		res = append(res, dto.KhataCustomerResponse{
			ID:                 c.ID,
			CustomerName:       c.CustomerName,
			CustomerMobile:     c.CustomerMobile,
			CreditLimit:        c.CreditLimit,
			CurrentBalance:     c.CurrentBalance,
			ClosureStatus:      c.ClosureStatus,
			ClosureRequestedBy: closureReqBy,
			ClosureOTP:         closureOTP,
			IsRegistered:       c.IsRegistered,
			LastActivityAt:     c.UpdatedAt.Format("2006-01-02 15:04"),
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

	return utils.RenderPDFWithConcurrencyLimit(ctx, func() ([]byte, error) {
		return utils.GenerateKhataStatementPDF(customer, transactions, shop)
	})
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

// GetCustomerKhataSummary returns the customer's total multi-shop credit portfolio.
func (s *KhataService) GetCustomerKhataSummary(ctx context.Context, customerUserID string) (*dto.CustomerKhataSummaryResponse, error) {
	u, err := s.userRepo.FindByID(ctx, customerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customer profile: %w", err)
	}

	phone := ""
	if u != nil {
		phone = u.Phone
	}

	shops, err := s.khataRepo.FindCustomerKhatasByUserIDOrPhone(ctx, customerUserID, phone)
	if err != nil {
		return nil, err
	}

	totalDue := 0.0
	for _, item := range shops {
		totalDue += item.CurrentBalance
	}

	return &dto.CustomerKhataSummaryResponse{
		TotalMarketDue:  totalDue,
		TotalShopsCount: len(shops),
		Shops:           shops,
	}, nil
}

// GetCustomerKhataPassbook returns the detailed transaction ledger for a specific shop.
func (s *KhataService) GetCustomerKhataPassbook(ctx context.Context, customerUserID, khataID string) (*dto.CustomerKhataPassbookResponse, error) {
	u, err := s.userRepo.FindByID(ctx, customerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customer profile: %w", err)
	}

	phone := ""
	if u != nil {
		phone = u.Phone
	}

	shopItem, customerName, transactions, err := s.khataRepo.GetCustomerKhataWithShop(ctx, khataID, customerUserID, phone)
	if err != nil {
		if errors.Is(err, repository.ErrKhataNotFound) {
			return nil, ErrKhataCustomerNotFound
		}
		return nil, err
	}

	return &dto.CustomerKhataPassbookResponse{
		Shop:         *shopItem,
		CustomerName: customerName,
		Transactions: transactions,
	}, nil
}

// DisputeTransaction flags an incorrect credit entry with a reason.
func (s *KhataService) DisputeTransaction(ctx context.Context, customerUserID, khataID string, input dto.DisputeTransactionRequest) error {
	u, err := s.userRepo.FindByID(ctx, customerUserID)
	if err != nil {
		return fmt.Errorf("failed to retrieve customer profile: %w", err)
	}

	phone := ""
	if u != nil {
		phone = u.Phone
	}

	// Verify that this khata belongs to the customer
	_, _, _, err = s.khataRepo.GetCustomerKhataWithShop(ctx, khataID, customerUserID, phone)
	if err != nil {
		if errors.Is(err, repository.ErrKhataNotFound) {
			return ErrKhataCustomerNotFound
		}
		return err
	}

	return s.khataRepo.DisputeTransaction(ctx, khataID, input.TransactionID, input.Reason)
}

// SubmitUPIPayment records a customer's direct UPI settlement with UTR reference.
func (s *KhataService) SubmitUPIPayment(ctx context.Context, customerUserID, khataID string, input dto.SubmitUPIPaymentRequest) (*model.KhataTransaction, error) {
	if input.Amount <= 0 {
		return nil, ErrInvalidPaymentAmount
	}

	u, err := s.userRepo.FindByID(ctx, customerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customer profile: %w", err)
	}

	phone := ""
	if u != nil {
		phone = u.Phone
	}

	// Verify that this khata belongs to the customer
	_, _, _, err = s.khataRepo.GetCustomerKhataWithShop(ctx, khataID, customerUserID, phone)
	if err != nil {
		if errors.Is(err, repository.ErrKhataNotFound) {
			return nil, ErrKhataCustomerNotFound
		}
		return nil, err
	}

	return s.khataRepo.SubmitUPIPayment(ctx, khataID, input.Amount, input.UPIRefNo, input.Notes)
}

// GenerateCustomerStatementPDF generates an official statement PDF for an authenticated customer.
func (s *KhataService) GenerateCustomerStatementPDF(ctx context.Context, customerUserID, khataID string) ([]byte, error) {
	u, err := s.userRepo.FindByID(ctx, customerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customer profile: %w", err)
	}

	phone := ""
	if u != nil {
		phone = u.Phone
	}

	shopItem, customerName, transactions, err := s.khataRepo.GetCustomerKhataWithShop(ctx, khataID, customerUserID, phone)
	if err != nil {
		if errors.Is(err, repository.ErrKhataNotFound) {
			return nil, ErrKhataCustomerNotFound
		}
		return nil, err
	}

	shop, err := s.shopRepo.FindByID(ctx, shopItem.ShopID)
	if err != nil {
		return nil, err
	}

	customer := &model.CustomerKhata{
		ID:             shopItem.KhataID,
		ShopID:         shopItem.ShopID,
		CustomerName:   customerName,
		CustomerMobile: phone,
		CurrentBalance: shopItem.CurrentBalance,
		CreditLimit:    shopItem.CreditLimit,
	}

	return utils.RenderPDFWithConcurrencyLimit(ctx, func() ([]byte, error) {
		return utils.GenerateKhataStatementPDF(customer, transactions, shop)
	})
}

// RequestKhataClosureByMerchant initiates closure from the merchant side.
func (s *KhataService) RequestKhataClosureByMerchant(ctx context.Context, shopOwnerUserID, khataID string) (*dto.RequestKhataClosureResponse, error) {
	_, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	_, err = s.khataRepo.RequestKhataClosure(ctx, khataID, "SHOP")
	if err != nil {
		return nil, err
	}

	return &dto.RequestKhataClosureResponse{
		KhataID:     khataID,
		Status:      model.KhataClosureStatusPendingOTP,
		Message:     "Closure request initiated. 6-digit OTP has been sent to customer's Mera Khata passbook. Ask customer for the OTP and verify to close.",
		RequestedBy: "SHOP",
	}, nil
}

// VerifyKhataClosureOTPByMerchant validates OTP entered by merchant to close the khata.
func (s *KhataService) VerifyKhataClosureOTPByMerchant(ctx context.Context, shopOwnerUserID, khataID, otp string) error {
	_, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return ErrShopNotFound
		}
		return err
	}

	return s.khataRepo.VerifyKhataClosureOTP(ctx, khataID, otp, "SHOP")
}

// RequestKhataClosureByCustomer initiates closure from customer side.
func (s *KhataService) RequestKhataClosureByCustomer(ctx context.Context, customerUserID, khataID string) (*dto.RequestKhataClosureResponse, error) {
	u, err := s.userRepo.FindByID(ctx, customerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	phone := ""
	if u != nil {
		phone = u.Phone
	}

	_, _, _, err = s.khataRepo.GetCustomerKhataWithShop(ctx, khataID, customerUserID, phone)
	if err != nil {
		return nil, err
	}

	otp, err := s.khataRepo.RequestKhataClosure(ctx, khataID, "CUSTOMER")
	if err != nil {
		return nil, err
	}

	return &dto.RequestKhataClosureResponse{
		KhataID:     khataID,
		Status:      model.KhataClosureStatusPendingOTP,
		Message:     "Khata band karne ka anurodh bhej diya gaya hai. Dukandar ke sath ye OTP share karein: " + otp,
		OTP:         otp,
		RequestedBy: "CUSTOMER",
	}, nil
}

// VerifyKhataClosureOTPByCustomer allows customer to verify closure OTP if requested by shopkeeper.
func (s *KhataService) VerifyKhataClosureOTPByCustomer(ctx context.Context, customerUserID, khataID, otp string) error {
	u, err := s.userRepo.FindByID(ctx, customerUserID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}

	phone := ""
	if u != nil {
		phone = u.Phone
	}

	_, _, _, err = s.khataRepo.GetCustomerKhataWithShop(ctx, khataID, customerUserID, phone)
	if err != nil {
		return err
	}

	return s.khataRepo.VerifyKhataClosureOTP(ctx, khataID, otp, "CUSTOMER")
}

// ReverseTransaction records an official audit reversal for an erroneous transaction (Protected - Shop).
func (s *KhataService) ReverseTransaction(ctx context.Context, shopOwnerUserID, khataID, txID, reason string) (*model.KhataTransaction, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.khataRepo.ReverseTransaction(ctx, shop.ID, khataID, txID, reason)
}

// ResolveDispute resolves a customer's dispute by either accepting (reversing) or rejecting (keeping) it (Protected - Shop).
func (s *KhataService) ResolveDispute(ctx context.Context, shopOwnerUserID, khataID, txID, action, notes string) error {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return ErrShopNotFound
		}
		return err
	}

	return s.khataRepo.ResolveDispute(ctx, shop.ID, khataID, txID, action, notes)
}

// SetCreditOTPProtection configures customer OTP requirement for high-value credit (Protected - Customer).
func (s *KhataService) SetCreditOTPProtection(ctx context.Context, customerUserID, khataID string, required bool, threshold float64) error {
	u, err := s.userRepo.FindByID(ctx, customerUserID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}

	phone := ""
	if u != nil {
		phone = u.Phone
	}

	_, _, _, err = s.khataRepo.GetCustomerKhataWithShop(ctx, khataID, customerUserID, phone)
	if err != nil {
		return err
	}

	return s.khataRepo.SetCreditOTPProtection(ctx, khataID, required, threshold)
}

// SetPromiseToPay updates the promised repayment date and installment target for a customer (Protected - Shop).
func (s *KhataService) SetPromiseToPay(ctx context.Context, shopOwnerUserID, khataID string, input dto.SetPromiseToPayRequest) error {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return ErrShopNotFound
		}
		return err
	}
	_ = shop

	var pDate *time.Time
	if strings.TrimSpace(input.PromiseDate) != "" {
		parsed, pErr := time.Parse(time.RFC3339, input.PromiseDate)
		if pErr != nil {
			parsed, pErr = time.Parse("2006-01-02", input.PromiseDate)
		}
		if pErr == nil {
			pDate = &parsed
		}
	}

	return s.khataRepo.SetPromiseToPay(ctx, khataID, pDate, input.InstallmentTarget)
}

// SetPromiseToPayByCustomer allows customer to set or confirm their repayment target (Protected - Customer).
func (s *KhataService) SetPromiseToPayByCustomer(ctx context.Context, customerUserID, khataID string, input dto.SetPromiseToPayRequest) error {
	u, err := s.userRepo.FindByID(ctx, customerUserID)
	if err != nil {
		return fmt.Errorf("failed to retrieve user: %w", err)
	}

	phone := ""
	if u != nil {
		phone = u.Phone
	}

	_, _, _, err = s.khataRepo.GetCustomerKhataWithShop(ctx, khataID, customerUserID, phone)
	if err != nil {
		return err
	}

	var pDate *time.Time
	if strings.TrimSpace(input.PromiseDate) != "" {
		parsed, pErr := time.Parse(time.RFC3339, input.PromiseDate)
		if pErr != nil {
			parsed, pErr = time.Parse("2006-01-02", input.PromiseDate)
		}
		if pErr == nil {
			pDate = &parsed
		}
	}

	return s.khataRepo.SetPromiseToPay(ctx, khataID, pDate, input.InstallmentTarget)
}

// GetCustomerTrustScore returns the detailed credit rating and repayment insights for a customer (Protected - Shop).
func (s *KhataService) GetCustomerTrustScore(ctx context.Context, shopOwnerUserID, customerMobile string) (*dto.CustomerTrustScoreResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	khata, _, err := s.khataRepo.GetCustomerHistory(ctx, shop.ID, customerMobile)
	if err != nil {
		return nil, err
	}

	score, badge, summary, onTimeRate, avgDays, totalTx, err := s.khataRepo.CalculateTrustScore(ctx, khata.ID)
	if err != nil {
		return nil, err
	}

	return &dto.CustomerTrustScoreResponse{
		CustomerName:       khata.CustomerName,
		CustomerMobile:     khata.CustomerMobile,
		TrustScore:         score,
		TrustBadge:         badge,
		Summary:            summary,
		OnTimeRate:         onTimeRate,
		AverageDaysToPay:   avgDays,
		TotalTransactions:  totalTx,
		CurrentBalance:     khata.CurrentBalance,
		CreditLimit:        khata.CreditLimit,
		CreditOTPRequired:  khata.CreditOTPRequired,
		CreditOTPThreshold: khata.CreditOTPThreshold,
	}, nil
}


