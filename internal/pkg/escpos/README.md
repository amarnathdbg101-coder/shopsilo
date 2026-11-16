# go-escpos

[![Go Reference](https://pkg.go.dev/badge/github.com/amarnathdbg101-coder/go/go-escpos.svg)](https://pkg.go.dev/github.com/amarnathdbg101-coder/go/go-escpos)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

High-speed, zero-dependency ESC/POS thermal receipt raw byte stream compiler for Point of Sale (POS) retail printers (58mm & 80mm).

---

## Features
- **Ultra-Lightweight Raw Output**: Compiles receipts into ~900 bytes, printing instantly (< 15ms) over Bluetooth, USB, or TCP LAN sockets.
- **Hardware Primitives**: Hardware paper cutting (full/partial), cash drawer solenoid kicks (pin 2 / pin 5), buzzer beeps.
- **Responsive Geometry**: Automatic column padding for 58mm (32 chars) and 80mm (48 chars) thermal rolls.
- **Fluent Typography**: Text justification (Left, Center, Right), font scaling (1x to 8x), bold emphasis, underlines, reverse video (white on black).
- **Tabular Formatting**: Built-in helpers for `TwoColumn`, `ThreeColumn`, and `FourColumn` product tables.
- **Zero Third-Party Dependencies**: Pure Go standard library (`bytes`, `strings`, `unicode/utf8`).

---

## Installation

```bash
go get github.com/amarnathdbg101-coder/go/go-escpos@v1.0.0
```

---

## Quickstart

```go
b := goescpos.New(goescpos.Profile80mm)

b.AlignCenter().
    Bold(true).
    Size(2, 2).
    TextLine("MY RETAIL STORE").
    NormalSize().
    Bold(false).
    TextLine("Connaught Place, New Delhi").
    Divider('=').
    AlignLeft().
    TextLine("Bill No: POS-101 | 08-Nov-2026").
    Divider('-').
    FourColumn("Item", "Qty", "Rate", "Amount").
    Divider('-').
    FourColumn("Atta 5kg", "1", "240.00", "240.00").
    FourColumn("Oil 1L", "2", "145.00", "290.00").
    Divider('-').
    TwoColumn("Subtotal", "Rs. 530.00").
    Bold(true).
    TwoColumn("TOTAL", "Rs. 530.00").
    Bold(false).
    Divider('=').
    AlignCenter().
    TextLine("Thank you!").
    KickDrawer(). // Pop open cash drawer
    Cut(false)    // Cut receipt paper

rawBytes := b.Bytes()
// Send rawBytes directly to TCP socket, USB port, or Bluetooth SPP
```

---

## License
MIT License.
