# go-ledger

[![Go Reference](https://pkg.go.dev/badge/github.com/amarnathdbg101-coder/go/go-ledger.svg)](https://pkg.go.dev/github.com/amarnathdbg101-coder/go/go-ledger)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Production-grade double-entry bookkeeping engine, general ledger, and accounts receivable aging processor with zero external dependencies.

---

## Features
- **Deterministic Balance Invariance**: Strict double-entry accounting guarantees $\sum \text{Debits} = \sum \text{Credits}$. Unbalanced transactions are rejected.
- **Integer Minor Subunits**: Computes in integer minor currency units (Paisa, Cents) to eliminate floating-point arithmetic errors.
- **Normal Balance Accounting**: Automatically adjusts balances based on account category (`ASSET`, `LIABILITY`, `EQUITY`, `REVENUE`, `EXPENSE`).
- **Trial Balance Verification**: One-call mathematical proof that the general ledger is in perfect equilibrium.
- **Reversal & Audit Trail**: Implements clean contra-entry generation (`CreateReversal`) for transparent audit trails.
- **Accounts Receivable Aging**: Automatically calculates delinquency brackets (Current 0-30d, 31-60d, 61-90d, 90+ days).
- **Zero Third-Party Dependencies**: Pure Go standard library (`math`, `sync`, `time`).

---

## Installation

```bash
go get github.com/amarnathdbg101-coder/go/go-ledger@v1.0.0
```

---

## Quickstart

### 1. Register Accounts

```go
ledger := goledger.NewLedger()

cash, _ := goledger.NewAccount("acc_cash", "Cash Drawer", goledger.TypeAsset, "INR")
khata, _ := goledger.NewAccount("acc_khata", "Customer Udhar Khata", goledger.TypeAsset, "INR")
sales, _ := goledger.NewAccount("acc_sales", "Sales Revenue", goledger.TypeRevenue, "INR")

ledger.RegisterAccount(cash)
ledger.RegisterAccount(khata)
ledger.RegisterAccount(sales)
```

### 2. Post a Double-Entry Sale

```go
// Customer purchases ₹500 groceries on credit (Udhar)
tx := goledger.NewJournalEntry("tx_101", "Credit sale", "inv_001", time.Now())
tx.AddDebitFloat("acc_khata", 500.00, "Udhar to customer")
tx.AddCreditFloat("acc_sales", 500.00, "Grocery sales revenue")

if err := ledger.Post(tx); err != nil {
    log.Fatal(err)
}
```

### 3. Delinquency Aging Buckets

```go
items := []goledger.AgingItem{
    {Reference: "inv_1", Date: time.Now().AddDate(0, 0, -15), AmountSubunits: 50000}, // 15 days
    {Reference: "inv_2", Date: time.Now().AddDate(0, 0, -45), AmountSubunits: 30000}, // 45 days
}

report := goledger.CalculateAging(items, time.Now())
fmt.Printf("Current: ₹%.2f, 31-60d: ₹%.2f\n", report.CurrentFloat(), report.Days31To60Float())
```

---

## License
MIT License.
