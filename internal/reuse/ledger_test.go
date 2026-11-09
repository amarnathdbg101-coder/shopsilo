package reuse_test

import (
	"testing"
	"time"

	"shopMe/internal/reuse"

	goledger "github.com/amarnathdbg101-coder/go/go-ledger"
)

func TestReuse_LedgerKhataDoubleEntry(t *testing.T) {
	shopID := "shop_999"
	khataID := "khata_ramesh"

	// 1. Give Credit
	creditEntry, err := reuse.BuildKhataDoubleEntry(
		"tx_1", shopID, khataID, "GIVE_CREDIT", 450.00, "credit", "bill_001", time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to build credit double entry: %v", err)
	}

	if creditEntry.TotalDebit() != 45000 || creditEntry.TotalCredit() != 45000 {
		t.Fatalf("expected 45000 paisa, got Dr=%d Cr=%d", creditEntry.TotalDebit(), creditEntry.TotalCredit())
	}

	// 2. Receive Payment via UPI
	payEntry, err := reuse.BuildKhataDoubleEntry(
		"tx_2", shopID, khataID, "RECEIVE_PAYMENT", 450.00, "upi", "upi_ref_101", time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to build payment double entry: %v", err)
	}

	if payEntry.TotalDebit() != 45000 || payEntry.TotalCredit() != 45000 {
		t.Fatalf("expected 45000 paisa, got Dr=%d Cr=%d", payEntry.TotalDebit(), payEntry.TotalCredit())
	}

	// 3. Post to Ledger
	ledger := reuse.NewLedger()
	khataAcc, _ := goledger.NewAccount("khata:"+khataID, "Ramesh Khata", goledger.TypeAsset, "INR")
	salesAcc, _ := goledger.NewAccount("sales:"+shopID, "Shop Sales", goledger.TypeRevenue, "INR")
	bankAcc, _ := goledger.NewAccount("bank:"+shopID, "Shop Bank UPI", goledger.TypeAsset, "INR")

	_ = ledger.RegisterAccount(khataAcc)
	_ = ledger.RegisterAccount(salesAcc)
	_ = ledger.RegisterAccount(bankAcc)

	_ = ledger.Post(creditEntry)
	_ = ledger.Post(payEntry)

	khataAfter, _ := ledger.GetAccount("khata:" + khataID)
	bankAfter, _ := ledger.GetAccount("bank:" + shopID)

	if khataAfter.BalanceFloat() != 0.00 {
		t.Fatalf("expected Ramesh khata to be settled (0.00), got %.2f", khataAfter.BalanceFloat())
	}
	if bankAfter.BalanceFloat() != 450.00 {
		t.Fatalf("expected Bank to have received 450.00, got %.2f", bankAfter.BalanceFloat())
	}
}

func TestReuse_CalculateKhataAging(t *testing.T) {
	now := time.Now().UTC()
	items := []reuse.AgingItem{
		{Reference: "inv_recent", Date: now.AddDate(0, 0, -5), AmountSubunits: 15000},
		{Reference: "inv_old", Date: now.AddDate(0, 0, -75), AmountSubunits: 35000},
	}

	report := reuse.CalculateKhataAging(items, now)
	if report.CurrentFloat() != 150.00 {
		t.Fatalf("expected current 150.00, got %.2f", report.CurrentFloat())
	}
	if report.Days61To90Float() != 350.00 {
		t.Fatalf("expected 61-90 days 350.00, got %.2f", report.Days61To90Float())
	}
	if report.TotalFloat() != 500.00 {
		t.Fatalf("expected total 500.00, got %.2f", report.TotalFloat())
	}
}
