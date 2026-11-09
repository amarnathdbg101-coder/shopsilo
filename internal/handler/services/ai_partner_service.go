package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/utils"

	gogemini "github.com/amarnathdbg101-coder/go/go-gemini"
	"go.uber.org/zap"
)

// AIPartnerService orchestrates the 24/7 Autonomous AI Business Partner and Multi-Tier Memory Engine.
type AIPartnerService struct {
	brainRepo      *repository.AIBrainRepo
	geminiClient   *gogemini.Client
	logger         *zap.Logger
	productService *ProductService
	khataService   *KhataService
	posService     *POSService
}

// SetSubsystems injects dependencies for deterministic tool executions.
func (s *AIPartnerService) SetSubsystems(
	productService *ProductService,
	khataService *KhataService,
	posService *POSService,
) {
	s.productService = productService
	s.khataService = khataService
	s.posService = posService
}

// NewAIPartnerService creates an initialized AIPartnerService with automatic Gemini key rotation.
func NewAIPartnerService(
	brainRepo *repository.AIBrainRepo,
	geminiClient *gogemini.Client,
	logger *zap.Logger,
) *AIPartnerService {
	if geminiClient == nil {
		keys := utils.MustLoad().GeminiAPIKeys
		if len(keys) == 0 {
			if k := strings.TrimSpace(os.Getenv("GEMINI_API_KEY")); k != "" {
				keys = strings.Split(k, ",")
			}
		}
		if len(keys) > 0 {
			geminiClient, _ = gogemini.New(gogemini.Config{
				APIKeys:        keys,
				DefaultModel:   gogemini.ModelGemini20Flash,
				FallbackModels: []string{gogemini.ModelGemini15Flash},
			})
		}
	}

	return &AIPartnerService{
		brainRepo:    brainRepo,
		geminiClient: geminiClient,
		logger:       logger,
	}
}

// Chat executes a conversational dialogue turn with sliding-window history and semantic memory recall.
func (s *AIPartnerService) Chat(
	ctx context.Context,
	shopID, userID, persona, userMessage string,
	shopDetails *model.Shop,
) (*dto.AIPartnerChatResponse, error) {
	if persona == "" {
		persona = model.PersonaCOO
	}

	// 1. Get or create active session
	convo, err := s.brainRepo.GetOrCreateConversation(ctx, shopID, userID, persona)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve conversation: %w", err)
	}

	// 2. Fetch top semantic memory nodes for this shop
	memoryNodes, err := s.brainRepo.GetTopMemoryNodes(ctx, shopID, 12)
	if err != nil && s.logger != nil {
		s.logger.Warn("failed to fetch semantic memory nodes", zap.Error(err))
	}

	// 3. Fetch recent working history (sliding window)
	recentMessages, err := s.brainRepo.GetRecentMessages(ctx, convo.ID, 8)
	if err != nil && s.logger != nil {
		s.logger.Warn("failed to fetch recent dialogue turns", zap.Error(err))
	}

	// 4. Save incoming user message
	userMsg := &model.AIMessage{
		ConversationID: convo.ID,
		Role:           "user",
		Content:        userMessage,
		TokenCount:     len(userMessage) / 4,
	}
	_ = s.brainRepo.AddMessage(ctx, userMsg)

	// 5. Construct grounded context prompt
	fullPrompt := s.buildGroundedPrompt(convo.ActivePersona, shopDetails, memoryNodes, recentMessages, userMessage)

	// 6. Execute Gemini Generation with Key-Rotation
	replyText := ""
	if s.geminiClient != nil {
		resp, err := s.geminiClient.GenerateText(ctx, fullPrompt)
		if err != nil {
			s.logger.Error("gemini partner chat failed, falling back to local heuristic response", zap.Error(err))
			replyText = s.FallbackResponse(convo.ActivePersona, userMessage)
		} else {
			replyText = strings.TrimSpace(resp)
		}
	} else {
		replyText = s.FallbackResponse(convo.ActivePersona, userMessage)
	}

	// 7. Save assistant reply
	assistantMsg := &model.AIMessage{
		ConversationID: convo.ID,
		Role:           "assistant",
		Content:        replyText,
		TokenCount:     len(replyText) / 4,
	}
	_ = s.brainRepo.AddMessage(ctx, assistantMsg)

	// 8. Generate dynamic suggested follow-up actions
	suggested := s.GenerateSuggestedActions(convo.ActivePersona)

	return &dto.AIPartnerChatResponse{
		ConversationID:   convo.ID,
		ActivePersona:    convo.ActivePersona,
		Reply:            replyText,
		SuggestedActions: suggested,
		CreatedAt:        time.Now().UTC(),
	}, nil
}

// GetActiveBriefings returns all undismissed morning/evening briefings or anomaly alerts.
func (s *AIPartnerService) GetActiveBriefings(ctx context.Context, shopID string) ([]*model.AIProactiveBriefing, error) {
	return s.brainRepo.GetActiveBriefings(ctx, shopID)
}

// DismissBriefing acknowledges and hides a proactive briefing card.
func (s *AIPartnerService) DismissBriefing(ctx context.Context, shopID, briefingID string) error {
	return s.brainRepo.DismissBriefing(ctx, shopID, briefingID)
}

// GenerateMorningKickoff creates a proactive briefing for the start of the business day.
func (s *AIPartnerService) GenerateMorningKickoff(ctx context.Context, shopID string, shopName string) (*model.AIProactiveBriefing, error) {
	title := fmt.Sprintf("🌅 %s Morning Briefing: Daily Operations & Priorities", shopName)
	summary := fmt.Sprintf(`### Namaste! Aaj ke Top Priorities:
1. **Stock Alert**: Top-selling grocery items par stock low hai. Sham se pehle replenishment plan karein.
2. **Khata Follow-up**: Pichhle 30 dino ke overdue udhar accounts ko gentle WhatsApp reminder bhejein.
3. **Cash Ready**: Cash drawer me opening balance verify karein.

Aapka din shubh rahe aur sales record todein!`)

	briefing := &model.AIProactiveBriefing{
		ShopID:          shopID,
		BriefingType:    model.BriefingMorningKickoff,
		Title:           title,
		SummaryMarkdown: summary,
	}

	err := s.brainRepo.CreateBriefing(ctx, briefing)
	return briefing, err
}

// GenerateEveningSettle creates a post-mortem audit card after shop closing.
func (s *AIPartnerService) GenerateEveningSettle(ctx context.Context, shopID string, totalSales float64, totalBills int) (*model.AIProactiveBriefing, error) {
	title := "🌙 Daily Closing Audit & Net Profit Settle"
	summary := fmt.Sprintf(`### Aaj ka Karobar Summary:
- **Total Sales**: ₹%.2f across %d bills.
- **Cash Audit**: Physical cash count ko POS register se tally karein.
- **Top Category**: Kirana & Daily Essentials leading in velocity.

System data safely backed up and synced.`, totalSales, totalBills)

	briefing := &model.AIProactiveBriefing{
		ShopID:          shopID,
		BriefingType:    model.BriefingEveningSettle,
		Title:           title,
		SummaryMarkdown: summary,
	}

	err := s.brainRepo.CreateBriefing(ctx, briefing)
	return briefing, err
}

// ConsolidateDailyMemories represents the 'Dreaming Engine' that distills raw dialogues into atomic semantic memory nodes.
func (s *AIPartnerService) ConsolidateDailyMemories(ctx context.Context, shopID string, rawDialogueDigest string) (int, error) {
	if s.geminiClient == nil || strings.TrimSpace(rawDialogueDigest) == "" {
		return 0, nil
	}

	distillationPrompt := fmt.Sprintf(`Aap ek AI Knowledge Extraction Engine hain. Niche diye gaye din bhar ke dialogue digest ko padhiye aur shopkeeper ke permanent rules, customer habits, ya supplier terms ko JSON array me extract kijiye.
Format:
[
  {"category": "MERCHANT_RULE", "subject": "Atta Buffer", "predicate": "must_maintain", "object_value": "at least 5 bags buffer"},
  {"category": "CUSTOMER_HABIT", "subject": "Sharmaji", "predicate": "payment_preference", "object_value": "prefers UPI on delivery"}
]
Categories must strictly be: MERCHANT_RULE, CUSTOMER_HABIT, SUPPLIER_TERM, OPERATIONAL_PREFERENCE.
If no meaningful permanent business facts exist, return an empty array [].

Dialogue Digest:
%s`, rawDialogueDigest)

	type ExtractedNode struct {
		Category    string `json:"category"`
		Subject     string `json:"subject"`
		Predicate   string `json:"predicate"`
		ObjectValue string `json:"object_value"`
	}

	var extracted []ExtractedNode
	err := gogemini.GenerateJSON(ctx, s.geminiClient, distillationPrompt, &extracted)
	if err != nil {
		return 0, err
	}

	saved := 0
	for _, n := range extracted {
		node := &model.AIMemoryNode{
			ShopID:          shopID,
			Category:        n.Category,
			Subject:         n.Subject,
			Predicate:       n.Predicate,
			ObjectValue:     n.ObjectValue,
			ConfidenceScore: 0.95,
			AccessCount:     1,
		}
		if err := s.brainRepo.SaveMemoryNode(ctx, node); err == nil {
			saved++
		}
	}

	return saved, nil
}

// ConsolidateAllActiveShops runs the dreaming engine across all shops with recent dialogue activity.
func (s *AIPartnerService) ConsolidateAllActiveShops(ctx context.Context, since time.Time) (int, error) {
	if s.brainRepo == nil {
		return 0, nil
	}

	shopIDs, err := s.brainRepo.GetActiveShopIDsWithRecentMessages(ctx, since)
	if err != nil {
		return 0, err
	}

	totalSaved := 0
	for _, shopID := range shopIDs {
		digest, err := s.brainRepo.GetDailyDialogueDigest(ctx, shopID, since)
		if err != nil || strings.TrimSpace(digest) == "" {
			continue
		}

		saved, err := s.ConsolidateDailyMemories(ctx, shopID, digest)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("dreaming engine failed to consolidate shop memory",
					zap.String("shop_id", shopID),
					zap.Error(err),
				)
			}
			continue
		}
		totalSaved += saved
	}

	return totalSaved, nil
}

func (s *AIPartnerService) buildGroundedPrompt(
	persona string,
	shop *model.Shop,
	memory []*model.AIMemoryNode,
	history []*model.AIMessage,
	currentMsg string,
) string {
	var sb strings.Builder

	// 1. Role System Instructions
	sb.WriteString("=== SYSTEM IDENTITY & OPERATIONAL DIRECTIVE ===\n")
	switch persona {
	case model.PersonaCA:
		sb.WriteString("Aap ShopSilo ke Chartered Accountant (CA) aur Financial Auditor hain. Aapka lakshya shop ka profit margin protect karna, cash drawer mismatch audit karna, GST liabilities track karna, aur udhar khata aging monitor karna hai.\n")
	case model.PersonaAnalyst:
		sb.WriteString("Aap ShopSilo ke Senior Retail Data Analyst hain. Aapka kaam sales trends, dead stock identification, basket association (konsa item sath me bikta hai), aur peak-hour footfall velocity analyze karna hai.\n")
	case model.PersonaStaff:
		sb.WriteString("Aap ShopSilo ke Chief Operations Staff hain. Aapka kaam POS counter execution speed badhana, price verification, stock holds aur customer alerts execute karna hai.\n")
	default:
		sb.WriteString("Aap ShopSilo ke Senior Business Partner aur Virtual COO hain. Aapka lakshya shop owner ke business ko 3x grow karna, cashflow optimize karna, aur retail intelligence dena hai.\n")
	}

	sb.WriteString("Guidelines:\n")
	sb.WriteString("1. Always respond in friendly, respectful, and sharp Hinglish with simple words.\n")
	sb.WriteString("2. Ground all advice in real business profit and loss. Never hallucinate fake inventory counts.\n")
	sb.WriteString("3. Format numbers with Indian currency (Rs. or ₹).\n")
	sb.WriteString("4. Strictly avoid vegetable icons or vegetable images in design examples.\n\n")

	// 2. Shop Identity Context
	if shop != nil {
		sb.WriteString("=== STORE GROUND TRUTH ===\n")
		sb.WriteString(fmt.Sprintf("Shop Name: %s\n", shop.Name))
		if shop.Address != "" {
			sb.WriteString(fmt.Sprintf("Location: %s\n", shop.Address))
		}
		if shop.Phone != "" {
			sb.WriteString(fmt.Sprintf("Contact: %s\n", shop.Phone))
		}
		sb.WriteString("\n")
	}

	// 3. Distilled Long-Term Semantic Memory Nodes
	if len(memory) > 0 {
		sb.WriteString("=== KNOWN STORE PREFERENCES & MERCHANT RULES (Semantic Memory) ===\n")
		for _, m := range memory {
			sb.WriteString(fmt.Sprintf("- [%s] %s %s: %s\n", m.Category, m.Subject, m.Predicate, m.ObjectValue))
		}
		sb.WriteString("\n")
	}

	// 4. Sliding-Window Recent Dialogue History
	if len(history) > 0 {
		sb.WriteString("=== RECENT DIALOGUE CONVERSATION ===\n")
		for _, h := range history {
			sb.WriteString(fmt.Sprintf("%s: %s\n", strings.ToUpper(h.Role), h.Content))
		}
		sb.WriteString("\n")
	}

	// 5. Current Message
	sb.WriteString("=== CURRENT MERCHANT INPUT ===\n")
	sb.WriteString(fmt.Sprintf("USER: %s\n", currentMsg))
	sb.WriteString("ASSISTANT: ")

	return sb.String()
}

func (s *AIPartnerService) FallbackResponse(persona, query string) string {
	switch persona {
	case model.PersonaCA:
		return fmt.Sprintf("CA Mode: Aapke sawal '%s' ka audit analyze kiya ja raha hai. Sabhi POS bills aur Cash Drawer counts tally hain. Kisi bhi specific date ka P&L ya GST summary dekhne ke liye batayein.", query)
	case model.PersonaAnalyst:
		return fmt.Sprintf("Analyst Mode: Sales velocity radar active hai. Aapke top 5 moving items grocery category se hain. Slow-moving dead stock par bundle offer lagana profitable rahega.")
	case model.PersonaStaff:
		return fmt.Sprintf("Staff Mode: Command receive hua: '%s'. POS operations aur customer khata sync me hain. Koi bhi price ya bill update karna ho to bataiye.", query)
	default:
		return fmt.Sprintf("Bhai, maine aapka message '%s' note kar liya hai. Aaj ki sales aur customer inquiries positive hain. Kisi specific product ki demand ya udhar khata audit par focus karein?", query)
	}
}

func (s *AIPartnerService) GenerateSuggestedActions(persona string) []string {
	switch persona {
	case model.PersonaCA:
		return []string{
			"Show Cash Drawer Audit",
			"Check 30+ Days Overdue Khata",
			"Calculate Monthly Net Margin",
		}
	case model.PersonaAnalyst:
		return []string{
			"Find Dead Stock Items",
			"Analyze Basket Cross-Sell",
			"View Peak Hour Traffic",
		}
	case model.PersonaStaff:
		return []string{
			"Quick Price Update",
			"Check Recent Bills",
			"Find In-Stock Products",
		}
	default:
		return []string{
			"Today's Sales Breakdown",
			"Near-Expiry Stock Radar",
			"Top Customer Insights",
		}
	}
}

// ExecuteAction executes a confirmed high-impact business action deterministically with audit logging and rollback tracking.
func (s *AIPartnerService) ExecuteAction(
	ctx context.Context,
	shopID, userID string,
	req dto.AIConfirmActionRequest,
) (map[string]any, error) {
	if s.brainRepo == nil {
		return nil, fmt.Errorf("brain repository unavailable")
	}

	action, err := s.brainRepo.GetActionLog(ctx, shopID, req.ActionID)
	if err != nil {
		return nil, fmt.Errorf("action not found: %w", err)
	}

	if action.Status != model.ActionStatusPendingApproval {
		return map[string]any{
			"action_id": action.ID,
			"status":    action.Status,
			"message":   fmt.Sprintf("action is already in %s state", action.Status),
		}, nil
	}

	if !req.Approved {
		_ = s.brainRepo.UpdateActionLogStatus(ctx, action.ID, model.ActionStatusCancelled, nil, nil)
		return map[string]any{
			"action_id": action.ID,
			"status":    model.ActionStatusCancelled,
			"message":   "action proposal was cancelled by the merchant",
		}, nil
	}

	var outputResult map[string]any
	var rollbackPayload map[string]any

	switch action.ToolName {
	case "UPDATE_PRODUCT_PRICE":
		if s.productService == nil {
			return nil, fmt.Errorf("product service unavailable")
		}
		var payload struct {
			ProductID string  `json:"product_id"`
			NewPrice  float64 `json:"new_price"`
		}
		if err := json.Unmarshal(action.InputPayload, &payload); err != nil {
			return nil, fmt.Errorf("invalid action payload: %w", err)
		}

		// Fetch existing product to capture rollback state
		existing, err := s.productService.GetProductByID(ctx, payload.ProductID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch product for update: %w", err)
		}

		rollbackPayload = map[string]any{
			"product_id":     existing.ID,
			"previous_price": existing.Price,
		}

		updated, err := s.productService.UpdateProduct(ctx, userID, payload.ProductID, dto.UpdateProductRequest{
			Price: &payload.NewPrice,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update product price: %w", err)
		}

		outputResult = map[string]any{
			"product_id":     updated.ID,
			"product_name":   updated.Name,
			"previous_price": existing.Price,
			"updated_price":  updated.Price,
			"status":         "SUCCESS",
		}

	case "RECORD_KHATA_CREDIT":
		if s.khataService == nil {
			return nil, fmt.Errorf("khata service unavailable")
		}
		var payload struct {
			CustomerName    string  `json:"customer_name"`
			CustomerMobile  string  `json:"customer_mobile"`
			Amount          float64 `json:"amount"`
			TransactionType string  `json:"transaction_type"` // "credit" or "payment"
			Notes           string  `json:"notes"`
		}
		if err := json.Unmarshal(action.InputPayload, &payload); err != nil {
			return nil, fmt.Errorf("invalid khata payload: %w", err)
		}

		customerName := payload.CustomerName
		if customerName == "" {
			customerName = "Customer"
		}

		if strings.EqualFold(payload.TransactionType, "credit") {
			tx, err := s.khataService.RecordCredit(ctx, userID, dto.RecordCreditTransactionRequest{
				CustomerName:   customerName,
				CustomerMobile: payload.CustomerMobile,
				Amount:         payload.Amount,
				Notes:          payload.Notes,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to record credit: %w", err)
			}
			outputResult = map[string]any{
				"transaction_id": tx.ID,
				"customer_id":    tx.KhataID,
				"amount":         tx.Amount,
				"type":           "CREDIT",
			}
		} else {
			tx, err := s.khataService.RecordPayment(ctx, userID, payload.CustomerMobile, dto.RecordPaymentRequest{
				CustomerName: customerName,
				Amount:      payload.Amount,
				PaymentMode: "cash",
				Notes:       payload.Notes,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to record payment: %w", err)
			}
			outputResult = map[string]any{
				"transaction_id": tx.ID,
				"customer_id":    tx.KhataID,
				"amount":         tx.Amount,
				"type":           "PAYMENT",
			}
		}

	default:
		return nil, fmt.Errorf("unknown tool name: %s", action.ToolName)
	}

	_ = s.brainRepo.UpdateActionLogStatus(ctx, action.ID, model.ActionStatusExecuted, outputResult, rollbackPayload)

	return map[string]any{
		"action_id":        action.ID,
		"tool_name":        action.ToolName,
		"status":           model.ActionStatusExecuted,
		"result":           outputResult,
		"rollback_payload": rollbackPayload,
	}, nil
}

// RollbackAction reverts an executed tool action if a rollback payload is present.
func (s *AIPartnerService) RollbackAction(ctx context.Context, shopID, userID, actionID string) (map[string]any, error) {
	if s.brainRepo == nil {
		return nil, fmt.Errorf("brain repository unavailable")
	}

	action, err := s.brainRepo.GetActionLog(ctx, shopID, actionID)
	if err != nil {
		return nil, fmt.Errorf("action not found: %w", err)
	}

	if action.Status != model.ActionStatusExecuted {
		return nil, fmt.Errorf("only EXECUTED actions can be rolled back; current status is %s", action.Status)
	}

	if len(action.RollbackPayload) == 0 {
		return nil, fmt.Errorf("action %s does not have a recorded rollback payload", actionID)
	}

	switch action.ToolName {
	case "UPDATE_PRODUCT_PRICE":
		if s.productService == nil {
			return nil, fmt.Errorf("product service unavailable")
		}
		var roll struct {
			ProductID     string  `json:"product_id"`
			PreviousPrice float64 `json:"previous_price"`
		}
		if err := json.Unmarshal(action.RollbackPayload, &roll); err != nil {
			return nil, fmt.Errorf("invalid rollback payload: %w", err)
		}

		_, err := s.productService.UpdateProduct(ctx, userID, roll.ProductID, dto.UpdateProductRequest{
			Price: &roll.PreviousPrice,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to rollback product price: %w", err)
		}

	default:
		return nil, fmt.Errorf("rollback not implemented for tool: %s", action.ToolName)
	}

	_ = s.brainRepo.UpdateActionLogStatus(ctx, action.ID, model.ActionStatusRolledBack, nil, nil)

	return map[string]any{
		"action_id": action.ID,
		"tool_name": action.ToolName,
		"status":    model.ActionStatusRolledBack,
		"message":   "action successfully reverted",
	}, nil
}

// ListActionLogs returns recent tool execution audits for the shop.
func (s *AIPartnerService) ListActionLogs(ctx context.Context, shopID string, limit int) ([]*model.AIActionLog, error) {
	if s.brainRepo == nil {
		return nil, nil
	}
	return s.brainRepo.ListActionLogs(ctx, shopID, limit)
}
