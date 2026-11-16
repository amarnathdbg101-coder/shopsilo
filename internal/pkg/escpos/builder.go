package goescpos

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

// Builder fluently constructs raw ESC/POS binary command streams.
type Builder struct {
	buf     bytes.Buffer
	profile Profile
}

// New creates an initialized Builder with default 80mm profile.
func New(p ...Profile) *Builder {
	prof := Profile80mm
	if len(p) > 0 && p[0].CharactersWidth > 0 {
		prof = p[0]
	}
	b := &Builder{profile: prof}
	b.Init()
	return b
}

// Profile returns current printer geometry profile.
func (b *Builder) Profile() Profile {
	return b.profile
}

// Raw appends arbitrary raw bytes to the output buffer.
func (b *Builder) Raw(data []byte) *Builder {
	b.buf.Write(data)
	return b
}

// Init resets printer hardware settings to defaults (ESC @).
func (b *Builder) Init() *Builder {
	b.buf.Write(CmdInit)
	return b
}

// Align sets text justification.
func (b *Builder) Align(align Alignment) *Builder {
	switch align {
	case AlignCenter:
		b.buf.Write(CmdAlignCenter)
	case AlignRight:
		b.buf.Write(CmdAlignRight)
	default:
		b.buf.Write(CmdAlignLeft)
	}
	return b
}

// AlignLeft sets left justification.
func (b *Builder) AlignLeft() *Builder { return b.Align(AlignLeft) }

// AlignCenter sets center justification.
func (b *Builder) AlignCenter() *Builder { return b.Align(AlignCenter) }

// AlignRight sets right justification.
func (b *Builder) AlignRight() *Builder { return b.Align(AlignRight) }

// Bold toggles double-strike bold typography.
func (b *Builder) Bold(enable bool) *Builder {
	if enable {
		b.buf.Write(CmdBoldOn)
	} else {
		b.buf.Write(CmdBoldOff)
	}
	return b
}

// Underline toggles single-line underline.
func (b *Builder) Underline(enable bool) *Builder {
	if enable {
		b.buf.Write(CmdUnderlineOn)
	} else {
		b.buf.Write(CmdUnderlineOff)
	}
	return b
}

// Invert toggles white text on black background (reverse video).
func (b *Builder) Invert(enable bool) *Builder {
	if enable {
		b.buf.Write(CmdInvertOn)
	} else {
		b.buf.Write(CmdInvertOff)
	}
	return b
}

// Size configures character width and height scaling multipliers (1 to 8).
func (b *Builder) Size(widthMult, heightMult int) *Builder {
	if widthMult < 1 {
		widthMult = 1
	} else if widthMult > 8 {
		widthMult = 8
	}
	if heightMult < 1 {
		heightMult = 1
	} else if heightMult > 8 {
		heightMult = 8
	}

	n := byte(((widthMult - 1) << 4) | (heightMult - 1))
	b.buf.Write([]byte{0x1D, 0x21, n}) // GS ! n
	return b
}

// NormalSize resets character scaling to 1x1.
func (b *Builder) NormalSize() *Builder {
	return b.Size(1, 1)
}

// Feed feeds paper forward by n lines.
func (b *Builder) Feed(lines int) *Builder {
	if lines <= 0 {
		lines = 1
	}
	for i := 0; i < lines; i++ {
		b.buf.Write(CmdLF)
	}
	return b
}

// Text appends text without an automatic newline.
func (b *Builder) Text(s string) *Builder {
	b.buf.WriteString(s)
	return b
}

// TextLine appends text followed by an immediate line feed.
func (b *Builder) TextLine(s string) *Builder {
	b.buf.WriteString(s)
	b.buf.Write(CmdLF)
	return b
}

// Divider prints a full-width divider line of repeating character (e.g. '-' or '=').
func (b *Builder) Divider(char rune) *Builder {
	line := strings.Repeat(string(char), b.profile.CharactersWidth)
	return b.TextLine(line)
}

// TwoColumn formats a line with left text justified to the left and right text justified to the right.
// Automatically pads the space between columns to fit the profile width.
func (b *Builder) TwoColumn(left, right string) *Builder {
	width := b.profile.CharactersWidth
	leftLen := utf8.RuneCountInString(left)
	rightLen := utf8.RuneCountInString(right)

	if leftLen+rightLen >= width {
		// Overflow: print on separate lines
		b.TextLine(left)
		b.AlignRight().TextLine(right).AlignLeft()
		return b
	}

	spaces := width - (leftLen + rightLen)
	b.Text(left)
	b.Text(strings.Repeat(" ", spaces))
	b.TextLine(right)
	return b
}

// ThreeColumn formats three columns with proportional or custom character widths.
func (b *Builder) ThreeColumn(col1, col2, col3 string, customWidths ...int) *Builder {
	total := b.profile.CharactersWidth
	var w1, w2, w3 int

	if len(customWidths) >= 3 {
		w1, w2, w3 = customWidths[0], customWidths[1], customWidths[2]
	} else {
		// Default 50% - 20% - 30% split
		w1 = int(float64(total) * 0.50)
		w2 = int(float64(total) * 0.20)
		w3 = total - (w1 + w2)
	}

	c1 := padRight(truncate(col1, w1), w1)
	c2 := padLeft(truncate(col2, w2), w2)
	c3 := padLeft(truncate(col3, w3), w3)

	b.TextLine(c1 + c2 + c3)
	return b
}

// FourColumn formats four columns (e.g. Item, Qty, Rate, Total).
func (b *Builder) FourColumn(col1, col2, col3, col4 string, customWidths ...int) *Builder {
	total := b.profile.CharactersWidth
	var w1, w2, w3, w4 int

	if len(customWidths) >= 4 {
		w1, w2, w3, w4 = customWidths[0], customWidths[1], customWidths[2], customWidths[3]
	} else {
		// Default 40% - 15% - 20% - 25% split
		w1 = int(float64(total) * 0.40)
		w2 = int(float64(total) * 0.15)
		w3 = int(float64(total) * 0.20)
		w4 = total - (w1 + w2 + w3)
	}

	c1 := padRight(truncate(col1, w1), w1)
	c2 := padLeft(truncate(col2, w2), w2)
	c3 := padLeft(truncate(col3, w3), w3)
	c4 := padLeft(truncate(col4, w4), w4)

	b.TextLine(c1 + c2 + c3 + c4)
	return b
}

// Cut executes paper cutting. If partial is true, performs partial cut leaving tab.
func (b *Builder) Cut(partial bool) *Builder {
	b.Feed(3)
	if partial {
		b.buf.Write(CmdPartialCut)
	} else {
		b.buf.Write(CmdFullCut)
	}
	return b
}

// KickDrawer triggers an electrical pulse to open the connected RJ11/RJ12 cash drawer.
// pin default is 2 (CmdPulsePin2); pin 5 triggers CmdPulsePin5.
func (b *Builder) KickDrawer(pin ...int) *Builder {
	if len(pin) > 0 && pin[0] == 5 {
		b.buf.Write(CmdPulsePin5)
	} else {
		b.buf.Write(CmdPulsePin2)
	}
	return b
}

// Beep sounds the internal hardware buzzer if present.
func (b *Builder) Beep() *Builder {
	b.buf.Write(CmdBeep)
	return b
}

// Bytes returns the compiled ESC/POS byte slice ready to send over TCP, Bluetooth, or USB.
func (b *Builder) Bytes() []byte {
	return b.buf.Bytes()
}

// Reset clears the buffer while retaining profile.
func (b *Builder) Reset() *Builder {
	b.buf.Reset()
	b.Init()
	return b
}

func truncate(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	if maxRunes > 1 {
		return string(runes[:maxRunes-1]) + "."
	}
	return string(runes[:maxRunes])
}

func padRight(s string, totalWidth int) string {
	count := utf8.RuneCountInString(s)
	if count >= totalWidth {
		return s
	}
	return s + strings.Repeat(" ", totalWidth-count)
}

func padLeft(s string, totalWidth int) string {
	count := utf8.RuneCountInString(s)
	if count >= totalWidth {
		return s
	}
	return strings.Repeat(" ", totalWidth-count) + s
}
