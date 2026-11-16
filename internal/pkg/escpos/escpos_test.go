package goescpos_test

import (
	"bytes"
	"strings"
	"testing"

	goescpos "shopMe/internal/pkg/escpos"
)

func TestBuilder_HardwareInitAndCut(t *testing.T) {
	b := goescpos.New(goescpos.Profile80mm)
	b.Cut(false) // Full cut

	raw := b.Bytes()
	if !bytes.HasPrefix(raw, goescpos.CmdInit) {
		t.Fatal("expected raw stream to start with ESC @ (CmdInit)")
	}
	if !bytes.HasSuffix(raw, goescpos.CmdFullCut) {
		t.Fatal("expected raw stream to end with GS V 0 (CmdFullCut)")
	}

	b2 := goescpos.New(goescpos.Profile58mm)
	b2.Cut(true) // Partial cut
	if !bytes.HasSuffix(b2.Bytes(), goescpos.CmdPartialCut) {
		t.Fatal("expected raw stream to end with GS V 1 (CmdPartialCut)")
	}
}

func TestBuilder_TypographyAndAlignment(t *testing.T) {
	b := goescpos.New()
	b.AlignCenter().
		Bold(true).
		Size(2, 2).
		TextLine("STORE NAME").
		NormalSize().
		Bold(false).
		AlignLeft()

	raw := b.Bytes()
	if !bytes.Contains(raw, goescpos.CmdAlignCenter) {
		t.Fatal("expected CmdAlignCenter")
	}
	if !bytes.Contains(raw, goescpos.CmdBoldOn) {
		t.Fatal("expected CmdBoldOn")
	}
	if !bytes.Contains(raw, []byte{0x1D, 0x21, 0x11}) { // 2x2 scaling
		t.Fatal("expected GS ! 0x11 scaling")
	}
	if !bytes.Contains(raw, []byte("STORE NAME\n")) {
		t.Fatal("expected text line 'STORE NAME\\n'")
	}
}

func TestBuilder_TwoColumnJustification(t *testing.T) {
	// 58mm has 32 chars per line
	b58 := goescpos.New(goescpos.Profile58mm)
	b58.TwoColumn("Subtotal", "Rs. 450.00")

	line := string(b58.Bytes())
	// Skip initial ESC @ command
	cleanLine := strings.TrimPrefix(line, string(goescpos.CmdInit))
	cleanLine = strings.TrimSuffix(cleanLine, "\n")

	if len(cleanLine) != 32 {
		t.Fatalf("expected 58mm TwoColumn line to be exactly 32 chars, got %d ('%s')", len(cleanLine), cleanLine)
	}

	// 80mm has 48 chars per line
	b80 := goescpos.New(goescpos.Profile80mm)
	b80.TwoColumn("Total Payable", "Rs. 1,250.00")

	line80 := string(b80.Bytes())
	cleanLine80 := strings.TrimPrefix(line80, string(goescpos.CmdInit))
	cleanLine80 = strings.TrimSuffix(cleanLine80, "\n")

	if len(cleanLine80) != 48 {
		t.Fatalf("expected 80mm TwoColumn line to be exactly 48 chars, got %d ('%s')", len(cleanLine80), cleanLine80)
	}
}

func TestBuilder_MultiColumnLayouts(t *testing.T) {
	b80 := goescpos.New(goescpos.Profile80mm)
	b80.ThreeColumn("Product Name", "Qty", "Total")
	b80.FourColumn("Product Name", "Qty", "Rate", "Total")

	raw := b80.Bytes()
	if !bytes.Contains(raw, []byte("Product Name")) {
		t.Fatal("expected multi-column to contain header")
	}
}

func TestBuilder_DrawerKickAndBuzzer(t *testing.T) {
	b := goescpos.New()
	b.KickDrawer(2)
	b.KickDrawer(5)
	b.Beep()

	raw := b.Bytes()
	if !bytes.Contains(raw, goescpos.CmdPulsePin2) {
		t.Fatal("expected CmdPulsePin2")
	}
	if !bytes.Contains(raw, goescpos.CmdPulsePin5) {
		t.Fatal("expected CmdPulsePin5")
	}
	if !bytes.Contains(raw, goescpos.CmdBeep) {
		t.Fatal("expected CmdBeep")
	}
}

func TestBuilder_FullRetailReceiptScenario(t *testing.T) {
	b := goescpos.New(goescpos.Profile80mm)

	b.AlignCenter().
		Bold(true).
		Size(2, 2).
		TextLine("SHOPSILO SUPERMART").
		NormalSize().
		Bold(false).
		TextLine("Main Market, Connaught Place, New Delhi").
		TextLine("GSTIN: 07AAAAA0000A1Z5 | Phone: +91 98765 43210").
		Divider('=').
		AlignLeft().
		TextLine("Bill No: POS-2026-0042           Date: 08-Nov-2026 23:45").
		TextLine("Customer: Cash Counter").
		Divider('-').
		Bold(true).
		FourColumn("Item", "Qty", "Rate", "Amount").
		Bold(false).
		Divider('-').
		FourColumn("Aashirvaad Atta 5kg", "1", "240.00", "240.00").
		FourColumn("Fortune Oil 1L", "2", "145.00", "290.00").
		FourColumn("Tata Salt 1kg", "1", "28.00", "28.00").
		Divider('-').
		TwoColumn("Subtotal", "Rs. 558.00").
		TwoColumn("CGST (2.5%)", "Rs. 13.95").
		TwoColumn("SGST (2.5%)", "Rs. 13.95").
		Bold(true).
		Size(1, 2).
		TwoColumn("TOTAL AMOUNT", "Rs. 585.90").
		NormalSize().
		Bold(false).
		Divider('=').
		AlignCenter().
		TextLine("Thank you for shopping with us!").
		TextLine("Visit again: www.shopsilo.in").
		KickDrawer().
		Cut(false)

	stream := b.Bytes()
	if len(stream) < 200 {
		t.Fatalf("receipt stream suspiciously small: %d bytes", len(stream))
	}
	t.Logf("Full retail thermal receipt compiled into %d raw ESC/POS bytes", len(stream))
}
