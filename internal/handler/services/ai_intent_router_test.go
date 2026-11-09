package services_test

import (
	"strings"
	"testing"

	"shopMe/internal/handler/model"
	"shopMe/internal/handler/services"
)

func TestAIIntentRouter(t *testing.T) {
	router := services.NewAIIntentRouter()

	mockSnap := &services.StoreTelemetrySnapshot{
		TodaySalesAmount:   15450.50,
		TodaySalesCount:    42,
		AverageBasketValue: 367.86,
		CashSales:          8400.00,
		UPISales:           6050.50,
		KhataSales:         1000.00,
		TodayGrossProfit:   3850.00,
		TodayExpenses:      650.00,
		LowStockCount:      2,
		CriticalItems: []services.CriticalItem{
			{Name: "Amul Butter 500g", Stock: 3, Threshold: 10},
			{Name: "Fortune Mustard Oil 1L", Stock: 1, Threshold: 5},
		},
		TotalKhataUdhar:     12500.00,
		TotalKhataCustomers: 8,
		TopOverdueDebtors: []services.OverdueDebtor{
			{Name: "Ramesh Kumar", Mobile: "9876543210", Balance: 4200.00},
			{Name: "Suresh Gupta", Mobile: "9812345678", Balance: 2800.00},
		},
		TopMovingProductsToday: []string{"Tata Salt", "Parle-G", "Fortune Oil"},
	}

	t.Run("Factual Sales Query Resolution (< 2ms)", func(t *testing.T) {
		queries := []string{
			"aaj kitna bikri hua?",
			"aaj ka hisab batao",
			"today total sales",
			"dukan ki kamai kitni hui?",
		}

		for _, q := range queries {
			reply, ok := router.RouteFastPath(q, model.PersonaCOO, mockSnap)
			if !ok {
				t.Fatalf("Expected fast path to match query: %q", q)
			}
			if !strings.Contains(reply, "15450.50") || !strings.Contains(reply, "42 bills") {
				t.Errorf("Sales response missing expected live telemetry for query %q, got: %s", q, reply)
			}
		}
	})

	t.Run("Persona Differentiation in Sales Audit", func(t *testing.T) {
		caReply, ok := router.RouteFastPath("aaj ka hisab", model.PersonaCA, mockSnap)
		if !ok || !strings.Contains(caReply, "CA Sales Audit") {
			t.Errorf("Expected CA formatted response, got: %s", caReply)
		}

		analystReply, ok := router.RouteFastPath("sales kitni hui", model.PersonaAnalyst, mockSnap)
		if !ok || !strings.Contains(analystReply, "Sales Velocity Radar") {
			t.Errorf("Expected Analyst formatted response, got: %s", analystReply)
		}

		staffReply, ok := router.RouteFastPath("bikri batao", model.PersonaStaff, mockSnap)
		if !ok || !strings.Contains(staffReply, "Counter Status") {
			t.Errorf("Expected Staff formatted response, got: %s", staffReply)
		}
	})

	t.Run("Low Stock Radar Intent", func(t *testing.T) {
		queries := []string{
			"kya maal khatam hai?",
			"low stock items kaun se hain?",
			"out of stock kya hai?",
		}

		for _, q := range queries {
			reply, ok := router.RouteFastPath(q, model.PersonaCOO, mockSnap)
			if !ok {
				t.Fatalf("Expected stock intent to match query: %q", q)
			}
			if !strings.Contains(reply, "Amul Butter") || !strings.Contains(reply, "Fortune Mustard Oil") {
				t.Errorf("Stock response missing critical items for %q, got: %s", q, reply)
			}
		}
	})

	t.Run("Khata Overdue Intent", func(t *testing.T) {
		queries := []string{
			"kiska udhar baki hai?",
			"khata me kitna recovery pending hai?",
			"market me kitni udhari hai?",
		}

		for _, q := range queries {
			reply, ok := router.RouteFastPath(q, model.PersonaCOO, mockSnap)
			if !ok {
				t.Fatalf("Expected khata intent to match query: %q", q)
			}
			if !strings.Contains(reply, "12500.00") || !strings.Contains(reply, "Ramesh Kumar") {
				t.Errorf("Khata response missing debtor info for %q, got: %s", q, reply)
			}
		}
	})

	t.Run("Profit & Margin Intent", func(t *testing.T) {
		reply, ok := router.RouteFastPath("aaj ka profit kitna hai?", model.PersonaCOO, mockSnap)
		if !ok {
			t.Fatalf("Expected profit intent to match")
		}
		// Net profit: 3850 - 650 = 3200.00
		if !strings.Contains(reply, "3200.00") || !strings.Contains(reply, "Operating Expenses") {
			t.Errorf("Profit breakdown incorrect, got: %s", reply)
		}
	})

	t.Run("Cash Drawer Register Intent", func(t *testing.T) {
		reply, ok := router.RouteFastPath("galla me kitna cash hai?", model.PersonaCOO, mockSnap)
		if !ok {
			t.Fatalf("Expected cash register intent to match")
		}
		if !strings.Contains(reply, "8400.00") || !strings.Contains(reply, "6050.50") {
			t.Errorf("Cash register breakdown incorrect, got: %s", reply)
		}
	})

	t.Run("Action Verbs Bypass Fast-Path to ReAct Tool Loop", func(t *testing.T) {
		// Queries with action verbs must NOT be handled fast-path; they must go to LLM for ReAct tool execution
		actionQueries := []string{
			"parle g ka price update kardo",
			"amul butter ka 20 packet restock add kar do",
			"ramesh kumar ke khata me 500 rupaye likh do",
			"sugar ka rate change karo",
			"supplier ko order bhejo",
		}

		for _, aq := range actionQueries {
			_, ok := router.RouteFastPath(aq, model.PersonaCOO, mockSnap)
			if ok {
				t.Errorf("Action query %q should NOT have matched fast-path (must delegate to ReAct tool loop)", aq)
			}
		}
	})

	t.Run("Nil Telemetry Snapshot Safety", func(t *testing.T) {
		_, ok := router.RouteFastPath("aaj kitna bikri hua?", model.PersonaCOO, nil)
		if ok {
			t.Errorf("Expected false when telemetry snapshot is nil")
		}
	})
}
