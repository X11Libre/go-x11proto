// Package term is a VT100/ANSI/xterm-family terminal emulator built on the tk
// toolkit: a fixed-size cell grid (grid.go), an ECMA-48 control-sequence
// parser that mutates it (parser.go), a PTY/subprocess layer (pty.go), a
// keyboard-to-byte-stream encoder (keys.go), and the Term widget itself
// (term.go) that wires all four to a tk_core.Window.
//
// It deliberately does not share TextView's data model: a terminal is a
// byte-stream-driven, fixed Rows x Cols grid of independently styled cells
// with no client-side insert/delete/undo, not a line-buffer text editor.
package term

// Type describes one terminal emulation profile: which control sequences and
// features it honours, and the TERM value to export to the child process so
// its own terminfo-based idea of the terminal matches what this package
// actually implements. The parser consults these flags to decide whether to
// act on a sequence or ignore it (e.g. a VT100 profile ignores SGR 256-colour
// and alternate-screen requests a real vt100 would never send), so one engine
// serves every profile — profiles differ in gating, not in a separate parser.
type Type struct {
	Name string // exported as $TERM for the child process

	Colors         int  // 8, 16, 256, or 1<<24 for truecolor (SGR 38/48;2;r;g;b)
	AltScreen      bool // DECSET/DECRST 1049/1047/1049 (alternate screen buffer)
	AppCursor      bool // DECSET/DECRST 1 (application vs. normal cursor-key codes)
	AppKeypad      bool // DECPAM/DECPNM (application vs. numeric keypad codes)
	MouseReport    bool // DECSET/DECRST 1000/1002/1003/1006 (mouse tracking)
	BracketedPaste bool // DECSET/DECRST 2004
	Scrollback     bool // keep erased/scrolled-off lines instead of dropping them
}

// Predefined profiles, ordered from least to most capable. XTerm256Color is
// the default a Term picks if Type is left zero-valued.
var (
	VT100 = Type{
		Name:   "vt100",
		Colors: 0, // no SGR colour at all, only the text attributes (bold/underline/etc.)
	}
	VT220 = Type{
		Name:   "vt220",
		Colors: 0,
	}
	LinuxConsole = Type{
		Name:       "linux",
		Colors:     16,
		AppCursor:  true,
		AppKeypad:  true,
		Scrollback: true,
	}
	XTerm = Type{
		Name:           "xterm",
		Colors:         16,
		AltScreen:      true,
		AppCursor:      true,
		AppKeypad:      true,
		MouseReport:    true,
		BracketedPaste: true,
		Scrollback:     true,
	}
	XTerm256Color = Type{
		Name:           "xterm-256color",
		Colors:         256,
		AltScreen:      true,
		AppCursor:      true,
		AppKeypad:      true,
		MouseReport:    true,
		BracketedPaste: true,
		Scrollback:     true,
	}
	XTermTrueColor = Type{
		Name:           "xterm-direct",
		Colors:         1 << 24,
		AltScreen:      true,
		AppCursor:      true,
		AppKeypad:      true,
		MouseReport:    true,
		BracketedPaste: true,
		Scrollback:     true,
	}
)

// clampColor maps an SGR-requested colour mode down to what t supports: a
// truecolor (r,g,b) request on a 256-colour Type is quantized to the nearest
// palette index, a 256-colour index on a 16-colour Type is mapped to its
// nearest basic colour, and any colour at all on a Type with Colors == 0
// yields (Default, false) so the caller falls back to no colour.
func (t Type) clampColor(c Color) (Color, bool) {
	if t.Colors == 0 {
		return Color{}, false
	}
	if t.Colors >= 1<<24 {
		return c, true
	}
	if c.Mode == ColorRGB && t.Colors < 1<<24 {
		return Color{Mode: ColorIndexed, Index: nearestIndexed(c, t.Colors)}, true
	}
	if c.Mode == ColorIndexed && int(c.Index) >= t.Colors {
		return Color{Mode: ColorIndexed, Index: nearestBasic(c.Index)}, true
	}
	return c, true
}
