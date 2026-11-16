package goledger_test

import (
	"testing"
	"time"

	goledger "shopMe/internal/pkg/ledger"
)

func TestJournalEntry_BalanceValidation(t *testing.T) {
	entry := goledger.NewJournalEntry("tx_1", "Test Sale", "inv_101", time.Now())

	// Less than 2 lines
	entry.AddDebit("cash", 1000, "Cash")
	if err := entry.Validate(); err != goledger.ErrInsufficientPostingLines {
		t.Fatalf("expected ErrInsufficientPostingLines, got %v", err)
	}

	// Unbalanced
	entry.AddCredit("sales", 800, "Sales")
	if err := entry.Validate(); err != goledger.ErrUnbalancedEntry {
		t.Fatalf("expected ErrUnbalancedEntry, got %v", err)
	}

	// Negative amount
	badEntry := goledger.NewJournalEntry("tx_bad", "Bad", "", time.Now())
	badEntry.AddDebit("cash", -500, "Negative")
	badEntry.AddCredit("sales", -500, "Negative")
	if err := badEntry.Validate(); err != goledger.ErrZeroOrNegativeAmount {
		t.Fatalf("expected ErrZeroOrNegativeAmount, got %v", err)
	}

	// Perfectly balanced
	entry.Lines[1].AmountSubunits = 1000
	if err := entry.Validate(); err != nil {
		t.Fatalf("unexpected validation error on balanced entry: %v", err)
	}
}

func TestLedger_FullRetailSaleAndSettlementLifecycle(t *testing.T) {
	ledger := goledger.NewLedger()

	cashAcc, _ := goledger.NewAccount("acc_cash", "Cash in Drawer", goledger.TypeAsset, "INR")
	khataAcc, _ := goledger.NewAccount("acc_khata_ramesh", "Ramesh Udhar Khata", goledger.TypeAsset, "INR")
	salesAcc, _ := goledger.NewAccount("acc_sales", "Grocery Sales Revenue", goledger.TypeRevenue, "INR")

	_ = ledger.RegisterAccount(cashAcc)
	_ = ledger.RegisterAccount(khataAcc)
	_ = ledger.RegisterAccount(salesAcc)

	// Step 1: Customer Ramesh takes Rs. 750 groceries on Udhar (Credit Sale)
	// Dr: Ramesh Khata (Asset) Rs. 750
	// Cr: Sales Revenue (Revenue) Rs. 750
	saleTx := goledger.NewJournalEntry("tx_sale_1", "Grocery udhar sale", "bill_999", time.Now())
	saleTx.AddDebitFloat("acc_khata_ramesh", 750.00, "Rice and Dal on udhar")
	saleTx.AddCreditFloat("acc_sales", 750.00, "Counter retail sale")

	if err := ledger.Post(saleTx); err != nil {
		t.Fatalf("failed to post sale txn: %v", err)
	}

	// Verify Ramesh owes Rs. 750
	rameshState, _ := ledger.GetAccount("acc_khata_ramesh")
	if rameshState.BalanceFloat() != 750.00 {
		t.Fatalf("expected Ramesh khata balance Rs. 750, got %.2f", rameshState.BalanceFloat())
	}

	// Verify Revenue is Rs. 750
	salesState, _ := ledger.GetAccount("acc_sales")
	if salesState.BalanceFloat() != 750.00 {
		t.Fatalf("expected Sales revenue Rs. 750, got %.2f", salesState.BalanceFloat())
	}

	// Step 2: Ramesh visits shop 3 days later and pays Rs. 500 cash partial settlement
	// Dr: Cash in Drawer (Asset) Rs. 500
	// Cr: Ramesh Khata (Asset) Rs. 500
	payTx := goledger.NewJournalEntry("tx_pay_1", "Ramesh partial payment", "receipt_101", time.Now())
	payTx.AddDebitFloat("acc_cash", 500.00, "Cash received at POS")
	payTx.AddCreditFloat("acc_khata_ramesh", 500.00, "Udhar jama kiya")

	if err := ledger.Post(payTx); err != nil {
		t.Fatalf("failed to post payment txn: %v", err)
	}

	// Verify Ramesh remaining debt is Rs. 250
	rameshState2, _ := ledger.GetAccount("acc_khata_ramesh")
	if rameshState2.BalanceFloat() != 250.00 {
		t.Fatalf("expected Ramesh remaining balance Rs. 250, got %.2f", rameshState2.BalanceFloat())
	}

	// Verify Cash is Rs. 500
	cashState, _ := ledger.GetAccount("acc_cash")
	if cashState.BalanceFloat() != 500.00 {
		t.Fatalf("expected Cash balance Rs. 500, got %.2f", cashState.BalanceFloat())
	}

	// Step 3: Check Trial Balance mathematical equality
	tb := ledger.TrialBalance()
	if !tb.IsBalanced {
		t.Fatalf("trial balance must be balanced, Dr=%.2f Cr=%.2f", tb.TotalDebitFloat(), tb.TotalCreditFloat())
	}
	if tb.TotalDebitFloat() != 750.00 || tb.TotalCreditFloat() != 750.00 {
		t.Fatalf("expected trial balance sum Rs. 750, got Dr=%.2f Cr=%.2f", tb.TotalDebitFloat(), tb.TotalCreditFloat())
	}
}

func TestJournalEntry_Reversal(t *testing.T) {
	ledger := goledger.NewLedger()
	accA, _ := goledger.NewAccount("a", "Account A", goledger.TypeAsset, "INR")
	accB, _ := goledger.NewAccount("b", "Account B", goledger.TypeRevenue, "INR")
	_ = ledger.RegisterAccount(accA)
	_ = ledger.RegisterAccount(accB)

	orig := goledger.NewJournalEntry("orig", "Mistake entry", "ref", time.Now())
	orig.AddDebitFloat("a", 100.0, "Debit A")
	orig.AddCreditFloat("b", 100.0, "Credit B")
	_ = ledger.Post(orig)

	// Create and post reversal
	rev, err := orig.CreateReversal("rev_1", "User cancelled order")
	if err != nil {
		t.Fatalf("failed to create reversal: %v", err)
	}

	if err := ledger.Post(rev); err != nil {
		t.Fatalf("failed to post reversal: %v", err)
	}

	aAfter, _ := ledger.GetAccount("a")
	bAfter, _ := ledger.GetAccount("b")

	if aAfter.BalanceFloat() != 0.0 || bAfter.BalanceFloat() != 0.0 {
		t.Fatalf("expected both accounts to be 0 after reversal, got A=%.2f, B=%.2f", aAfter.BalanceFloat(), bAfter.BalanceFloat())
	}
}

func TestCalculateAging(t *testing.T) {
	now := time.Now().UTC()

	items := []goledger.AgingItem{
		{Reference: "bill_1", Date: now.AddDate(0, 0, -10), AmountSubunits: 10000},  // 10 days old (Current)
		{Reference: "bill_2", Date: now.AddDate(0, 0, -45), AmountSubunits: 20000},  // 45 days old (31-60)
		{Reference: "bill_3", Date: now.AddDate(0, 0, -80), AmountSubunits: 30000},  // 80 days old (61-90)
		{Reference: "bill_4", Date: now.AddDate(0, 0, -120), AmountSubunits: 40000}, // 120 days old (90+)
	}

	report := goledger.CalculateAging(items, now)

	if report.CurrentFloat() != 100.00 {
		t.Fatalf("expected Current Rs. 100, got %.2f", report.CurrentFloat())
	}
	if report.Days31To60Float() != 200.00 {
		t.Fatalf("expected 31-60 Rs. 200, got %.2f", report.Days31To60Float())
	}
	if report.Days61To90Float() != 300.00 {
		t.Fatalf("expected 61-90 Rs. 300, got %.2f", report.Days61To90Float())
	}
	if report.Over90DaysFloat() != 400.00 {
		t.Fatalf("expected Over90 Rs. 400, got %.2f", report.Over90DaysFloat())
	}
	if report.TotalFloat() != 1000.00 {
		t.Fatalf("expected Total Rs. 1000, got %.2f", report.TotalFloat())
	}
	if !report.TotalOutstandingFlag {
		t.Fatal("expected has_overdue flag to be true")
	}
}
