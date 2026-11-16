package goescpos

// Standard ESC/POS binary command constants.
var (
	// Hardware Control
	CmdInit       = []byte{0x1B, 0x40}             // ESC @ (Initialize printer)
	CmdPulsePin2  = []byte{0x1B, 0x70, 0x00, 0x19, 0xFA} // ESC p 0 25 250 (Kick drawer pin 2)
	CmdPulsePin5  = []byte{0x1B, 0x70, 0x01, 0x19, 0xFA} // ESC p 1 25 250 (Kick drawer pin 5)
	CmdFullCut    = []byte{0x1D, 0x56, 0x00}       // GS V 0 (Full paper cut)
	CmdPartialCut = []byte{0x1D, 0x56, 0x01}       // GS V 1 (Partial paper cut)
	CmdFeedAndCut = []byte{0x1D, 0x56, 0x42, 0x03} // GS V 66 3 (Feed 3 lines and cut)
	CmdBeep       = []byte{0x1B, 0x42, 0x02, 0x02} // ESC B 2 2 (Buzzer beep 2 times)

	// Line feeds
	CmdLF = []byte{0x0A} // Line feed

	// Alignment
	CmdAlignLeft   = []byte{0x1B, 0x61, 0x00} // ESC a 0
	CmdAlignCenter = []byte{0x1B, 0x61, 0x01} // ESC a 1
	CmdAlignRight  = []byte{0x1B, 0x61, 0x02} // ESC a 2

	// Typography & Emphasis
	CmdBoldOn       = []byte{0x1B, 0x45, 0x01} // ESC E 1
	CmdBoldOff      = []byte{0x1B, 0x45, 0x00} // ESC E 0
	CmdUnderlineOn  = []byte{0x1B, 0x2D, 0x01} // ESC - 1
	CmdUnderlineOff = []byte{0x1B, 0x2D, 0x00} // ESC - 0
	CmdInvertOn     = []byte{0x1D, 0x42, 0x01} // GS B 1 (White text on black background)
	CmdInvertOff    = []byte{0x1D, 0x42, 0x00} // GS B 0
)

// Alignment defines text justification.
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// Profile defines physical printer paper geometry.
type Profile struct {
	Name            string
	CharactersWidth int // Typically 32 for 58mm, 48 for 80mm
}

var (
	// Profile58mm standard 2-inch thermal receipt paper (32 characters per line).
	Profile58mm = Profile{Name: "58mm", CharactersWidth: 32}

	// Profile80mm standard 3-inch thermal receipt paper (48 characters per line).
	Profile80mm = Profile{Name: "80mm", CharactersWidth: 48}
)
