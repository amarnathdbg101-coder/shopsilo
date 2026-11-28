// Package services handles business logic for retail merchant intelligence.
package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"

	"go.uber.org/zap"
)

type CopilotService struct {
	repo     *repository.CopilotRepo
	shopRepo *repository.ShopRepo
	logger   *zap.Logger
}

func NewCopilotService(repo *repository.CopilotRepo, shopRepo *repository.ShopRepo, logger *zap.Logger) *CopilotService {
	return &CopilotService{
		repo:     repo,
		shopRepo: shopRepo,
		logger:   logger,
	}
}

// GetDailyPrescriptions returns the store's 3-action prescriptive brief.
func (s *CopilotService) GetDailyPrescriptions(ctx context.Context, userID string) ([]*model.DailyPrescription, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}
	return s.repo.GetDailyPrescriptions(ctx, shop.ID)
}

// ExecutePrescription handles the 1-tap action execution of a prescription card.
func (s *CopilotService) ExecutePrescription(ctx context.Context, userID, prescriptionID string) error {
	cleanID := strings.TrimSpace(prescriptionID)
	if cleanID == "" {
		return errors.New("prescription_id is required")
	}
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return ErrShopRequired
	}
	return s.repo.MarkPrescriptionExecuted(ctx, shop.ID, cleanID)
}

// GetDeadCapitalSummary returns aggregate dead capital metrics and monthly carrying cost.
func (s *CopilotService) GetDeadCapitalSummary(ctx context.Context, userID string) (*model.DeadCapitalSummary, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}
	return s.repo.GetDeadCapitalSummary(ctx, shop.ID)
}

// EvaluateBargain provides real-time, confidential margin-aware negotiation advice for the cashier.
func (s *CopilotService) EvaluateBargain(ctx context.Context, userID string, req dto.EvaluateBargainRequest) (*model.BargainEvaluation, error) {
	if strings.TrimSpace(req.ProductID) == "" {
		return nil, errors.New("product_id is required")
	}
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, ErrShopRequired
	}

	if math.IsNaN(req.CustomerOfferedPrice) || math.IsInf(req.CustomerOfferedPrice, 0) || req.CustomerOfferedPrice <= 0 || req.CustomerOfferedPrice > 1e8 {
		return nil, errors.New("invalid customer offered price: must be a positive realistic number")
	}

	intel, err := s.repo.GetProductIntelligence(ctx, shop.ID, req.ProductID)
	if err != nil {
		return nil, err
	}

	psych, err := s.repo.GetCustomerPsychology(ctx, shop.ID, req.CustomerPhone)
	if err != nil {
		// Log and continue with standard baseline
		s.logger.Warn("customer psychology lookup failed, using baseline", zap.Error(err))
		psych = &model.CustomerPsychology{RFMSegment: "REGULAR", BargainTendencyScore: 0.50}
	}

	costPrice := intel.CostPrice
	originalPrice := intel.ProductPrice
	if costPrice <= 0 && originalPrice > 0 {
		// Conservative safety fallback: If cost was unrecorded by merchant, assume 70% cost basis to prevent margin bleed
		costPrice = math.Round(originalPrice * 0.70)
	}

	hardFloorPrice := intel.FloorPrice
	if hardFloorPrice <= costPrice {
		// Absolute safety rule: Never sell below Cost + 12%
		hardFloorPrice = math.Max(costPrice*1.12, costPrice+15.0)
	}

	offered := req.CustomerOfferedPrice
	isAcceptable := false
	var suggestedCounterOffer float64
	var netProfit float64
	var script string
	var warning string

	// Decision Engine logic
	if offered < costPrice {
		// DANGER: Customer offered below purchase cost
		isAcceptable = false
		suggestedCounterOffer = hardFloorPrice
		netProfit = suggestedCounterOffer - costPrice
		warning = fmt.Sprintf("Customer price ₹%.0f is BELOW your purchase cost (₹%.0f)! Strictly reject.", offered, costPrice)
		script = fmt.Sprintf("Bhaiya ₹%.0f me to hamari khareed bhi nahi aati. ₹%.0f me final kariye, isse kam dukan me sambhav nahi hai.", offered, suggestedCounterOffer)
	} else if offered < hardFloorPrice {
		// Below hard floor margin
		isAcceptable = false
		if intel.IsDeadStock {
			// If item has been sitting for > 60 days, allow near-cost clearance to free blocked capital
			deadStockFloor := math.Max(costPrice*1.05, costPrice+10.0)
			if offered >= deadStockFloor {
				isAcceptable = true
				suggestedCounterOffer = offered
				netProfit = offered - costPrice
				warning = "Dead stock clearance deal accepted (Recovers blocked capital)."
				script = fmt.Sprintf("Thik hai bhaiya, aap purane customer ho isliye ₹%.0f me de rahe hain.", offered)
			} else {
				suggestedCounterOffer = deadStockFloor
				netProfit = suggestedCounterOffer - costPrice
				script = fmt.Sprintf("Bhaiya kam se kam ₹%.0f lagaiye, dukan ka kharcha bhi nikalna mushkil ho jayega.", suggestedCounterOffer)
			}
		} else {
			suggestedCounterOffer = hardFloorPrice
			netProfit = suggestedCounterOffer - costPrice
			script = fmt.Sprintf("Bhaiya ₹%.0f me final kardo, is rate me market me kahi nahi milega.", suggestedCounterOffer)
		}
	} else {
		// Offered price meets or exceeds floor price!
		isAcceptable = true
		suggestedCounterOffer = offered
		netProfit = offered - costPrice
		if psych.RFMSegment == "VIP_GOLD" {
			script = fmt.Sprintf("Bilkul bhaiya, aapke liye ₹%.0f done karte hain. Dhanyawaad!", offered)
		} else {
			script = fmt.Sprintf("Thik hai bhaiya, ₹%.0f me pack kar dete hain.", offered)
		}
	}

	eval := &model.BargainEvaluation{
		ProductID:             intel.ProductID,
		ProductName:           intel.ProductName,
		OriginalPrice:         originalPrice,
		CustomerOfferedPrice:  offered,
		CostPrice:             costPrice,
		HardFloorPrice:        hardFloorPrice,
		IsAcceptable:          isAcceptable,
		SuggestedCounterOffer: math.Round(suggestedCounterOffer*100) / 100,
		NetProfitINR:          math.Round(netProfit*100) / 100,
		InventoryAgeDays:      intel.DaysSinceLastSale,
		ABCTier:               intel.ABCTier,
		XYZTier:               intel.XYZTier,
		CustomerRFM:           psych.RFMSegment,
		SalesScript:           script,
		WarningMessage:        warning,
	}

	return eval, nil
}

// RecalculateIntelligence runs on-demand or nightly ABC-XYZ demand recalculation.
func (s *CopilotService) RecalculateIntelligence(ctx context.Context, userID string) error {
	shop, err := s.shopRepo.FindByUserID(ctx, userID)
	if err != nil {
		return ErrShopRequired
	}
	if err := s.repo.RecalculateABCXYZ(ctx, shop.ID); err != nil {
		return err
	}
	return s.repo.GenerateDailyPrescriptions(ctx, shop.ID)
}
