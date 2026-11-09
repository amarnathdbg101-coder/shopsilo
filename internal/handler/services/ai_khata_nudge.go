package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// KhataRecoveryNudgeResult holds the generated WhatsApp payment reminders and deep links.
type KhataRecoveryNudgeResult struct {
	CustomerID       string  `json:"customer_id"`
	CustomerName     string  `json:"customer_name"`
	CustomerMobile   string  `json:"customer_mobile"`
	CurrentBalance   float64 `json:"current_balance"`
	TrustBadge       string  `json:"trust_badge"`
	DaysOverdue      int     `json:"days_overdue"`
	Tone             string  `json:"tone"` // "gentle", "micro_plan", "formal"
	MessageText      string  `json:"message_text"`
	WhatsAppDeepLink string  `json:"whatsapp_deep_link"`
	UPIPaymentLink   string  `json:"upi_payment_link"`
}

// AIKhataNudgeEngine generates psychology-based respectful WhatsApp recovery messages.
type AIKhataNudgeEngine struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewAIKhataNudgeEngine initializes the Khata payment recovery nudge engine.
func NewAIKhataNudgeEngine(db *pgxpool.Pool, logger *zap.Logger) *AIKhataNudgeEngine {
	return &AIKhataNudgeEngine{
		db:     db,
		logger: logger,
	}
}

// GenerateRecoveryNudge finds the customer ledger and drafts an optimal WhatsApp reminder.
func (e *AIKhataNudgeEngine) GenerateRecoveryNudge(
	ctx context.Context,
	shopID string,
	customerIdentifier string,
	requestedTone string,
	shopName string,
	shopUPIID string,
) (*KhataRecoveryNudgeResult, error) {
	if e.db == nil {
		return nil, errors.New("database connection unavailable")
	}

	cleanIdent := strings.TrimSpace(customerIdentifier)
	if cleanIdent == "" {
		return nil, errors.New("customer name or mobile number is required")
	}

	// 1. Query customer khata account
	query := `
		SELECT id, customer_name, customer_mobile, current_balance, trust_badge, promise_to_pay_date, updated_at
		FROM customer_khata
		WHERE shop_id = $1 AND (customer_mobile = $2 OR LOWER(customer_name) = LOWER($2) OR customer_mobile LIKE '%' || $2 || '%')
		ORDER BY current_balance DESC
		LIMIT 1;
	`
	var khataID, name, mobile, trustBadge string
	var balance float64
	var promiseDate *time.Time
	var updatedAt time.Time

	err := e.db.QueryRow(ctx, query, shopID, cleanIdent).Scan(
		&khataID, &name, &mobile, &balance, &trustBadge, &promiseDate, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("customer '%s' ka khata account nahi mila", cleanIdent)
		}
		return nil, fmt.Errorf("failed to fetch customer khata: %w", err)
	}

	// Calculate overdue days
	daysOverdue := int(time.Since(updatedAt).Hours() / 24)
	if daysOverdue < 0 {
		daysOverdue = 0
	}

	if shopName == "" {
		shopName = "Aapki Apni Dukan"
	}
	if shopUPIID == "" {
		shopUPIID = "shopsilo@okhdfcbank"
	}

	// Default tone selection based on trust badge and overdue time if not explicitly provided
	tone := strings.ToLower(strings.TrimSpace(requestedTone))
	if tone == "" {
		if trustBadge == "HIGH_RISK" || daysOverdue > 30 {
			tone = "formal"
		} else if balance > 2000 {
			tone = "micro_plan"
		} else {
			tone = "gentle"
		}
	}

	upiLink := fmt.Sprintf("upi://pay?pa=%s&pn=%s&am=%.2f&cu=INR",
		url.QueryEscape(shopUPIID), url.QueryEscape(shopName), balance)

	var msgText string
	switch tone {
	case "formal", "strict", "firm":
		msgText = fmt.Sprintf("📋 *Hisab Suchna - %s*\n\nNamaste %s ji,\nAapka dukan par pichle %d dino se kul udhar hisab *₹%.2f* pending hai.\n\nKripya dukan aakar ya niche diye gaye instant UPI link se aaj hi payment clear karein:\n👉 %s\n\nDhanyawad,\n*%s*",
			shopName, name, daysOverdue, balance, upiLink, shopName)

	case "micro_plan", "installment":
		dailyChunk := 100.0
		if balance < 1000 {
			dailyChunk = 50.0
		}
		msgText = fmt.Sprintf("🤝 *Namaste %s ji!*\n\nAasha hai aap parivar sahit kushal hain. Agar ek sath hisab dena mushkil ho, toh koi chinta nahi.\n\nAap hamari dukan par rozana sirf *₹%.0f - ₹%.0f* jama karke apna baaki *₹%.2f* ka hisab aasan kishton me chukta kar sakte hain.\n\nAapka aur hamara rishta hamesha anmol hai!\nUPI Link: %s\n\n-%s",
			name, dailyChunk, dailyChunk*1.5, balance, upiLink, shopName)

	default: // "gentle"
		msgText = fmt.Sprintf("🌸 *Namaste %s ji!*\n\nAasha hai aap aur parivar svasth hain. %s se vinamra nivedan hai ki aapka pichla hisab *₹%.2f* baaki hai.\n\nJab bhi aapko suvidha ho, aakar ya diye gaye UPI link se jama kar sakte hain:\n👉 %s\n\nAapka din shubh rahe!\n*%s*",
			name, shopName, balance, upiLink, shopName)
	}

	// Format universal WhatsApp click-to-chat deep link
	cleanMobile := strings.TrimPrefix(mobile, "+")
	if len(cleanMobile) == 10 {
		cleanMobile = "91" + cleanMobile
	}
	waDeepLink := fmt.Sprintf("https://wa.me/%s?text=%s", cleanMobile, url.QueryEscape(msgText))

	return &KhataRecoveryNudgeResult{
		CustomerID:       khataID,
		CustomerName:     name,
		CustomerMobile:   mobile,
		CurrentBalance:   balance,
		TrustBadge:       trustBadge,
		DaysOverdue:      daysOverdue,
		Tone:             tone,
		MessageText:      msgText,
		WhatsAppDeepLink: waDeepLink,
		UPIPaymentLink:   upiLink,
	}, nil
}
