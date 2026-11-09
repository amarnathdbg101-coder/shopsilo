package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"

	gogemini "github.com/amarnathdbg101-coder/go/go-gemini"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// AIToolDispatcher provides declarative tool declarations and deterministic execution against PostgreSQL.
type AIToolDispatcher struct {
	db         *pgxpool.Pool
	brainRepo  *repository.AIBrainRepo
	telemetry  *TelemetryProvider
	auditor    *AIFinancialAuditor
	khataNudge *AIKhataNudgeEngine
	logger     *zap.Logger
}

// NewAIToolDispatcher initializes the tool execution engine.
func NewAIToolDispatcher(
	db *pgxpool.Pool,
	brainRepo *repository.AIBrainRepo,
	telemetry *TelemetryProvider,
	logger *zap.Logger,
) *AIToolDispatcher {
	return &AIToolDispatcher{
		db:         db,
		brainRepo:  brainRepo,
		telemetry:  telemetry,
		auditor:    NewAIFinancialAuditor(db, logger),
		khataNudge: NewAIKhataNudgeEngine(db, logger),
		logger:     logger,
	}
}

// GetToolDeclarations exports the Gemini tools available to the AI Business Partner.
func (d *AIToolDispatcher) GetToolDeclarations() []gogemini.Tool {
	return []gogemini.Tool{
		{
			FunctionDeclarations: []gogemini.FunctionDeclaration{
				{
					Name:        "get_live_sales_audit",
					Description: "Fetch live factual sales, net profit, bill count, cash and UPI breakdown for today or this week.",
					Parameters: map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"period": map[string]any{
								"type":        "STRING",
								"description": "Period to audit: 'today', 'this_week', or 'this_month'",
								"enum":        []string{"today", "this_week", "this_month"},
							},
						},
						"required": []string{"period"},
					},
				},
				{
					Name:        "get_inventory_health_radar",
					Description: "Query critical low-stock and out-of-stock items requiring urgent supplier replenishment.",
					Parameters: map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"limit": map[string]any{
								"type":        "INTEGER",
								"description": "Maximum number of items to return (default 10)",
							},
						},
					},
				},
				{
					Name:        "get_overdue_khata_radar",
					Description: "List customer udhar (credit) ledger accounts with pending balances and overdue amounts.",
					Parameters: map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"min_balance": map[string]any{
								"type":        "NUMBER",
								"description": "Filter customers with minimum pending balance in Rupees (e.g. 100)",
							},
						},
					},
				},
				{
					Name:        "search_store_catalog",
					Description: "Search store products by name, title, or barcode to check exact selling price, cost price, and stock in hand.",
					Parameters: map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"query": map[string]any{
								"type":        "STRING",
								"description": "Product search query (e.g. 'Atta', 'Amul Butter', 'Oil')",
							},
						},
						"required": []string{"query"},
					},
				},
				{
					Name:        "propose_price_update",
					Description: "Propose an official selling price change for a product. Generates a confirmed action card with diff preview and rollback capability.",
					Parameters: map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"product_name": map[string]any{
								"type":        "STRING",
								"description": "Exact or approximate name of the product",
							},
							"new_price": map[string]any{
								"type":        "NUMBER",
								"description": "New target selling price in Rupees",
							},
							"reason": map[string]any{
								"type":        "STRING",
								"description": "Business justification (e.g. 'Supplier price hike', 'Diwali discount')",
							},
						},
						"required": []string{"product_name", "new_price"},
					},
				},
				{
					Name:        "propose_stock_restock",
					Description: "Propose an inventory restock addition for a product. Generates a confirmed action card with diff preview.",
					Parameters: map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"product_name": map[string]any{
								"type":        "STRING",
								"description": "Name of the product being restocked",
							},
							"quantity_to_add": map[string]any{
								"type":        "INTEGER",
								"description": "Number of units added to inventory",
							},
							"reason": map[string]any{
								"type":        "STRING",
								"description": "Restock notes (e.g. 'Morning supplier delivery')",
							},
						},
						"required": []string{"product_name", "quantity_to_add"},
					},
				},
				{
					Name:        "propose_khata_entry",
					Description: "Propose recording an udhar credit transaction or payment receipt for a customer in their Khata ledger.",
					Parameters: map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"customer_name": map[string]any{
								"type":        "STRING",
								"description": "Customer full or display name",
							},
							"mobile": map[string]any{
								"type":        "STRING",
								"description": "10-digit mobile number",
							},
							"amount": map[string]any{
								"type":        "NUMBER",
								"description": "Transaction amount in Rupees",
							},
							"tx_type": map[string]any{
								"type":        "STRING",
								"description": "GIVE_CREDIT (Udhar diya) or RECEIVE_PAYMENT (Jama liya)",
								"enum":        []string{"GIVE_CREDIT", "RECEIVE_PAYMENT"},
							},
							"notes": map[string]any{
								"type":        "STRING",
								"description": "Item description or payment reference",
							},
						},
						"required": []string{"customer_name", "amount", "tx_type"},
					},
				},
				{
					Name:        "audit_financial_cash_reconciliation",
					Description: "Reconcile register cash drawer with bills, safe drops, and petty expenses to catch theft, cash shortages, or discrepancies.",
					Parameters: map[string]any{
						"type":       "OBJECT",
						"properties": map[string]any{},
					},
				},
				{
					Name:        "get_tax_gst_breakdown",
					Description: "Audit GST turnover, CGST, SGST, and exempt sales for GSTR-1 and GSTR-3B tax preparation.",
					Parameters: map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"period": map[string]any{
								"type":        "STRING",
								"description": "Period to calculate tax for: 'today', 'this_week', or 'this_month'",
								"enum":        []string{"today", "this_week", "this_month"},
							},
						},
					},
				},
				{
					Name:        "generate_khata_payment_nudge",
					Description: "Generate a psychology-based respectful WhatsApp payment reminder with instant WhatsApp click-to-chat deep link and UPI payment link.",
					Parameters: map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"customer_name_or_phone": map[string]any{
								"type":        "STRING",
								"description": "Customer mobile number or full name in Khata ledger",
							},
							"tone": map[string]any{
								"type":        "STRING",
								"description": "Tone of reminder: 'gentle' (polite & friendly), 'micro_plan' (daily installments for large balance), or 'formal' (firm statement notice)",
								"enum":        []string{"gentle", "micro_plan", "formal"},
							},
						},
						"required": []string{"customer_name_or_phone"},
					},
				},
				{
					Name:        "get_dead_stock_clearance_radar",
					Description: "Audit stagnant dead stock items with locked working capital and generate smart liquidation combos.",
					Parameters: map[string]any{
						"type": "OBJECT",
						"properties": map[string]any{
							"min_days_unsold": map[string]any{
								"type":        "INTEGER",
								"description": "Minimum days without a sale to qualify as dead stock (default 30 days)",
							},
						},
					},
				},
			},
		},
	}
}

// ExecuteToolCall dispatches a single Gemini function call to its corresponding Go implementation.
func (d *AIToolDispatcher) ExecuteToolCall(
	ctx context.Context,
	shopID, userID string,
	call gogemini.FunctionCall,
) (map[string]any, error) {
	switch call.Name {
	case "get_live_sales_audit":
		period, _ := call.Args["period"].(string)
		return d.executeSalesAudit(ctx, shopID, period)

	case "get_inventory_health_radar":
		limit := 10
		if l, ok := call.Args["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}
		return d.executeInventoryRadar(ctx, shopID, limit)

	case "get_overdue_khata_radar":
		minBal := 0.0
		if mb, ok := call.Args["min_balance"].(float64); ok {
			minBal = mb
		}
		return d.executeKhataRadar(ctx, shopID, minBal)

	case "search_store_catalog":
		q, _ := call.Args["query"].(string)
		return d.executeSearchCatalog(ctx, shopID, q)

	case "propose_price_update":
		pName, _ := call.Args["product_name"].(string)
		newPrice, _ := call.Args["new_price"].(float64)
		reason, _ := call.Args["reason"].(string)
		return d.executeProposePrice(ctx, shopID, userID, pName, newPrice, reason)

	case "propose_stock_restock":
		pName, _ := call.Args["product_name"].(string)
		qty, _ := call.Args["quantity_to_add"].(float64)
		reason, _ := call.Args["reason"].(string)
		return d.executeProposeStock(ctx, shopID, userID, pName, int(qty), reason)

	case "propose_khata_entry":
		cName, _ := call.Args["customer_name"].(string)
		mob, _ := call.Args["mobile"].(string)
		amt, _ := call.Args["amount"].(float64)
		txType, _ := call.Args["tx_type"].(string)
		notes, _ := call.Args["notes"].(string)
		return d.executeProposeKhata(ctx, shopID, userID, cName, mob, amt, txType, notes)

	case "audit_financial_cash_reconciliation":
		return d.executeCashAudit(ctx, shopID)

	case "get_tax_gst_breakdown":
		period, _ := call.Args["period"].(string)
		if period == "" {
			period = "today"
		}
		return d.executeGSTTaxAudit(ctx, shopID, period)

	case "generate_khata_payment_nudge":
		ident, _ := call.Args["customer_name_or_phone"].(string)
		tone, _ := call.Args["tone"].(string)
		return d.executeKhataNudge(ctx, shopID, ident, tone)

	case "get_dead_stock_clearance_radar":
		days := 30
		if dVal, ok := call.Args["min_days_unsold"].(float64); ok && dVal > 0 {
			days = int(dVal)
		}
		return d.executeDeadStockAudit(ctx, shopID, days)

	default:
		return map[string]any{
			"status": "error",
			"error":  fmt.Sprintf("unknown tool function: %s", call.Name),
		}, nil
	}
}

func (d *AIToolDispatcher) executeSalesAudit(ctx context.Context, shopID, period string) (map[string]any, error) {
	if d.telemetry != nil && (period == "today" || period == "") {
		snap, err := d.telemetry.GetSnapshot(ctx, shopID)
		if err == nil {
			return map[string]any{
				"period":             "today",
				"total_sales_amount": snap.TodaySalesAmount,
				"total_bills_count":  snap.TodaySalesCount,
				"average_basket":     snap.AverageBasketValue,
				"cash_sales":         snap.CashSales,
				"upi_sales":          snap.UPISales,
				"khata_sales":        snap.KhataSales,
				"gross_profit":       snap.TodayGrossProfit,
				"operating_expenses": snap.TodayExpenses,
				"net_profit":         snap.TodayGrossProfit - snap.TodayExpenses,
			}, nil
		}
	}

	dateFilter := "pb.created_at >= NOW() - INTERVAL '7 days'"
	if period == "this_month" {
		dateFilter = "pb.created_at >= NOW() - INTERVAL '30 days'"
	}

	query := fmt.Sprintf(`
		SELECT
			COALESCE(SUM(pb.total_amount), 0),
			COUNT(*),
			COALESCE(SUM(pb.net_profit), 0),
			COALESCE(SUM(pb.cash_amount), 0),
			COALESCE(SUM(pb.online_amount), 0),
			COALESCE(SUM(pb.khata_amount), 0)
		FROM pos_bills pb
		WHERE pb.shop_id = $1 AND pb.status = 'completed' AND %s
	`, dateFilter)

	var totalSales, profit, cash, upi, khata float64
	var count int
	_ = d.db.QueryRow(ctx, query, shopID).Scan(&totalSales, &count, &profit, &cash, &upi, &khata)

	return map[string]any{
		"period":             period,
		"total_sales_amount": totalSales,
		"total_bills_count":  count,
		"gross_profit":       profit,
		"cash_sales":         cash,
		"upi_sales":          upi,
		"khata_sales":        khata,
	}, nil
}

func (d *AIToolDispatcher) executeInventoryRadar(ctx context.Context, shopID string, limit int) (map[string]any, error) {
	query := `
		SELECT 
			p.id, p.title, p.price, i.quantity, i.low_stock_threshold
		FROM products p
		JOIN inventory i ON p.id = i.product_id
		WHERE p.shop_id = $1 AND p.is_active = true AND i.quantity <= i.low_stock_threshold
		ORDER BY i.quantity ASC
		LIMIT $2
	`
	rows, err := d.db.Query(ctx, query, shopID, limit)
	if err != nil {
		return map[string]any{"error": err.Error()}, nil
	}
	defer rows.Close()

	var items []map[string]any
	for rows.Next() {
		var id, title string
		var price float64
		var qty, threshold int
		if err := rows.Scan(&id, &title, &price, &qty, &threshold); err == nil {
			urgency := "LOW_STOCK"
			if qty <= 0 {
				urgency = "OUT_OF_STOCK"
			}
			items = append(items, map[string]any{
				"id":        id,
				"name":      title,
				"price":     price,
				"stock":     qty,
				"threshold": threshold,
				"urgency":   urgency,
			})
		}
	}

	return map[string]any{
		"low_stock_items": items,
		"count":           len(items),
	}, nil
}

func (d *AIToolDispatcher) executeKhataRadar(ctx context.Context, shopID string, minBal float64) (map[string]any, error) {
	query := `
		SELECT id, customer_name, customer_mobile, current_balance, trust_badge
		FROM customer_khata
		WHERE shop_id = $1 AND current_balance >= $2
		ORDER BY current_balance DESC
		LIMIT 10
	`
	rows, err := d.db.Query(ctx, query, shopID, minBal)
	if err != nil {
		return map[string]any{"error": err.Error()}, nil
	}
	defer rows.Close()

	var customers []map[string]any
	var totalBal float64
	for rows.Next() {
		var id, name, mobile, badge string
		var bal float64
		if err := rows.Scan(&id, &name, &mobile, &bal, &badge); err == nil {
			totalBal += bal
			customers = append(customers, map[string]any{
				"id":          id,
				"name":        name,
				"mobile":      mobile,
				"balance":     bal,
				"trust_badge": badge,
			})
		}
	}

	return map[string]any{
		"customers":     customers,
		"total_pending": totalBal,
		"count":         len(customers),
	}, nil
}

func (d *AIToolDispatcher) executeSearchCatalog(ctx context.Context, shopID, q string) (map[string]any, error) {
	query := `
		SELECT p.id, p.title, p.price, p.cost_price, COALESCE(p.barcode, ''), COALESCE(i.quantity, 0)
		FROM products p
		LEFT JOIN inventory i ON p.id = i.product_id
		WHERE p.shop_id = $1 AND p.is_active = true AND (p.title ILIKE $2 OR p.barcode = $3)
		LIMIT 5
	`
	pattern := "%" + strings.TrimSpace(q) + "%"
	rows, err := d.db.Query(ctx, query, shopID, pattern, strings.TrimSpace(q))
	if err != nil {
		return map[string]any{"error": err.Error()}, nil
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var id, title, barcode string
		var price, costPrice float64
		var stock int
		if err := rows.Scan(&id, &title, &price, &costPrice, &barcode, &stock); err == nil {
			results = append(results, map[string]any{
				"id":         id,
				"name":       title,
				"price":      price,
				"cost_price": costPrice,
				"barcode":    barcode,
				"stock":      stock,
			})
		}
	}

	return map[string]any{
		"results": results,
		"count":   len(results),
	}, nil
}

func (d *AIToolDispatcher) executeProposePrice(
	ctx context.Context,
	shopID, userID, pName string,
	newPrice float64,
	reason string,
) (map[string]any, error) {
	// Find product by title
	var pID, title string
	var currentPrice float64
	err := d.db.QueryRow(ctx, `
		SELECT id, title, price 
		FROM products 
		WHERE shop_id = $1 AND title ILIKE $2 AND is_active = true 
		LIMIT 1
	`, shopID, "%"+strings.TrimSpace(pName)+"%").Scan(&pID, &title, &currentPrice)

	if err != nil {
		return map[string]any{
			"status": "error",
			"error":  fmt.Sprintf("Product %q not found in shop catalog", pName),
		}, nil
	}

	inputPayload, _ := json.Marshal(map[string]any{
		"product_id":   pID,
		"product_name": title,
		"old_price":    currentPrice,
		"new_price":    newPrice,
		"reason":       reason,
	})

	diffPreview := fmt.Sprintf("Price Update for %s: ₹%.2f ➔ ₹%.2f (%s)", title, currentPrice, newPrice, reason)
	outBytes, _ := json.Marshal(map[string]string{"preview": diffPreview})
	rollbackPayload, _ := json.Marshal(map[string]any{
		"product_id":     pID,
		"original_price": currentPrice,
	})

	action := &model.AIActionLog{
		ShopID:          shopID,
		UserID:          userID,
		ToolName:        model.ToolUpdateProductPrice,
		InputPayload:    inputPayload,
		OutputResult:    outBytes,
		Status:          model.ActionStatusProposed,
		RollbackPayload: rollbackPayload,
	}

	if d.brainRepo != nil {
		_ = d.brainRepo.RecordActionLog(ctx, action)
	}

	return map[string]any{
		"status":        "PROPOSED",
		"action_id":     action.ID,
		"product_id":    pID,
		"product_name":  title,
		"old_price":     currentPrice,
		"new_price":     newPrice,
		"diff_preview":  diffPreview,
		"user_must_tap": "Confirm in app to apply price change",
	}, nil
}

func (d *AIToolDispatcher) executeProposeStock(
	ctx context.Context,
	shopID, userID, pName string,
	qtyToAdd int,
	reason string,
) (map[string]any, error) {
	var pID, title string
	var currentStock int
	err := d.db.QueryRow(ctx, `
		SELECT p.id, p.title, COALESCE(i.quantity, 0)
		FROM products p
		LEFT JOIN inventory i ON p.id = i.product_id
		WHERE p.shop_id = $1 AND p.title ILIKE $2 AND p.is_active = true
		LIMIT 1
	`, shopID, "%"+strings.TrimSpace(pName)+"%").Scan(&pID, &title, &currentStock)

	if err != nil {
		return map[string]any{
			"status": "error",
			"error":  fmt.Sprintf("Product %q not found in catalog", pName),
		}, nil
	}

	targetStock := currentStock + qtyToAdd
	inputPayload, _ := json.Marshal(map[string]any{
		"product_id":      pID,
		"product_name":    title,
		"current_stock":   currentStock,
		"quantity_to_add": qtyToAdd,
		"target_stock":    targetStock,
		"reason":          reason,
	})

	diffPreview := fmt.Sprintf("Stock Restock for %s: %d units ➔ %d units (+%d)", title, currentStock, targetStock, qtyToAdd)
	outBytes, _ := json.Marshal(map[string]string{"preview": diffPreview})
	rollbackPayload, _ := json.Marshal(map[string]any{
		"product_id":     pID,
		"original_stock": currentStock,
	})

	action := &model.AIActionLog{
		ShopID:          shopID,
		UserID:          userID,
		ToolName:        model.ToolRestockInventory,
		InputPayload:    inputPayload,
		OutputResult:    outBytes,
		Status:          model.ActionStatusProposed,
		RollbackPayload: rollbackPayload,
	}

	if d.brainRepo != nil {
		_ = d.brainRepo.RecordActionLog(ctx, action)
	}

	return map[string]any{
		"status":        "PROPOSED",
		"action_id":     action.ID,
		"product_id":    pID,
		"product_name":  title,
		"current_stock": currentStock,
		"target_stock":  targetStock,
		"diff_preview":  diffPreview,
		"user_must_tap": "Confirm in app to apply restock",
	}, nil
}

func (d *AIToolDispatcher) executeProposeKhata(
	ctx context.Context,
	shopID, userID, cName, mob string,
	amt float64,
	txType, notes string,
) (map[string]any, error) {
	inputPayload, _ := json.Marshal(map[string]any{
		"customer_name":   cName,
		"customer_mobile": mob,
		"amount":          amt,
		"tx_type":         txType,
		"notes":           notes,
	})

	typeLabel := "Udhar Diya"
	if txType == "RECEIVE_PAYMENT" {
		typeLabel = "Jama Liya"
	}

	diffPreview := fmt.Sprintf("Khata Entry: %s - ₹%.2f to %s (%s)", typeLabel, amt, cName, notes)
	outBytes, _ := json.Marshal(map[string]string{"preview": diffPreview})

	action := &model.AIActionLog{
		ShopID:          shopID,
		UserID:          userID,
		ToolName:        model.ToolRecordCreditTransaction,
		InputPayload:    inputPayload,
		OutputResult:    outBytes,
		Status:          model.ActionStatusProposed,
		RollbackPayload: json.RawMessage("{}"),
	}

	if d.brainRepo != nil {
		_ = d.brainRepo.RecordActionLog(ctx, action)
	}

	return map[string]any{
		"status":        "PROPOSED",
		"action_id":     action.ID,
		"customer_name": cName,
		"amount":        amt,
		"tx_type":       txType,
		"diff_preview":  diffPreview,
		"user_must_tap": "Confirm in app to post khata transaction",
	}, nil
}

func (d *AIToolDispatcher) executeCashAudit(ctx context.Context, shopID string) (map[string]any, error) {
	if d.auditor == nil {
		return map[string]any{"status": "error", "message": "financial auditor unavailable"}, nil
	}
	report, err := d.auditor.AuditCashReconciliation(ctx, shopID)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	res := map[string]any{
		"session_id":         report.SessionID,
		"shift_name":         report.ShiftName,
		"staff_name":         report.StaffName,
		"status":             report.Status,
		"opening_cash":       report.OpeningCash,
		"cash_sales":         report.CashSalesTotal,
		"expenses":           report.ExpensesTotal,
		"safe_drops":         report.SafeDropTotal,
		"expected_cash":      report.ExpectedCash,
		"discrepancy":        report.DiscrepancyAmount,
		"manual_opens":       report.ManualOpensCount,
		"bill_voids":         report.BillVoidsCount,
		"safe_drop_alert":    report.SafeDropAlert,
		"audit_summary_text": report.AuditSummaryText,
	}
	if report.ActualCash != nil {
		res["actual_cash"] = *report.ActualCash
	}
	return res, nil
}

func (d *AIToolDispatcher) executeGSTTaxAudit(ctx context.Context, shopID, period string) (map[string]any, error) {
	if d.auditor == nil {
		return map[string]any{"status": "error", "message": "financial auditor unavailable"}, nil
	}
	report, err := d.auditor.AuditGSTTaxBreakdown(ctx, shopID, period)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	return map[string]any{
		"period":             report.Period,
		"total_bills":        report.TotalBillsAudited,
		"gross_turnover":     report.TotalGrossSales,
		"customer_discounts": report.TotalDiscounts,
		"net_turnover":       report.NetTaxableSales,
		"exempt_sales":       report.ExemptSales,
		"cgst":               report.TotalCGST,
		"sgst":               report.TotalSGST,
		"total_gst":          report.TotalTaxAmount,
		"audit_summary_text": report.AuditSummaryText,
	}, nil
}

func (d *AIToolDispatcher) executeKhataNudge(ctx context.Context, shopID, ident, tone string) (map[string]any, error) {
	if d.khataNudge == nil {
		return map[string]any{"status": "error", "message": "khata nudge engine unavailable"}, nil
	}
	var shopName, shopUPI string
	if d.db != nil {
		_ = d.db.QueryRow(ctx, "SELECT name, COALESCE(upi_id, '') FROM shops WHERE id = $1 LIMIT 1", shopID).Scan(&shopName, &shopUPI)
	}
	result, err := d.khataNudge.GenerateRecoveryNudge(ctx, shopID, ident, tone, shopName, shopUPI)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	return map[string]any{
		"customer_name":      result.CustomerName,
		"customer_mobile":    result.CustomerMobile,
		"current_balance":    result.CurrentBalance,
		"trust_badge":        result.TrustBadge,
		"days_overdue":       result.DaysOverdue,
		"tone":               result.Tone,
		"message_text":       result.MessageText,
		"whatsapp_deep_link": result.WhatsAppDeepLink,
		"upi_payment_link":   result.UPIPaymentLink,
	}, nil
}

func (d *AIToolDispatcher) executeDeadStockAudit(ctx context.Context, shopID string, days int) (map[string]any, error) {
	if d.auditor == nil {
		return map[string]any{"status": "error", "message": "financial auditor unavailable"}, nil
	}
	report, err := d.auditor.AuditDeadStockWorkingCapital(ctx, shopID, days)
	if err != nil {
		return map[string]any{"status": "error", "error": err.Error()}, nil
	}
	return map[string]any{
		"total_dead_items":      report.TotalDeadItems,
		"total_locked_capital":  report.TotalLockedCapital,
		"top_stagnant_products": report.TopStagnantProducts,
		"suggested_liquidation": report.SuggestedLiquidation,
	}, nil
}
