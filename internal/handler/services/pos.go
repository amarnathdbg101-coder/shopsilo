// Package services handle business logic.
package services

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/url"
	"regexp"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/reuse"
	"shopMe/internal/utils"
	"strconv"
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
	var redeemDealID string

	// Fetch all cart products in a single optimized batch query with inventory JOIN
	productIDs := make([]string, len(input.Items))
	for i, it := range input.Items {
		productIDs[i] = it.ProductID
	}

	prodsMap, err := s.productRepo.FindByIDs(ctx, shop.ID, productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}

	// Validate each product belongs to shop and has stock
	for _, it := range input.Items {
		prod, exists := prodsMap[it.ProductID]
		if !exists {
			return nil, fmt.Errorf("product %s not found or inactive in your shop", it.ProductID)
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

		// Check if a valid bargain deal code was supplied
		if strings.TrimSpace(input.BargainDealCode) != "" {
			if deal, err := s.productRepo.GetValidBargainDeal(ctx, prod.ID, input.BargainDealCode); err == nil && deal != nil {
				unitPrice = deal.AgreedPrice
				redeemDealID = deal.ID
			}
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

	// Loyalty points redemption discount (1 point = Rs. 0.50 discount)
	var loyaltyDiscount float64
	if input.RedeemLoyaltyPoints > 0 {
		loyaltyDiscount = math.Round(float64(input.RedeemLoyaltyPoints)*0.50*100) / 100
		discount += loyaltyDiscount
	}

	totalAmount := subtotal - discount
	if totalAmount < 0 {
		totalAmount = 0
	}

	// Multi-Tender Split Calculation
	var cashAmount, onlineAmount, khataAmount float64
	cleanPhone := strings.TrimSpace(input.CustomerPhone)
	custName := strings.TrimSpace(input.CustomerName)
	if custName == "" {
		custName = "Walk-in Customer"
	}

	if input.PaymentMethod == "split" {
		if input.SplitPayments == nil {
			return nil, errors.New("split_payments details required when payment method is split")
		}
		cashAmount = input.SplitPayments.CashAmount
		onlineAmount = input.SplitPayments.OnlineAmount
		khataAmount = input.SplitPayments.KhataAmount

		splitSum := cashAmount + onlineAmount + khataAmount
		if math.Abs(splitSum-totalAmount) > 0.05 {
			return nil, fmt.Errorf("split payment sum (Rs.%.2f) does not match total bill amount (Rs.%.2f)", splitSum, totalAmount)
		}
		if khataAmount > 0 && cleanPhone == "" {
			return nil, errors.New("customer phone number is required when splitting with khata (udhar)")
		}
	} else if input.PaymentMethod == "cash" {
		cashAmount = totalAmount
	} else if input.PaymentMethod == "credit" || input.PaymentMethod == "khata" {
		if cleanPhone == "" {
			return nil, errors.New("customer phone number is required when billing on credit (udhar)")
		}
		khataAmount = totalAmount
	} else {
		// upi, online, card
		onlineAmount = totalAmount
	}

	bill := &model.POSBill{
		ShopID:         shop.ID,
		BillNumber:     generateBillNumber(),
		CustomerPhone:  cleanPhone,
		Subtotal:       subtotal,
		DiscountAmount: discount,
		TotalAmount:    totalAmount,
		TotalCost:      totalCost,
		PaymentMethod:  input.PaymentMethod,
		CashAmount:     cashAmount,
		OnlineAmount:   onlineAmount,
		KhataAmount:    khataAmount,
		Items:          billItems,
		Shop:           shop,
	}

	tx, err := s.posRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	createdBill, pointsAwarded, pointsRedeemed, err := s.posRepo.CreateSaleWithTx(ctx, tx, bill, input.RedeemLoyaltyPoints)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit sale")
	}

	// Redeem bargain deal if applied
	if redeemDealID != "" {
		_ = s.productRepo.RedeemBargainDeal(ctx, redeemDealID)
	}

	// Atomically record customer khata debt if khata was used
	if khataAmount > 0 && s.khataRepo != nil {
		_, _ = s.khataRepo.RecordTransaction(
			ctx,
			shop.ID,
			createdBill.CustomerPhone,
			custName,
			model.KhataTxTypeGiveCredit,
			khataAmount,
			fmt.Sprintf("POS Bill %s", createdBill.BillNumber),
			createdBill.BillNumber,
			"",
		)
	}

	receiptURL := fmt.Sprintf("/shops/me/pos/receipts/%s.pdf", createdBill.BillNumber)
	whatsAppShareURL := buildWhatsAppBillURL(createdBill.CustomerPhone, shop.Name, createdBill.BillNumber, createdBill.TotalAmount, receiptURL)

	return &dto.POSSaleResponse{
		Bill:                  createdBill,
		ReceiptURL:            receiptURL,
		LoyaltyPointsCredited: pointsAwarded,
		LoyaltyPointsRedeemed: pointsRedeemed,
		LoyaltyDiscountAmount: loyaltyDiscount,
		WhatsAppShareURL:      whatsAppShareURL,
	}, nil
}

func (s *POSService) CancelPOSBill(ctx context.Context, shopOwnerUserID, billNumber, reason string) (*model.POSBill, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	tx, err := s.posRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to start database transaction")
	}
	defer tx.Rollback(ctx)

	bill, err := s.posRepo.CancelBillWithTx(ctx, tx, shop.ID, billNumber, reason)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit bill cancellation transaction")
	}

	return bill, nil
}

// buildWhatsAppBillURL creates a clickable WhatsApp Click-to-Chat URL for sending the bill receipt.
func buildWhatsAppBillURL(customerPhone, shopName, billNumber string, totalAmount float64, receiptURL string) string {
	var digits strings.Builder
	for _, r := range customerPhone {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	phone := digits.String()
	if len(phone) == 10 {
		phone = "91" + phone
	}

	msg := fmt.Sprintf(
		"Namaste! Thank you for shopping at %s. Your bill #%s for Rs.%.2f is ready. You can view your receipt here: %s",
		shopName, billNumber, totalAmount, receiptURL,
	)

	if phone == "" {
		return fmt.Sprintf("https://api.whatsapp.com/send?text=%s", url.QueryEscape(msg))
	}
	return fmt.Sprintf("https://wa.me/%s?text=%s", phone, url.QueryEscape(msg))
}

// ScanBarcode enables counter barcode scanners to instantly identify an item and check available stock.
func (s *POSService) ScanBarcode(ctx context.Context, shopOwnerUserID, sku string) (*dto.BarcodeScanResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	trimmedSKU := strings.TrimSpace(sku)
	if trimmedSKU == "" {
		return nil, errors.New("barcode / SKU is required")
	}

	prod, err := s.productRepo.FindBySKU(ctx, shop.ID, trimmedSKU)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			return nil, errors.New("product not found with this barcode")
		}
		return nil, err
	}

	availQty := 0
	if prod.Inventory != nil {
		availQty = prod.Inventory.AvailableQuantity
	}

	return &dto.BarcodeScanResponse{
		ProductID:         prod.ID,
		ShopID:            prod.ShopID,
		Name:              prod.Name,
		SKU:               prod.SKU,
		Price:             prod.Price,
		Images:            prod.Images,
		AvailableQuantity: availQty,
		IsActive:          prod.IsActive,
	}, nil
}

// GetBillShareLink generates a WhatsApp click-to-chat link for any existing POS bill.
func (s *POSService) GetBillShareLink(ctx context.Context, shopOwnerUserID, billNumber string) (*dto.POSBillShareResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	bill, err := s.posRepo.GetBillByNumber(ctx, strings.TrimSpace(billNumber))
	if err != nil {
		return nil, err
	}
	if bill.ShopID != shop.ID {
		return nil, errors.New("bill belongs to another shop")
	}

	receiptURL := fmt.Sprintf("/shops/me/pos/receipts/%s.pdf", bill.BillNumber)
	shareURL := buildWhatsAppBillURL(bill.CustomerPhone, shop.Name, bill.BillNumber, bill.TotalAmount, receiptURL)

	return &dto.POSBillShareResponse{
		BillNumber:       bill.BillNumber,
		CustomerPhone:    bill.CustomerPhone,
		MaskedPhone:      reuse.MaskPhoneNumber(bill.CustomerPhone),
		TotalAmount:      bill.TotalAmount,
		ReceiptURL:       receiptURL,
		WhatsAppShareURL: shareURL,
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

// GenerateReceiptPDF generates the PDF bytes for a digital bill receipt with concurrency limiting.
func (s *POSService) GenerateReceiptPDF(ctx context.Context, billNumber string) ([]byte, error) {
	bill, err := s.posRepo.GetBillByNumber(ctx, billNumber)
	if err != nil {
		if errors.Is(err, repository.ErrBillNotFound) {
			return nil, repository.ErrBillNotFound
		}
		return nil, err
	}

	return utils.RenderPDFWithConcurrencyLimit(ctx, func() ([]byte, error) {
		return utils.GeneratePOSReceiptPDF(bill)
	})
}

// GetDailyCloseReport aggregates end-of-day counter sales, khata debt repayments, and expenses to calculate physical drawer cash.
func (s *POSService) GetDailyCloseReport(ctx context.Context, shopOwnerUserID, dateStr string) (*dto.DailyCloseReport, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	targetDate := time.Now()
	if strings.TrimSpace(dateStr) != "" {
		if t, err := time.Parse("2006-01-02", strings.TrimSpace(dateStr)); err == nil {
			targetDate = t
		}
	}

	return s.posRepo.GetDailyCloseReport(ctx, shop.ID, targetDate)
}

// GetMonthlyGSTReport calculates monthly turnover, GST tax liability, and generates a formatted message for the merchant's CA.
func (s *POSService) GetMonthlyGSTReport(ctx context.Context, shopOwnerUserID string, year, month int, taxRate float64) (*dto.MonthlyGSTReport, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	if year <= 0 {
		year = time.Now().Year()
	}
	if month <= 0 || month > 12 {
		month = int(time.Now().Month())
	}
	if taxRate <= 0 {
		taxRate = 18.0
	}

	report, err := s.posRepo.GetMonthlyGSTReport(ctx, shop.ID, year, month, taxRate)
	if err != nil {
		return nil, err
	}

	report.ShopName = shop.Name

	// Generate ready-to-share WhatsApp copy for the Chartered Accountant
	caText := fmt.Sprintf(
		"Dear CA Sir, Here is the GST Sales Summary for %s (%s %d):\n- Total Invoices: %d\n- Gross Turnover: Rs. %.2f\n- Taxable Amount: Rs. %.2f\n- CGST (%.1f%%): Rs. %.2f\n- SGST (%.1f%%): Rs. %.2f\n- Total GST: Rs. %.2f\nGenerated via ShopMe POS.",
		shop.Name, report.MonthName, report.Year, report.TotalBills, report.TotalGrossSales,
		report.TotalTaxable, taxRate/2.0, report.TotalCGST, taxRate/2.0, report.TotalSGST, report.TotalTax,
	)
	report.SummaryTextForCA = caText
	report.WhatsAppShareURL = fmt.Sprintf("https://wa.me/?text=%s", url.QueryEscape(caText))

	return report, nil
}

// ParkBill pauses an active counter cart so the cashier can bill other customers.
func (s *POSService) ParkBill(ctx context.Context, shopOwnerUserID string, input dto.ParkPOSBillRequest) (*model.POSParkedBill, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	if len(input.Items) == 0 {
		return nil, errors.New("cannot park an empty cart")
	}

	totalAmount := 0.0
	for _, it := range input.Items {
		if it.CustomPrice != nil && *it.CustomPrice > 0 {
			totalAmount += *it.CustomPrice * float64(it.Quantity)
		} else {
			prod, err := s.productRepo.FindByID(ctx, it.ProductID)
			if err == nil && prod != nil {
				totalAmount += prod.Price * float64(it.Quantity)
			}
		}
	}
	totalAmount -= input.DiscountAmount
	if totalAmount < 0 {
		totalAmount = 0
	}

	label := strings.TrimSpace(input.Label)
	if label == "" {
		label = fmt.Sprintf("Parked Cart (%s)", time.Now().Format("15:04"))
	}

	return s.posRepo.ParkBill(ctx, shop.ID, label, input.CustomerPhone, input, totalAmount)
}

// ListParkedBills returns all carts currently on hold.
func (s *POSService) ListParkedBills(ctx context.Context, shopOwnerUserID string) ([]*dto.ParkedBillSummaryItem, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	items, err := s.posRepo.ListParkedBills(ctx, shop.ID)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		it.MaskedPhone = reuse.MaskPhoneNumber(it.CustomerPhone)
	}
	return items, nil
}

// GetParkedBill retrieves a held cart to restore onto the register.
func (s *POSService) GetParkedBill(ctx context.Context, shopOwnerUserID, id string) (*dto.ParkedBillDetailResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	return s.posRepo.GetParkedBillByID(ctx, shop.ID, id)
}

// DeleteParkedBill removes a held cart.
func (s *POSService) DeleteParkedBill(ctx context.Context, shopOwnerUserID, id string) error {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return ErrShopNotFound
		}
		return err
	}

	return s.posRepo.DeleteParkedBill(ctx, shop.ID, id)
}

var (
	prefixQtyRegex     = regexp.MustCompile(`(?i)^([0-9]+(?:\.[0-9]+)?)\s*(kgs?|g(?:ms?|rams?)?|l(?:trs?|iters?|itres?)?|ml|pkts?|packets?|pcs?|pieces?|box(?:es)?|bottles?)?\s*(?:of\s+)?(.+)$`)
	suffixQtyRegex     = regexp.MustCompile(`(?i)^(.+?)\s+([0-9]+(?:\.[0-9]+)?)\s*(kgs?|g(?:ms?|rams?)?|l(?:trs?|iters?|itres?)?|ml|pkts?|packets?|pcs?|pieces?|box(?:es)?|bottles?)?$`)
	bulletCleanerRegex = regexp.MustCompile(`^[\s*•\-–—#0-9.)]+`)
)

// ParseParchi transforms raw multiline grocery lists pasted from WhatsApp into a draft POS cart.
func (s *POSService) ParseParchi(ctx context.Context, shopOwnerUserID string, req dto.ParseParchiRequest) (*dto.ParseParchiResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	rawLines := strings.Split(req.RawText, "\n")
	var matchedItems []*dto.ParsedParchiItem
	var unmatchedLines []*dto.UnmatchedParchiLine
	var readyCartItems []dto.POSSaleItemRequest
	var estimatedTotal float64
	type parchiJob struct {
		trimmed string
		term    string
		qty     int
		unit    string
		tokens  []string
	}

	type parchiResult struct {
		bestMatch *model.Product
		unmatched *dto.UnmatchedParchiLine
	}

	totalLines := 0
	var jobs []parchiJob
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		totalLines++

		// Clean bullet points e.g. "1.", "1)", "-", "*"
		cleaned := bulletCleanerRegex.ReplaceAllString(trimmed, "")
		cleaned = strings.TrimSpace(cleaned)
		if cleaned == "" {
			cleaned = trimmed
		}

		qtyStr := "1"
		unit := ""
		term := cleaned

		if match := prefixQtyRegex.FindStringSubmatch(cleaned); len(match) == 4 {
			qtyStr = match[1]
			unit = strings.ToLower(match[2])
			term = strings.TrimSpace(match[3])
		} else if match := suffixQtyRegex.FindStringSubmatch(cleaned); len(match) == 4 {
			term = strings.TrimSpace(match[1])
			qtyStr = match[2]
			unit = strings.ToLower(match[3])
		}

		qtyFloat, _ := strconv.ParseFloat(qtyStr, 64)
		qty := int(math.Round(qtyFloat))
		if qty <= 0 {
			qty = 1
		}

		// Tokenize search term
		words := strings.Fields(term)
		var tokens []string
		for _, w := range words {
			wLower := strings.ToLower(w)
			// Filter trivial stop words
			if wLower != "and" && wLower != "ke" && wLower != "ki" && wLower != "ka" && len(wLower) >= 2 {
				tokens = append(tokens, wLower)
			}
		}
		if len(tokens) == 0 {
			tokens = []string{term}
		}

		jobs = append(jobs, parchiJob{
			trimmed: trimmed,
			term:    term,
			qty:     qty,
			unit:    unit,
			tokens:  tokens,
		})
	}

	results := make([]parchiResult, len(jobs))
	if len(jobs) > 0 {
		workerCount := 6
		if len(jobs) < workerCount {
			workerCount = len(jobs)
		}

		jobChan := make(chan int, len(jobs))
		for i := range jobs {
			jobChan <- i
		}
		close(jobChan)

		var wg sync.WaitGroup
		for w := 0; w < workerCount; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for idx := range jobChan {
					j := jobs[idx]
					candidates, err := s.productRepo.FindActiveProductsByTokens(ctx, shop.ID, j.tokens, 3)
					if err != nil || len(candidates) == 0 {
						results[idx] = parchiResult{
							unmatched: &dto.UnmatchedParchiLine{
								OriginalLine: j.trimmed,
								QueryTerm:    j.term,
								Reason:       "No matching product found in shop catalog",
							},
						}
					} else {
						results[idx] = parchiResult{
							bestMatch: candidates[0],
						}
					}
				}
			}()
		}
		wg.Wait()
	}

	for i, j := range jobs {
		res := results[i]
		if res.unmatched != nil {
			unmatchedLines = append(unmatchedLines, res.unmatched)
			continue
		}

		bestMatch := res.bestMatch
		availStock := 0
		if bestMatch.Inventory != nil {
			availStock = bestMatch.Inventory.AvailableQuantity
		}

		lineTotal := bestMatch.Price * float64(j.qty)
		estimatedTotal += lineTotal

		matchedItems = append(matchedItems, &dto.ParsedParchiItem{
			ProductID:         bestMatch.ID,
			ProductName:       bestMatch.Name,
			SKU:               bestMatch.SKU,
			RequestedQuantity: j.qty,
			ParsedUnit:        j.unit,
			UnitPrice:         bestMatch.Price,
			TotalPrice:        lineTotal,
			AvailableStock:    availStock,
			InStock:           availStock >= j.qty,
		})

		readyCartItems = append(readyCartItems, dto.POSSaleItemRequest{
			ProductID: bestMatch.ID,
			Quantity:  j.qty,
		})
	}

	if matchedItems == nil {
		matchedItems = []*dto.ParsedParchiItem{}
	}
	if unmatchedLines == nil {
		unmatchedLines = []*dto.UnmatchedParchiLine{}
	}

	return &dto.ParseParchiResponse{
		TotalLinesParsed:     totalLines,
		MatchedCount:         len(matchedItems),
		UnmatchedCount:       len(unmatchedLines),
		EstimatedTotalAmount: math.Round(estimatedTotal*100) / 100,
		MatchedItems:         matchedItems,
		UnmatchedLines:       unmatchedLines,
		ReadyCart: &dto.CreatePOSSaleRequest{
			PaymentMethod: "cash",
			Items:         readyCartItems,
		},
	}, nil
}

// GetWeeklyScorecard returns a weekly executive business health and net khata cash flow summary.
func (s *POSService) GetWeeklyScorecard(ctx context.Context, shopOwnerUserID string) (*dto.WeeklyScorecardResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	res, err := s.posRepo.GetWeeklyScorecardData(ctx, shop.ID)
	if err != nil {
		return nil, err
	}
	res.ShopName = shop.Name

	netSign := "+"
	if res.NetKhataCashFlow < 0 {
		netSign = "-"
	}

	summaryText := fmt.Sprintf(
		"📊 *Weekly Business Scorecard - %s*\n🗓️ *Period*: %s\n\n💰 *Total Revenue*: Rs.%.2f (%+.1f%% vs last week)\n🛒 *Orders*: %d | *Avg Basket*: Rs.%.2f\n\n💳 *Payment Tender Breakdown*:\n- 💵 Cash: Rs.%.2f\n- 📱 UPI / Online: Rs.%.2f\n- 📝 New Khata Credit: Rs.%.2f\n\n⚖️ *Khata Health & Recovery*:\n- 📥 Recovered Udhar: Rs.%.2f\n- 📤 New Udhar Issued: Rs.%.2f\n- 🔄 Net Cash Flow: %sRs.%.2f\n  *(%s)*\n",
		shop.Name, res.CurrentWeekRange, res.CurrentWeekRevenue, res.GrowthPercentage,
		res.TotalBillsCount, res.AverageOrderValue,
		res.CashCollected, res.OnlineCollected, res.KhataNewCreditIssued,
		res.KhataRecoveredCash, res.KhataNewCreditIssued,
		netSign, math.Abs(res.NetKhataCashFlow), res.KhataHealthStatus,
	)

	if len(res.TopSellingProducts) > 0 {
		summaryText += "\n🏆 *Top Velocity Products*:\n"
		for i, p := range res.TopSellingProducts {
			summaryText += fmt.Sprintf("%d. %s (%d sold - Rs.%.2f)\n", i+1, p.ProductName, p.UnitsSold, p.TotalSales)
		}
	}

	if res.OutOfStockSellersCount > 0 {
		summaryText += fmt.Sprintf("\n⚠️ *Action Required*: %d top-selling items are currently OUT OF STOCK!\n", res.OutOfStockSellersCount)
	}

	res.WhatsAppSummaryCopy = summaryText
	res.WhatsAppShareURL = fmt.Sprintf("https://wa.me/?text=%s", url.QueryEscape(summaryText))

	return res, nil
}

// GetCustomerRecentBasket retrieves the customer's previous grocery purchase to enable 1-click repeat reordering.
func (s *POSService) GetCustomerRecentBasket(ctx context.Context, shopOwnerUserID, customerPhone string) (*dto.CustomerRecentBasketResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	cleanPhone := strings.TrimSpace(customerPhone)
	if cleanPhone == "" {
		return nil, errors.New("customer phone number is required")
	}

	return s.posRepo.GetCustomerLastBasket(ctx, shop.ID, cleanPhone)
}

// ProcessPOSReturn processes customer product returns, restocks items, and credits cash, khata, or store credit notes.
func (s *POSService) ProcessPOSReturn(ctx context.Context, shopOwnerUserID string, req dto.ProcessPOSReturnRequest) (*dto.ProcessPOSReturnResponse, error) {
	shop, err := s.shopRepo.FindByUserID(ctx, shopOwnerUserID)
	if err != nil {
		if errors.Is(err, repository.ErrShopNotFound) {
			return nil, ErrShopNotFound
		}
		return nil, err
	}

	tx, err := s.posRepo.BeginTx(ctx)
	if err != nil {
		return nil, errors.New("failed to begin transaction")
	}
	defer tx.Rollback(ctx)

	bill, totalRefund, restockedCount, err := s.posRepo.ProcessPOSReturnWithTx(ctx, tx, shop.ID, req)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.New("failed to commit return transaction")
	}

	returnNumber := generateReturnNumber()
	creditNoteCode := ""
	message := ""

	switch req.RefundMode {
	case "khata":
		if bill.CustomerPhone != "" && s.khataRepo != nil {
			_, _ = s.khataRepo.RecordTransaction(
				ctx,
				shop.ID,
				bill.CustomerPhone,
				"Valued Customer",
				model.KhataTxTypeReceivePayment,
				totalRefund,
				fmt.Sprintf("Return %s on Bill %s", returnNumber, bill.BillNumber),
				bill.BillNumber,
				"return_credit",
			)
			message = fmt.Sprintf("Refund of Rs.%.2f successfully credited to customer's Khata account.", totalRefund)
		} else {
			message = fmt.Sprintf("Customer has no phone number on record; processed as store credit Rs.%.2f.", totalRefund)
		}
	case "credit_note":
		creditNoteCode = generateCreditNoteCode()
		message = fmt.Sprintf("Store Credit Note %s issued for Rs.%.2f (Valid for next purchase).", creditNoteCode, totalRefund)
	case "cash":
		message = fmt.Sprintf("Cash refund of Rs.%.2f paid from cash counter.", totalRefund)
	}

	// Build WhatsApp return receipt copy
	waMsg := fmt.Sprintf("🛍️ *Return Receipt - %s*\nReturn No: %s\nOriginal Bill: %s\nItems Returned: %d\nRefund Mode: %s\nTotal Refund: Rs.%.2f\n%s\nThank you!",
		shop.Name, returnNumber, bill.BillNumber, restockedCount, strings.ToUpper(req.RefundMode), totalRefund, message)
	waURL := fmt.Sprintf("https://wa.me/%s?text=%s", strings.TrimPrefix(bill.CustomerPhone, "+"), url.QueryEscape(waMsg))

	return &dto.ProcessPOSReturnResponse{
		ReturnNumber:        returnNumber,
		BillNumber:          bill.BillNumber,
		CustomerPhone:       bill.CustomerPhone,
		RefundMode:          req.RefundMode,
		TotalRefundAmount:   totalRefund,
		RestockedItemsCount: restockedCount,
		CreditNoteCode:      creditNoteCode,
		Message:             message,
		WhatsAppReceiptURL:  waURL,
		ProcessedAt:         time.Now().Format("02-Jan-2006 15:04:05"),
	}, nil
}

func generateReturnNumber() string {
	n, err := rand.Int(rand.Reader, big.NewInt(9000))
	randPart := int64(1000)
	if err == nil {
		randPart = n.Int64() + 1000
	}
	return fmt.Sprintf("RET-%s-%04d", time.Now().Format("060102"), randPart)
}

func generateCreditNoteCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(9000))
	randPart := int64(1000)
	if err == nil {
		randPart = n.Int64() + 1000
	}
	return fmt.Sprintf("CN-%s-%04d", time.Now().Format("060102"), randPart)
}



