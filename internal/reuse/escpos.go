package reuse

import (
	"fmt"
	"strings"

	"shopMe/internal/handler/model"

	goescpos "github.com/amarnathdbg101-coder/go/go-escpos"
)

// Re-export ESC/POS builder types.
type ESCPOSBuilder = goescpos.Builder
type ESCPOSProfile = goescpos.Profile

var (
	ESCPOSProfile58mm = goescpos.Profile58mm
	ESCPOSProfile80mm = goescpos.Profile80mm
)

// NewESCPOSBuilder creates a new ESC/POS stream compiler.
func NewESCPOSBuilder(profile ...ESCPOSProfile) *ESCPOSBuilder {
	return goescpos.New(profile...)
}

// GenerateESCPOSBillReceipt compiles a POSBill into raw ESC/POS bytes for thermal printers (58mm or 80mm).
// Payload is < 1 KB, executing in < 15ms over Bluetooth SPP, USB raw pipe, or TCP socket (port 9100).
func GenerateESCPOSBillReceipt(bill *model.POSBill, is58mm bool, kickDrawer bool) ([]byte, error) {
	if bill == nil {
		return nil, fmt.Errorf("bill cannot be nil")
	}

	prof := goescpos.Profile80mm
	if is58mm {
		prof = goescpos.Profile58mm
	}

	b := goescpos.New(prof)

	storeName := "ShopSilo Retail Store"
	storeAddress := ""
	storePhone := ""

	if bill.Shop != nil {
		if strings.TrimSpace(bill.Shop.Name) != "" {
			storeName = strings.TrimSpace(bill.Shop.Name)
		}
		if strings.TrimSpace(bill.Shop.Address) != "" {
			storeAddress = strings.TrimSpace(bill.Shop.Address)
		}
		if strings.TrimSpace(bill.Shop.Phone) != "" {
			storePhone = strings.TrimSpace(bill.Shop.Phone)
		}
	}

	// 1. HEADER
	b.AlignCenter().
		Bold(true).
		Size(2, 2).
		TextLine(storeName).
		NormalSize().
		Bold(false)

	if storeAddress != "" {
		b.TextLine(storeAddress)
	}
	if storePhone != "" {
		b.TextLine("Phone: " + storePhone)
	}

	b.Divider('=')

	// 2. METADATA
	b.AlignLeft()
	b.TwoColumn("Bill No: "+bill.BillNumber, bill.CreatedAt.Format("02-Jan-06 15:04"))
	if bill.CustomerPhone != "" {
		b.TextLine("Customer: " + bill.CustomerPhone)
	}
	b.TextLine("Payment Mode: " + strings.ToUpper(bill.PaymentMethod))
	b.Divider('-')

	// 3. ITEM TABLE
	if is58mm {
		// 58mm 3-column layout
		b.Bold(true).ThreeColumn("Item", "Qty", "Amt").Bold(false)
		b.Divider('-')
		for _, it := range bill.Items {
			b.ThreeColumn(it.ProductName, fmt.Sprintf("%d", it.Quantity), fmt.Sprintf("%.2f", it.TotalPrice))
		}
	} else {
		// 80mm 4-column layout
		b.Bold(true).FourColumn("Item", "Qty", "Rate", "Amount").Bold(false)
		b.Divider('-')
		for _, it := range bill.Items {
			b.FourColumn(it.ProductName, fmt.Sprintf("%d", it.Quantity), fmt.Sprintf("%.2f", it.UnitPrice), fmt.Sprintf("%.2f", it.TotalPrice))
		}
	}

	b.Divider('-')

	// 4. TOTALS
	if bill.DiscountAmount > 0 {
		b.TwoColumn("Subtotal", fmt.Sprintf("Rs. %.2f", bill.Subtotal))
		b.TwoColumn("Discount", fmt.Sprintf("-Rs. %.2f", bill.DiscountAmount))
	}

	b.Bold(true).Size(1, 2)
	b.TwoColumn("TOTAL PAYABLE", fmt.Sprintf("Rs. %.2f", bill.TotalAmount))
	b.NormalSize().Bold(false)

	// Payment split details
	if bill.CashAmount > 0 || bill.OnlineAmount > 0 || bill.KhataAmount > 0 {
		b.Divider('.')
		if bill.CashAmount > 0 {
			b.TwoColumn("Paid by Cash", fmt.Sprintf("Rs. %.2f", bill.CashAmount))
		}
		if bill.OnlineAmount > 0 {
			b.TwoColumn("Paid by UPI/Online", fmt.Sprintf("Rs. %.2f", bill.OnlineAmount))
		}
		if bill.KhataAmount > 0 {
			b.TwoColumn("Added to Khata", fmt.Sprintf("Rs. %.2f", bill.KhataAmount))
		}
	}

	b.Divider('=')

	// 5. FOOTER
	b.AlignCenter().
		TextLine("Thank you for your visit!").
		TextLine("Please retain this receipt for warranty.")

	// 6. CASH DRAWER & CUT
	if kickDrawer && bill.CashAmount > 0 {
		b.KickDrawer()
	}
	b.Cut(false)

	return b.Bytes(), nil
}
