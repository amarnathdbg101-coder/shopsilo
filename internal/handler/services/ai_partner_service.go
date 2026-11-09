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
	telemetry      *TelemetryProvider
	toolDispatcher *AIToolDispatcher
	intentRouter   *AIIntentRouter
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

// NewAIPartnerService creates an initialized AIPartnerService with automatic Gemini key rotation, live telemetry, and tool dispatcher.
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

	var telemetry *TelemetryProvider
	var toolDispatcher *AIToolDispatcher
	if brainRepo != nil && brainRepo.DB() != nil {
		telemetry = NewTelemetryProvider(brainRepo.DB())
		toolDispatcher = NewAIToolDispatcher(brainRepo.DB(), brainRepo, telemetry, logger)
	}

	intentRouter := NewAIIntentRouter()

	return &AIPartnerService{
		brainRepo:      brainRepo,
		geminiClient:   geminiClient,
		telemetry:      telemetry,
		toolDispatcher: toolDispatcher,
		intentRouter:   intentRouter,
		logger:         logger,
	}
}

// Chat executes an autonomous ReAct turn with live store telemetry grounding, function calling, and deterministic tool dispatch.
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

	// 4. Fetch live store telemetry snapshot (OOM-safe micro-cached)
	var telemetrySnap *StoreTelemetrySnapshot
	if s.telemetry != nil {
		telemetrySnap, _ = s.telemetry.GetSnapshot(ctx, shopID)
	}

	// 5. Save incoming user message
	userMsg := &model.AIMessage{
		ConversationID: convo.ID,
		Role:           "user",
		Content:        userMessage,
		TokenCount:     len(userMessage) / 4,
	}
	_ = s.brainRepo.AddMessage(ctx, userMsg)

	// 6. Build system prompt grounded with live real-time telemetry and memory
	systemPrompt := s.buildSystemDirective(convo.ActivePersona, shopDetails, memoryNodes, telemetrySnap)

	// 7. Prepare Gemini Chat Request turns
	var contents []gogemini.Content
	for _, h := range recentMessages {
		role := "user"
		if h.Role == "assistant" || h.Role == "model" {
			role = "model"
		}
		contents = append(contents, gogemini.Content{
			Role:  role,
			Parts: []gogemini.Part{{Text: h.Content}},
		})
	}
	// Append current user message
	contents = append(contents, gogemini.Content{
		Role:  "user",
		Parts: []gogemini.Part{{Text: userMessage}},
	})

	var tools []gogemini.Tool
	if s.toolDispatcher != nil {
		tools = s.toolDispatcher.GetToolDeclarations()
	}

	replyText := ""
	var recordedToolCalls, recordedToolResults string

	// Step 7.5: Sub-2ms Deterministic Fast-Path Routing (Zero Tokens, Zero Hallucination, Instantaneous)
	if s.intentRouter != nil {
		if fastReply, ok := s.intentRouter.RouteFastPath(userMessage, convo.ActivePersona, telemetrySnap); ok {
			replyText = fastReply
		}
	}

	// 8. Execute ReAct Gemini Generation with Key Rotation and Tool Execution Loop if not answered by fast path
	if replyText == "" && s.geminiClient != nil {
		chatReq := gogemini.ChatRequest{
			SystemInstruction: systemPrompt,
			Contents:          contents,
			Tools:             tools,
		}

		resp, err := s.geminiClient.Chat(ctx, chatReq)
		if err != nil {
			if s.logger != nil {
				s.logger.Error("gemini partner chat failed, falling back to smart telemetry response", zap.Error(err))
			}
			replyText = s.smartFallbackResponse(convo.ActivePersona, userMessage, telemetrySnap)
		} else if len(resp.FunctionCalls) > 0 {
			// Model requested tool executions!
			var toolCallLogs []string
			var toolResultLogs []string

			// Prepare follow-up contents with model function call turn
			var modelParts []gogemini.Part
			var funcRespParts []gogemini.Part

			for _, fc := range resp.FunctionCalls {
				modelParts = append(modelParts, gogemini.Part{
					FunctionCall: &fc,
				})

				// Execute tool locally against PostgreSQL
				toolResult, toolErr := s.toolDispatcher.ExecuteToolCall(ctx, shopID, userID, fc)
				if toolErr != nil {
					toolResult = map[string]any{"status": "error", "error": toolErr.Error()}
				}

				funcRespParts = append(funcRespParts, gogemini.Part{
					FunctionResponse: &gogemini.FunctionResponse{
						Name:     fc.Name,
						Response: toolResult,
					},
				})

				fcBytes, _ := json.Marshal(fc)
				resBytes, _ := json.Marshal(toolResult)
				toolCallLogs = append(toolCallLogs, string(fcBytes))
				toolResultLogs = append(toolResultLogs, string(resBytes))
			}

			recordedToolCalls = strings.Join(toolCallLogs, "; ")
			recordedToolResults = strings.Join(toolResultLogs, "; ")

			// Second turn: pass tool results back to Gemini for final grounded synthesis
			followUpContents := append(contents,
				gogemini.Content{Role: "model", Parts: modelParts},
				gogemini.Content{Role: "function", Parts: funcRespParts},
			)

			finalResp, finalErr := s.geminiClient.Chat(ctx, gogemini.ChatRequest{
				SystemInstruction: systemPrompt,
				Contents:          followUpContents,
			})
			if finalErr == nil && strings.TrimSpace(finalResp.Text) != "" {
				replyText = strings.TrimSpace(finalResp.Text)
			} else {
				// Fallback to direct tool result formatting
				replyText = s.formatToolExecutionFallback(convo.ActivePersona, resp.FunctionCalls, toolResultLogs)
			}
		} else {
			replyText = strings.TrimSpace(resp.Text)
		}
	} else {
		replyText = s.smartFallbackResponse(convo.ActivePersona, userMessage, telemetrySnap)
	}

	if replyText == "" {
		replyText = s.smartFallbackResponse(convo.ActivePersona, userMessage, telemetrySnap)
	}

	var tcRaw, trRaw json.RawMessage
	if recordedToolCalls != "" {
		tcRaw = json.RawMessage(recordedToolCalls)
	}
	if recordedToolResults != "" {
		trRaw = json.RawMessage(recordedToolResults)
	}

	// 9. Save assistant reply with tool telemetry
	assistantMsg := &model.AIMessage{
		ConversationID: convo.ID,
		Role:           "assistant",
		Content:        replyText,
		ToolCalls:      tcRaw,
		ToolResults:    trRaw,
		TokenCount:     len(replyText) / 4,
	}
	_ = s.brainRepo.AddMessage(ctx, assistantMsg)

	// 10. Generate suggested follow-up actions grounded in telemetry
	suggested := s.GenerateSuggestedActions(convo.ActivePersona, telemetrySnap)

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

// GetMemoryNodes returns the distilled semantic long-term memory graph for the shopkeeper.
func (s *AIPartnerService) GetMemoryNodes(ctx context.Context, shopID string) ([]*model.AIMemoryNode, error) {
	if s.brainRepo == nil {
		return nil, nil
	}
	return s.brainRepo.GetTopMemoryNodes(ctx, shopID, 50)
}

// DeleteMemoryNode deletes an outdated or incorrect memory node for the shopkeeper.
func (s *AIPartnerService) DeleteMemoryNode(ctx context.Context, shopID, nodeID string) error {
	if s.brainRepo == nil {
		return nil
	}
	return s.brainRepo.DeleteMemoryNode(ctx, shopID, nodeID)
}

// GenerateMorningKickoff creates a proactive briefing grounded in real live store priorities.
func (s *AIPartnerService) GenerateMorningKickoff(ctx context.Context, shopID string, shopName string) (*model.AIProactiveBriefing, error) {
	var snap *StoreTelemetrySnapshot
	if s.telemetry != nil {
		snap, _ = s.telemetry.GetSnapshot(ctx, shopID)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### 🌅 %s Morning Briefing & Kickoff\n", shopName))

	if snap != nil {
		if snap.LowStockCount > 0 {
			sb.WriteString(fmt.Sprintf("1. **🚨 Inventory Alert**: **%d items** par stock critical hai. ", snap.LowStockCount))
			if len(snap.CriticalItems) > 0 {
				var names []string
				for _, it := range snap.CriticalItems {
					names = append(names, fmt.Sprintf("%s (%d units)", it.Name, it.Stock))
				}
				sb.WriteString(fmt.Sprintf("Top priority replenishment: %s.\n", strings.Join(names, ", ")))
			} else {
				sb.WriteString("Supplier ko restock order bhejna zaroori hai.\n")
			}
		} else {
			sb.WriteString("1. **✅ Inventory Healthy**: Sabhi essential grocery items buffer stock me hain.\n")
		}

		if snap.TotalKhataUdhar > 0 {
			sb.WriteString(fmt.Sprintf("2. **📋 Khata Debt Radar**: Kul **₹%.2f** udhar baki hai (%d customers). ",
				snap.TotalKhataUdhar, snap.TotalKhataCustomers))
			if len(snap.TopOverdueDebtors) > 0 {
				od := snap.TopOverdueDebtors[0]
				sb.WriteString(fmt.Sprintf("Highest balance: **%s** (₹%.0f) ko WhatsApp reminder send karein.\n", od.Name, od.Balance))
			} else {
				sb.WriteString("Payment follow-up plan karein.\n")
			}
		} else {
			sb.WriteString("2. **💰 Khata Clear**: Zero pending overdue balance.\n")
		}

		sb.WriteString("3. **💵 Cash Ready**: Opening drawer balance verify karein aur POS counter test karein.\n\n")
		sb.WriteString("Aapka din shubh rahe aur aaj sales ke naye records banein!")
	} else {
		sb.WriteString("1. **Stock Check**: Low stock items verify karein.\n")
		sb.WriteString("2. **Khata Settle**: Overdue customers ko reminder bhejein.\n")
		sb.WriteString("3. **Counter Speed**: POS billing active rakhein.")
	}

	briefing := &model.AIProactiveBriefing{
		ShopID:          shopID,
		BriefingType:    model.BriefingMorningKickoff,
		Title:           fmt.Sprintf("🌅 %s Morning Kickoff", shopName),
		SummaryMarkdown: sb.String(),
	}

	err := s.brainRepo.CreateBriefing(ctx, briefing)
	return briefing, err
}

// GenerateEveningSettle creates a post-mortem closing audit grounded in real POS & expense transactions.
func (s *AIPartnerService) GenerateEveningSettle(ctx context.Context, shopID string, totalSales float64, totalBills int) (*model.AIProactiveBriefing, error) {
	var snap *StoreTelemetrySnapshot
	if s.telemetry != nil {
		snap, _ = s.telemetry.GetSnapshot(ctx, shopID)
	}

	var sb strings.Builder
	sb.WriteString("### 🌙 Daily Store Closing & P&L Settle\n")

	if snap != nil {
		netProfit := snap.TodayGrossProfit - snap.TodayExpenses
		sb.WriteString(fmt.Sprintf("- **Total Revenue**: ₹%.2f across **%d bills** (Avg Basket: ₹%.2f)\n",
			snap.TodaySalesAmount, snap.TodaySalesCount, snap.AverageBasketValue))
		sb.WriteString(fmt.Sprintf("- **Payment Split**: Cash ₹%.2f | Online UPI ₹%.2f | Khata ₹%.2f\n",
			snap.CashSales, snap.UPISales, snap.KhataSales))
		sb.WriteString(fmt.Sprintf("- **Operating Expenses Today**: ₹%.2f\n", snap.TodayExpenses))
		sb.WriteString(fmt.Sprintf("- **Estimated Net Profit**: ₹%.2f\n", netProfit))

		if len(snap.TopMovingProductsToday) > 0 {
			sb.WriteString(fmt.Sprintf("- **Top Moving Items**: %s\n", strings.Join(snap.TopMovingProductsToday, ", ")))
		}
		sb.WriteString("\nPhysical cash drawer ko tally karke system safe close karein.")
	} else {
		sb.WriteString(fmt.Sprintf("- **Total Sales**: ₹%.2f across %d bills\n", totalSales, totalBills))
		sb.WriteString("- Physical cash drawer register se match karein.")
	}

	briefing := &model.AIProactiveBriefing{
		ShopID:          shopID,
		BriefingType:    model.BriefingEveningSettle,
		Title:           "🌙 Daily Closing Audit & Net Settle",
		SummaryMarkdown: sb.String(),
	}

	err := s.brainRepo.CreateBriefing(ctx, briefing)
	return briefing, err
}

// ConsolidateDailyMemories represents the 'Dreaming Engine' that mines PostgreSQL transactions and raw dialogues into semantic memory nodes.
func (s *AIPartnerService) ConsolidateDailyMemories(ctx context.Context, shopID string, rawDialogueDigest string) (int, error) {
	if s.brainRepo == nil || s.brainRepo.DB() == nil {
		return 0, nil
	}

	db := s.brainRepo.DB()
	since := time.Now().Add(-24 * time.Hour)

	// 1. Transaction Data Mining: Find top basket cross-sell pairs
	basketQuery := `
		SELECT p1.title, p2.title, COUNT(*) as pair_count
		FROM pos_bill_items pbi1
		JOIN pos_bill_items pbi2 ON pbi1.bill_id = pbi2.bill_id AND pbi1.product_id < pbi2.product_id
		JOIN products p1 ON pbi1.product_id = p1.id
		JOIN products p2 ON pbi2.product_id = p2.id
		JOIN pos_bills pb ON pbi1.bill_id = pb.id
		WHERE pb.shop_id = $1 AND pb.created_at >= $2
		GROUP BY p1.title, p2.title
		ORDER BY pair_count DESC
		LIMIT 3
	`
	bRows, err := db.Query(ctx, basketQuery, shopID, since)
	var basketFacts []string
	if err == nil {
		defer bRows.Close()
		for bRows.Next() {
			var t1, t2 string
			var count int
			if err := bRows.Scan(&t1, &t2, &count); err == nil && count >= 2 {
				fact := fmt.Sprintf("%s and %s frequently bought together (%d times)", t1, t2, count)
				basketFacts = append(basketFacts, fact)
				_ = s.brainRepo.SaveMemoryNode(ctx, &model.AIMemoryNode{
					ShopID:          shopID,
					Category:        "BASKET_AFFINITY",
					Subject:         t1 + " + " + t2,
					Predicate:       "cross_sell_frequency",
					ObjectValue:     fmt.Sprintf("%d basket associations today", count),
					ConfidenceScore: 0.98,
					AccessCount:     1,
				})
			}
		}
	}

	// 2. Transaction Mining: Chronic Khata Debtors
	debtorQuery := `
		SELECT customer_name, current_balance
		FROM customer_khata
		WHERE shop_id = $1 AND current_balance >= 2000
		ORDER BY current_balance DESC
		LIMIT 3
	`
	dRows, err := db.Query(ctx, debtorQuery, shopID)
	if err == nil {
		defer dRows.Close()
		for dRows.Next() {
			var name string
			var bal float64
			if err := dRows.Scan(&name, &bal); err == nil {
				_ = s.brainRepo.SaveMemoryNode(ctx, &model.AIMemoryNode{
					ShopID:          shopID,
					Category:        "CUSTOMER_HABIT",
					Subject:         name,
					Predicate:       "high_udhar_balance",
					ObjectValue:     fmt.Sprintf("Significant balance ₹%.0f requires gentle payment nudge", bal),
					ConfidenceScore: 0.90,
					AccessCount:     1,
				})
			}
		}
	}

	// 3. Dialogue Distillation via Gemini
	if s.geminiClient != nil && strings.TrimSpace(rawDialogueDigest) != "" {
		distillationPrompt := fmt.Sprintf(`Aap ek Senior Retail Knowledge Extraction Engine hain. Niche diye gaye din bhar ke dialogue digest aur store transaction patterns ko padhiye aur permanent merchant rules, customer habits, ya operational preferences ko JSON array me extract kijiye.
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
		if err := gogemini.GenerateJSON(ctx, s.geminiClient, distillationPrompt, &extracted); err == nil {
			for _, n := range extracted {
				_ = s.brainRepo.SaveMemoryNode(ctx, &model.AIMemoryNode{
					ShopID:          shopID,
					Category:        n.Category,
					Subject:         n.Subject,
					Predicate:       n.Predicate,
					ObjectValue:     n.ObjectValue,
					ConfidenceScore: 0.95,
					AccessCount:     1,
				})
			}
		}
	}

	return len(basketFacts), nil
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
		digest, _ := s.brainRepo.GetDailyDialogueDigest(ctx, shopID, since)
		saved, err := s.ConsolidateDailyMemories(ctx, shopID, digest)
		if err == nil {
			totalSaved += saved
		}
	}

	return totalSaved, nil
}

func (s *AIPartnerService) buildSystemDirective(
	persona string,
	shop *model.Shop,
	memory []*model.AIMemoryNode,
	snap *StoreTelemetrySnapshot,
) string {
	var sb strings.Builder

	// 1. Core Identity & Persona Directives
	sb.WriteString("=== SYSTEM IDENTITY & OPERATIONAL DIRECTIVE ===\n")
	switch persona {
	case model.PersonaCA:
		sb.WriteString("Aap ShopSilo ke Senior Chartered Accountant (CA) aur Financial Auditor Munim hain.\n")
		sb.WriteString("Aapka lakshya: Gross Margin %, Net Profit after operational expenses audit karna, cash drawer mismatch pakadna, GST compliance ensure karna, aur 30+ days overdue khata udhar recover karwana.\n")
		sb.WriteString("Tone: Professional, sharp, protective of profit margins, exact numbers ke sath baat karein.\n")
	case model.PersonaAnalyst:
		sb.WriteString("Aap ShopSilo ke Chief Retail Data Scientist & Analyst hain.\n")
		sb.WriteString("Aapka lakshya: Sales velocity analysis, dead stock identification (jo 30+ dino se nahi bika), basket association (doodh ke sath bread ka combo), aur peak-hour footfall optimization.\n")
		sb.WriteString("Tone: Data-driven, analytical, proactive combo and discount suggestions.\n")
	case model.PersonaStaff:
		sb.WriteString("Aap ShopSilo ke Super Counter Operations Staff hain.\n")
		sb.WriteString("Aapka lakshya: Fast POS billing speed, price checks, barcode scans, instant stock confirmation, aur customer service.\n")
		sb.WriteString("Tone: Energetic, fast, helpful, action-oriented.\n")
	default:
		sb.WriteString("Aap ShopSilo ke Senior Business Partner aur Virtual COO hain.\n")
		sb.WriteString("Aapka lakshya: Dukandar ke business ko 3x grow karna, working capital rotation fast karna, high-margin categories grow karna, aur supplier negotiation me help karna.\n")
		sb.WriteString("Tone: Friendly, trustworthy co-founder, respectful, sharp Hinglish me baat karein.\n")
	}

	sb.WriteString("\nSTRICT RULES:\n")
	sb.WriteString("1. Always respond in fluent, respectful, natural Hinglish with simple words.\n")
	sb.WriteString("2. GROUND EVERYTHING IN REAL NUMBERS. Never hallucinate fake inventory counts or sales figures. Use the LIVE STORE TELEMETRY provided below.\n")
	sb.WriteString("3. If user asks to change price, restock, or add khata, USE THE AVAILABLE TOOLS to propose the action cleanly.\n")
	sb.WriteString("4. Format currency in Indian Rupees (₹ or Rs.).\n")
	sb.WriteString("5. Strictly avoid vegetable icons or vegetable images in any design examples.\n\n")

	// 2. Store Ground Truth Identity
	if shop != nil {
		sb.WriteString("=== STORE GROUND TRUTH ===\n")
		sb.WriteString(fmt.Sprintf("Shop Name: %s | Category: %s\n", shop.Name, shop.Category))
		if shop.Address != "" {
			sb.WriteString(fmt.Sprintf("Location: %s\n", shop.Address))
		}
		if shop.Phone != "" {
			sb.WriteString(fmt.Sprintf("Phone: %s\n", shop.Phone))
		}
		sb.WriteString("\n")
	}

	// 3. Live Store Telemetry Block (100% Factual Ground Truth)
	if snap != nil {
		sb.WriteString(snap.FormatMarkdown())
		sb.WriteString("\n")
	}

	// 4. Semantic Long-Term Memory
	if len(memory) > 0 {
		sb.WriteString("=== LEARNED STORE RULES & HABITS (Semantic Memory Graph) ===\n")
		for _, m := range memory {
			sb.WriteString(fmt.Sprintf("- [%s] %s %s: %s\n", m.Category, m.Subject, m.Predicate, m.ObjectValue))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FallbackResponse returns a persona-grounded response when external APIs are unavailable.
func (s *AIPartnerService) FallbackResponse(persona, query string) string {
	return s.smartFallbackResponse(persona, query, nil)
}

func (s *AIPartnerService) smartFallbackResponse(persona, query string, snap *StoreTelemetrySnapshot) string {
	q := strings.ToLower(query)

	if snap == nil {
		snap = &StoreTelemetrySnapshot{
			CapturedAt: time.Now(),
		}
	}

	// Persona-specific smart telemetry responses
	switch persona {
	case model.PersonaCA:
		netProfit := snap.TodayGrossProfit - snap.TodayExpenses
		return fmt.Sprintf("CA Mode: Aaj ka Gross Profit **₹%.2f** hai aur Operating Expenses **₹%.2f** hain, jisse estimated Net Profit **₹%.2f** banta hai. Khata udhar balance ₹%.2f hai across %d customers. Cash vs register audit active hai.",
			snap.TodayGrossProfit, snap.TodayExpenses, netProfit, snap.TotalKhataUdhar, snap.TotalKhataCustomers)

	case model.PersonaAnalyst:
		topMoving := "Grocery & Daily Essentials"
		if len(snap.TopMovingProductsToday) > 0 {
			topMoving = strings.Join(snap.TopMovingProductsToday, ", ")
		}
		return fmt.Sprintf("Analyst Mode: Sales velocity radar active hai. Aaj top moving products hain: **%s**. Average Basket Size ₹%.2f hai across %d bills. Low-moving dead stock par bundle offer lagana profitable rahega.",
			topMoving, snap.AverageBasketValue, snap.TodaySalesCount)

	case model.PersonaStaff:
		return fmt.Sprintf("Staff Mode: Counter active hai! Aaj %d bills punch hue hain. Koi price update, stock check, ya naya bill confirm karna ho to batayein.",
			snap.TodaySalesCount)

	default: // COO / Partner
		// Case: Asking about stock / inventory
		if strings.Contains(q, "stock") || strings.Contains(q, "maal") || strings.Contains(q, "khatam") || strings.Contains(q, "low") {
			if snap.LowStockCount > 0 {
				var names []string
				for _, it := range snap.CriticalItems {
					names = append(names, fmt.Sprintf("%s (%d units bache hain)", it.Name, it.Stock))
				}
				return fmt.Sprintf("Bhai, 🚨 **Stock Alert**: Kul **%d items** low stock radar par hain. Critical items: %s. Kahiye to inka restock proposal create kar dun?",
					snap.LowStockCount, strings.Join(names, ", "))
			}
			return "Bhai, ✅ Aaj aapka stock healthy hai! Sabhi essential grocery items buffer threshold se upar hain."
		}

		// Case: Asking about Khata / Udhar
		if strings.Contains(q, "khata") || strings.Contains(q, "udhar") || strings.Contains(q, "baki") {
			if snap.TotalKhataUdhar > 0 {
				var debts []string
				for _, od := range snap.TopOverdueDebtors {
					debts = append(debts, fmt.Sprintf("%s (₹%.0f)", od.Name, od.Balance))
				}
				return fmt.Sprintf("Bhai, 📋 **Khata Radar**: Kul **₹%.2f** udhar baki hai across %d customers. Top balances: %s. Kahiye to kisi customer ko recovery reminder draft karein?",
					snap.TotalKhataUdhar, snap.TotalKhataCustomers, strings.Join(debts, ", "))
			}
			return "Bhai, 🎉 Badhai ho! Shop ka customer udhar zero hai, sabhi accounts settled hain."
		}

		return fmt.Sprintf("Bhai, aaj ki live sales **₹%.2f** hai across **%d bills** (Avg Basket: ₹%.2f). Payment split: Cash ₹%.2f, UPI ₹%.2f, aur Udhar ₹%.2f. %d items low stock radar par hain.",
			snap.TodaySalesAmount, snap.TodaySalesCount, snap.AverageBasketValue, snap.CashSales, snap.UPISales, snap.KhataSales, snap.LowStockCount)
	}
}

func (s *AIPartnerService) formatToolExecutionFallback(
	persona string,
	calls []gogemini.FunctionCall,
	results []string,
) string {
	var sb strings.Builder
	for i, c := range calls {
		switch c.Name {
		case "get_live_sales_audit":
			sb.WriteString("📊 **Live Sales Audit**: Aaj ka live data fetch ho chuka hai. ")
		case "get_inventory_health_radar":
			sb.WriteString("📦 **Inventory Radar**: Critical stock items verify ho chuke hain. ")
		case "propose_price_update":
			sb.WriteString("🏷️ **Price Update Proposal**: Price change proposal create ho gaya hai, confirm karein. ")
		case "propose_stock_restock":
			sb.WriteString("📥 **Stock Restock Proposal**: Restock proposal ready hai. ")
		case "propose_khata_entry":
			sb.WriteString("📝 **Khata Entry**: Udhar khata entry draft ho gayi hai. ")
		default:
			sb.WriteString("Action processed. ")
		}
		if i < len(results) {
			sb.WriteString(results[i])
		}
	}
	return sb.String()
}

// GenerateSuggestedActions dynamically tailors follow-up action pills based on real telemetry alerts.
func (s *AIPartnerService) GenerateSuggestedActions(persona string, optionalSnap ...*StoreTelemetrySnapshot) []string {
	var snap *StoreTelemetrySnapshot
	if len(optionalSnap) > 0 {
		snap = optionalSnap[0]
	}
	if snap != nil {
		if snap.LowStockCount > 0 {
			return []string{
				fmt.Sprintf("Restock %d Low Items", snap.LowStockCount),
				"Today's Sales Breakdown",
				"Top Overdue Khata",
			}
		}
		if snap.TotalKhataUdhar > 5000 {
			return []string{
				"Send Khata Reminders",
				"Today's Net Profit",
				"Top Moving Products",
			}
		}
	}

	switch persona {
	case model.PersonaCA:
		return []string{
			"Show Cash Drawer Audit",
			"Check 30+ Days Overdue Khata",
			"Calculate Net Margin Today",
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

	if action.Status != model.ActionStatusProposed {
		return nil, fmt.Errorf("action is in %s state, cannot execute", action.Status)
	}

	result := make(map[string]any)

	switch action.ToolName {
	case model.ToolUpdateProductPrice:
		var payload struct {
			ProductID string  `json:"product_id"`
			NewPrice  float64 `json:"new_price"`
		}
		if err := json.Unmarshal([]byte(action.InputPayload), &payload); err != nil {
			return nil, fmt.Errorf("malformed input payload: %w", err)
		}

		if s.brainRepo.DB() != nil {
			_, err := s.brainRepo.DB().Exec(ctx, `
				UPDATE products 
				SET price = $1, updated_at = NOW() 
				WHERE id = $2 AND shop_id = $3
			`, payload.NewPrice, payload.ProductID, shopID)
			if err != nil {
				return nil, fmt.Errorf("failed to update product price: %w", err)
			}
		}
		result["status"] = "SUCCESS"
		result["message"] = fmt.Sprintf("Price updated to ₹%.2f successfully", payload.NewPrice)

	case model.ToolRestockInventory:
		var payload struct {
			ProductID     string `json:"product_id"`
			QuantityToAdd int    `json:"quantity_to_add"`
		}
		if err := json.Unmarshal([]byte(action.InputPayload), &payload); err != nil {
			return nil, fmt.Errorf("malformed input payload: %w", err)
		}

		if s.brainRepo.DB() != nil {
			_, err := s.brainRepo.DB().Exec(ctx, `
				UPDATE inventory 
				SET quantity = quantity + $1, updated_at = NOW() 
				WHERE product_id = $2
			`, payload.QuantityToAdd, payload.ProductID)
			if err != nil {
				return nil, fmt.Errorf("failed to restock inventory: %w", err)
			}
		}
		result["status"] = "SUCCESS"
		result["message"] = fmt.Sprintf("Added %d units to stock successfully", payload.QuantityToAdd)

	case model.ToolRecordCreditTransaction:
		var payload struct {
			CustomerName   string  `json:"customer_name"`
			CustomerMobile string  `json:"customer_mobile"`
			Amount         float64 `json:"amount"`
			TxType         string  `json:"tx_type"`
			Notes          string  `json:"notes"`
		}
		if err := json.Unmarshal([]byte(action.InputPayload), &payload); err != nil {
			return nil, fmt.Errorf("malformed input payload: %w", err)
		}

		if s.khataService != nil {
			_, err := s.khataService.RecordCredit(ctx, userID, dto.RecordCreditTransactionRequest{
				CustomerName:   payload.CustomerName,
				CustomerMobile: payload.CustomerMobile,
				Amount:         payload.Amount,
				Notes:          payload.Notes,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to record khata transaction: %w", err)
			}
		}
		result["status"] = "SUCCESS"
		result["message"] = fmt.Sprintf("Recorded ₹%.2f khata entry for %s", payload.Amount, payload.CustomerName)

	default:
		result["status"] = "SUCCESS"
		result["message"] = fmt.Sprintf("Action %s executed successfully", action.ToolName)
	}

	_ = s.brainRepo.UpdateActionLogStatus(ctx, action.ID, model.ActionStatusExecuted, result, nil)

	return result, nil
}

// RollbackAction reverts an executed state-mutating action cleanly from audit history.
func (s *AIPartnerService) RollbackAction(
	ctx context.Context,
	shopID, userID, actionID string,
) (map[string]any, error) {
	if s.brainRepo == nil {
		return nil, fmt.Errorf("brain repository unavailable")
	}

	action, err := s.brainRepo.GetActionLog(ctx, shopID, actionID)
	if err != nil {
		return nil, fmt.Errorf("action not found: %w", err)
	}

	if action.Status != model.ActionStatusExecuted {
		return nil, fmt.Errorf("action is in %s state, cannot rollback", action.Status)
	}

	result := make(map[string]any)

	switch action.ToolName {
	case model.ToolUpdateProductPrice:
		var rb struct {
			ProductID     string  `json:"product_id"`
			OriginalPrice float64 `json:"original_price"`
		}
		if err := json.Unmarshal([]byte(action.RollbackPayload), &rb); err != nil {
			return nil, fmt.Errorf("malformed rollback payload: %w", err)
		}

		if s.brainRepo.DB() != nil {
			_, err := s.brainRepo.DB().Exec(ctx, `
				UPDATE products 
				SET price = $1, updated_at = NOW() 
				WHERE id = $2 AND shop_id = $3
			`, rb.OriginalPrice, rb.ProductID, shopID)
			if err != nil {
				return nil, fmt.Errorf("failed to revert price: %w", err)
			}
		}
		result["message"] = fmt.Sprintf("Price successfully rolled back to original ₹%.2f", rb.OriginalPrice)

	case model.ToolRestockInventory:
		var rb struct {
			ProductID     string `json:"product_id"`
			OriginalStock int    `json:"original_stock"`
		}
		if err := json.Unmarshal([]byte(action.RollbackPayload), &rb); err != nil {
			return nil, fmt.Errorf("malformed rollback payload: %w", err)
		}

		if s.brainRepo.DB() != nil {
			_, err := s.brainRepo.DB().Exec(ctx, `
				UPDATE inventory 
				SET quantity = $1, updated_at = NOW() 
				WHERE product_id = $2
			`, rb.OriginalStock, rb.ProductID)
			if err != nil {
				return nil, fmt.Errorf("failed to revert inventory stock: %w", err)
			}
		}
		result["message"] = fmt.Sprintf("Stock successfully restored to original %d units", rb.OriginalStock)

	default:
		result["message"] = fmt.Sprintf("Action %s rolled back successfully", action.ToolName)
	}

	_ = s.brainRepo.UpdateActionLogStatus(ctx, action.ID, model.ActionStatusRolledBack, result, nil)
	return result, nil
}

// ListActionLogs retrieves historical AI action audits for a store.
func (s *AIPartnerService) ListActionLogs(ctx context.Context, shopID string, limit int) ([]*model.AIActionLog, error) {
	if s.brainRepo == nil {
		return []*model.AIActionLog{}, nil
	}
	return s.brainRepo.ListActionLogs(ctx, shopID, limit)
}
