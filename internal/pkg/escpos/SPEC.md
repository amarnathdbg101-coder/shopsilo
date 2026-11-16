# Technical Specification: go-escpos

## 1. Domain & Purpose
`go-escpos` is a deterministic, zero-dependency ESC/POS raw byte stream compiler for Point of Sale (POS) thermal receipt printers (EPSON, TVS, NGX, MUNBYN, HOIN, RONGTA). It compiles high-speed formatted receipts, multi-column product tables, hardware paper cuts, cash drawer kick pulses, and buzzer beeps for standard 58mm (32-col) and 80mm (48-col) thermal rolls.

## 2. Architecture & Design Principles
- **Binary ESC/POS Protocol Compliance**: Generates standard command sequences (`ESC @` init, `ESC a` alignment, `ESC E` bold, `GS !` character scaling, `GS V` paper cut, `ESC p` cash drawer pulse).
- **Sub-Kilobyte Payload Footprint**: Compiles full itemized grocery receipts into ~800 to 1,000 raw bytes, executing in < 15ms over Bluetooth, USB, or TCP socket connections without requiring heavy PDF rasterizers.
- **Strict Profile Geometry**: Formats table columns (`TwoColumn`, `ThreeColumn`, `FourColumn`) based on target physical paper width (32 chars for 58mm / 2-inch roll; 48 chars for 80mm / 3-inch roll).
- **Hardware Integration**: Built-in support for RJ11/RJ12 electrical cash drawer solenoids (pin 2 / pin 5) and hardware paper cutters (full and partial cut).
- **Zero Third-Party Dependencies**: Pure Go standard library (`bytes`, `strings`, `unicode/utf8`).

## 3. API Surface
- `type Alignment int (AlignLeft, AlignCenter, AlignRight)`
- `type Profile struct { Name string, CharactersWidth int }`
- `var Profile58mm, Profile80mm Profile`
- `type Builder struct`
- `func New(p ...Profile) *Builder`
- `func (b *Builder) Init() *Builder`
- `func (b *Builder) Align(align Alignment) *Builder`
- `func (b *Builder) AlignLeft() *Builder`
- `func (b *Builder) AlignCenter() *Builder`
- `func (b *Builder) AlignRight() *Builder`
- `func (b *Builder) Bold(enable bool) *Builder`
- `func (b *Builder) Underline(enable bool) *Builder`
- `func (b *Builder) Invert(enable bool) *Builder`
- `func (b *Builder) Size(widthMult, heightMult int) *Builder`
- `func (b *Builder) NormalSize() *Builder`
- `func (b *Builder) Feed(lines int) *Builder`
- `func (b *Builder) Text(s string) *Builder`
- `func (b *Builder) TextLine(s string) *Builder`
- `func (b *Builder) Divider(char rune) *Builder`
- `func (b *Builder) TwoColumn(left, right string) *Builder`
- `func (b *Builder) ThreeColumn(col1, col2, col3 string, colWidths ...int) *Builder`
- `func (b *Builder) FourColumn(col1, col2, col3, col4 string, colWidths ...int) *Builder`
- `func (b *Builder) Cut(partial bool) *Builder`
- `func (b *Builder) KickDrawer(pin ...int) *Builder`
- `func (b *Builder) Beep() *Builder`
- `func (b *Builder) Bytes() []byte`
