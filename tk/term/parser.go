package term

import (
	"strings"
	"unicode/utf8"
)

// ModeState is the runtime-toggled subset of terminal behaviour that DECSET/
// DECRST (CSI ? Pm h/l) turns on and off during a session — distinct from
// Type, which is the static capability set a Type profile advertises. A
// mode can only ever be turned on if the Type says it's supported (see
// Parser.setPrivateMode), so a VT100 profile's SM/RM ?1049h is simply
// ignored rather than corrupting state that profile never renders anyway.
type ModeState struct {
	AppCursor      bool // DECSET 1: arrow keys send SS3 instead of CSI codes
	AppKeypad      bool // DECPAM/DECPNM (ESC = / ESC >)
	BracketedPaste bool // DECSET 2004
	MouseReport    int  // 0 = off, else which tracking mode is on (1000/1002/1003)
	MouseSGR       bool // DECSET 1006: SGR extended coordinate encoding

	AutoWrap bool // DECSET 7 (default on)
}

// Parser is an ECMA-48/VT100/xterm control-sequence parser that decodes a
// byte stream from a PTY and mutates a Grid, following the classic
// Ground/Escape/CSI/OSC state-machine structure every real terminal
// implementation (xterm, VTE, Alacritty's vte crate, ...) uses. Sequences
// this Parser doesn't recognise are consumed (so they never leak into the
// display as garbage) but otherwise ignored.
type Parser struct {
	Grid  *Grid
	Type  Type
	Modes ModeState

	// Respond, if set, is called with bytes that must be written back to the
	// PTY (device attribute/status reports the appli­cation asked for with DA
	// or DSR). SetTitle, if set, is called on OSC 0/1/2 (window/icon title).
	// The remaining callbacks fire on the extra OSC codes; each is a no-op
	// when nil, so a caller only wires what it cares about.
	Respond  func([]byte)
	SetTitle func(string)

	// SetClipboard, if set, is called on OSC 52 with the selection name
	// ("c"=CLIPBOARD, "p"=PRIMARY, "s"=SECONDARY, "0"/"1"/…=clipboard N) and
	// the raw (base64) payload. A "?" payload is a request: the consumer
	// should instead respond via RequestClipboard.
	SetClipboard func(selection, data string)
	// RequestClipboard, if set, is called on OSC 52 when the application asks
	// for the current selection (payload "?"); the consumer should write the
	// value back to the PTY as OSC 52 ; selection ; <base64>.
	RequestClipboard func(selection string)
	// SetHyperlink, if set, is called on OSC 8 with the ';'-separated params
	// (e.g. "id=xyz") and the URI. An empty URI ends the current hyperlink.
	SetHyperlink func(params, uri string)
	// Notify, if set, is called on OSC 9 with a desktop-notification message.
	Notify func(message string)
	// OSC777, if set, is called on OSC 777 with the raw payload, for custom
	// extensions / custom protocols the terminal doesn't interpret itself.
	OSC777 func(payload string)

	curFg, curBg Color
	curAttr      Attr

	state   pstate
	params  []int
	private byte // '?', '<', '=', '>', or 0
	strBuf  []byte
	pending []byte // incomplete UTF-8 bytes carried across Feed calls

	// curLink is the OSC 8 hyperlink URI currently in effect; cells written
	// while it is non-empty get tagged with it (cleared on an empty-URI OSC 8).
	curLink string

	// awaitingST is set while collecting an OSC/DCS/PM/APC/SOS string and we
	// just saw ESC: the next byte decides whether it's '\' (ST, terminating
	// the string) or not (abort back to ground). Kept separate from any other
	// use of stEscape so it can never be confused with a fresh top-level
	// escape sequence.
	awaitingST bool

	// charsetSlot remembers which of G0-G3 (the byte after ESC: '(' ')' '*'
	// '+') is being designated, so stCharset's next byte can be interpreted
	// correctly. altCharset tracks whether G0 is currently designated as DEC
	// Special Graphics (ESC ( 0) rather than the default ASCII (ESC ( B) —
	// the common way ncurses/xterm terminfo entries draw box borders. Only
	// G0 is tracked: SI/SO (switching which of G0-G3 is active) isn't
	// implemented, since real terminfo entries almost always designate G0
	// itself rather than shifting between slots.
	charsetSlot byte
	altCharset  bool
}

type pstate int

const (
	stGround pstate = iota
	stEscape
	stCsi
	stOsc
	stStrIgnore // DCS/PM/APC/SOS: consume to ST, no-op
	stCharset   // ESC ( ) * + seen: consume exactly one more byte, no-op
)

// NewParser returns a Parser bound to g, gated by profile t. AutoWrap starts
// on, matching every real terminal's default.
func NewParser(g *Grid, t Type) *Parser {
	return &Parser{Grid: g, Type: t, Modes: ModeState{AutoWrap: true}}
}

// Feed decodes data, applying every control sequence and printable character
// it contains to the Grid. It may be called repeatedly with arbitrary chunk
// boundaries (as PTY reads naturally are) — an incomplete UTF-8 sequence or
// escape sequence at the end of one call is carried over to the next.
func (p *Parser) Feed(data []byte) {
	for _, b := range data {
		p.step(b)
	}
}

func (p *Parser) step(b byte) {
	switch p.state {
	case stGround:
		p.ground(b)
	case stEscape:
		p.escape(b)
	case stCsi:
		p.csi(b)
	case stOsc:
		p.osc(b)
	case stStrIgnore:
		if b == 0x1b {
			p.state = stEscape
			p.awaitingST = true
		}
	case stCharset:
		if p.charsetSlot == '(' { // only G0 is tracked, see altCharset's doc comment
			p.altCharset = b == '0'
		}
		p.state = stGround
	}
}

func (p *Parser) ground(b byte) {
	switch {
	case b == 0x1b:
		p.enterEscape()
	case b < 0x20 || b == 0x7f:
		p.c0(b)
	case b < 0x80:
		if p.altCharset {
			if r, ok := decSpecialGraphics(b); ok {
				p.putRune(r)
				return
			}
		}
		p.putRune(rune(b))
	default:
		p.utf8Byte(b)
	}
}

// decSpecialGraphics maps a byte to the Unicode rune it represents while G0
// is designated as the VT100 DEC Special Graphics set (ESC ( 0) — the
// classic mechanism ncurses/xterm terminfo entries use to draw box borders
// (smacs=\E(0, rmacs=\E(B). Bytes with no special meaning in this set (most
// of the GL range) aren't listed here and print as plain ASCII, same as
// always. The rendering side (see tk/term.asciiApprox) still degrades these
// to plain-ASCII look-alikes for a Latin-1 core font, but the Grid itself
// stores the real, correct Unicode character either way.
func decSpecialGraphics(b byte) (rune, bool) {
	switch b {
	case '`':
		return '◆', true
	case 'a':
		return '▒', true
	case 'f':
		return '°', true
	case 'g':
		return '±', true
	case 'j':
		return '┘', true
	case 'k':
		return '┐', true
	case 'l':
		return '┌', true
	case 'm':
		return '└', true
	case 'n':
		return '┼', true
	case 'q':
		return '─', true
	case 't':
		return '├', true
	case 'u':
		return '┤', true
	case 'v':
		return '┴', true
	case 'w':
		return '┬', true
	case 'x':
		return '│', true
	case 'y':
		return '≤', true
	case 'z':
		return '≥', true
	case '{':
		return 'π', true
	case '|':
		return '≠', true
	case '}':
		return '£', true
	case '~':
		return '·', true
	}
	return 0, false
}

// utf8Byte accumulates multi-byte UTF-8 sequences across Feed boundaries.
func (p *Parser) utf8Byte(b byte) {
	p.pending = append(p.pending, b)
	if !utf8.FullRune(p.pending) {
		if len(p.pending) >= utf8.UTFMax {
			p.pending = p.pending[:0] // malformed run-on sequence: drop it
		}
		return
	}
	r, size := utf8.DecodeRune(p.pending)
	if r == utf8.RuneError && size <= 1 {
		p.pending = p.pending[:0]
		return
	}
	p.pending = p.pending[:0]
	p.putRune(r)
}

func (p *Parser) putRune(r rune) {
	g := p.Grid
	if p.Modes.AutoWrap && g.CursorCol >= g.Cols {
		g.CursorCol = 0
		g.newline()
	}
	g.PutRune(r, p.curFg, p.curBg, p.curAttr, p.curLink)
}

func (p *Parser) c0(b byte) {
	g := p.Grid
	switch b {
	case '\a': // BEL: no audible/visual bell implemented yet
	case '\b':
		if g.CursorCol > 0 {
			g.CursorCol--
		}
	case '\t':
		next := (g.CursorCol/8 + 1) * 8
		if next >= g.Cols {
			next = g.Cols - 1
		}
		g.CursorCol = next
	case '\n', '\v', '\f':
		g.newline()
	case '\r':
		g.CursorCol = 0
	}
}

func (p *Parser) enterEscape() {
	p.state = stEscape
}

func (p *Parser) escape(b byte) {
	if p.awaitingST { // saw ESC while collecting an OSC/DCS/PM/APC/SOS string
		p.awaitingST = false
		if b == '\\' {
			p.state = stGround
		} else {
			// not a real ST: abort the string and reprocess b as ground input,
			// matching how a stray ESC would resync a real terminal.
			p.state = stGround
			p.step(b)
		}
		return
	}
	switch b {
	case '[':
		p.state = stCsi
		p.params = p.params[:0]
		p.private = 0
	case ']':
		p.state = stOsc
		p.strBuf = p.strBuf[:0]
	case 'P', '^', '_', 'X': // DCS, PM, APC, SOS: unsupported, consume to ST
		p.state = stStrIgnore
	case '7':
		p.Grid.SaveCursor()
		p.state = stGround
	case '8':
		p.Grid.RestoreCursor()
		p.state = stGround
	case 'c':
		p.reset()
	case 'D':
		p.Grid.Index()
		p.state = stGround
	case 'M':
		p.Grid.ScrollDown(1)
		p.state = stGround
	case '=':
		p.Modes.AppKeypad = true
		p.state = stGround
	case '>':
		p.Modes.AppKeypad = false
		p.state = stGround
	case '(', ')', '*', '+': // G0-G3 charset designation
		p.charsetSlot = b
		p.state = stCharset
	default:
		p.state = stGround
	}
}

func (p *Parser) reset() {
	*p.Grid = *NewGrid(p.Grid.Rows, p.Grid.Cols)
	p.Modes = ModeState{AutoWrap: true}
	p.curFg, p.curBg, p.curAttr = Color{}, Color{}, 0
	p.altCharset, p.charsetSlot = false, 0
	p.state = stGround
}

func (p *Parser) csi(b byte) {
	switch {
	case b >= '0' && b <= '9':
		if len(p.params) == 0 {
			p.params = append(p.params, 0)
		}
		last := len(p.params) - 1
		p.params[last] = p.params[last]*10 + int(b-'0')
	case b == ';':
		p.params = append(p.params, 0)
	case b == '?' || b == '<' || b == '=' || b == '>':
		p.private = b
	case b >= 0x20 && b <= 0x2f:
		// intermediate byte: not used by any sequence handled below, ignored.
	case b >= 0x40 && b <= 0x7e:
		p.dispatchCSI(b)
		p.state = stGround
	default:
		p.state = stGround
	}
}

// arg returns params[i] or def if not given/zero (the ECMA-48 convention:
// an omitted or zero parameter means "use the default", nearly always 1).
func (p *Parser) arg(i, def int) int {
	if i >= len(p.params) || p.params[i] == 0 {
		return def
	}
	return p.params[i]
}

func (p *Parser) dispatchCSI(final byte) {
	g := p.Grid
	if p.private != 0 && p.private != '?' {
		return // DEC <, =, > private sequences (rare, e.g. some xterm keyboard modes): unsupported, ignore
	}
	if p.private == '?' {
		p.dispatchPrivate(final)
		return
	}
	switch final {
	case 'A':
		g.SetCursor(g.CursorRow-p.arg(0, 1), g.CursorCol)
	case 'B', 'e':
		g.SetCursor(g.CursorRow+p.arg(0, 1), g.CursorCol)
	case 'C', 'a':
		g.SetCursor(g.CursorRow, g.CursorCol+p.arg(0, 1))
	case 'D':
		g.SetCursor(g.CursorRow, g.CursorCol-p.arg(0, 1))
	case 'E':
		g.SetCursor(g.CursorRow+p.arg(0, 1), 0)
	case 'F':
		g.SetCursor(g.CursorRow-p.arg(0, 1), 0)
	case 'G', '`':
		g.SetCursor(g.CursorRow, p.arg(0, 1)-1)
	case 'H', 'f':
		g.SetCursor(p.arg(0, 1)-1, p.arg(1, 1)-1)
	case 'J':
		g.EraseDisplay(EraseMode(p.arg(0, 0)))
	case 'K':
		g.EraseLine(EraseMode(p.arg(0, 0)))
	case 'S':
		g.ScrollUp(p.arg(0, 1))
	case 'T':
		g.ScrollDown(p.arg(0, 1))
	case 'L':
		p.insertLines(p.arg(0, 1))
	case 'M':
		p.deleteLines(p.arg(0, 1))
	case '@':
		p.insertChars(p.arg(0, 1))
	case 'P':
		p.deleteChars(p.arg(0, 1))
	case 'X':
		p.eraseChars(p.arg(0, 1))
	case 'd':
		g.SetCursor(p.arg(0, 1)-1, g.CursorCol)
	case 'm':
		p.sgr()
	case 'r':
		g.SetScrollRegion(p.arg(0, 1)-1, p.arg(1, g.Rows)-1)
	case 'n':
		p.dsr()
	case 'c':
		p.da()
	case 's':
		g.SaveCursor()
	case 'u':
		g.RestoreCursor()
	}
}

// insertLines/deleteLines (IL/DL) act within the scroll region at the
// cursor's row, the same as ScrollDown/ScrollUp but anchored there instead of
// at the region's own top/bottom — implemented by temporarily narrowing the
// region to [cursorRow, scrollBot].
func (p *Parser) insertLines(n int) {
	g := p.Grid
	if g.CursorRow < g.scrollTop || g.CursorRow > g.scrollBot {
		return
	}
	top, bot := g.scrollTop, g.scrollBot
	g.scrollTop = g.CursorRow
	g.ScrollDown(n)
	g.scrollTop = top
	_ = bot
}

func (p *Parser) deleteLines(n int) {
	g := p.Grid
	if g.CursorRow < g.scrollTop || g.CursorRow > g.scrollBot {
		return
	}
	top := g.scrollTop
	g.scrollTop = g.CursorRow
	g.ScrollUp(n)
	g.scrollTop = top
}

func (p *Parser) insertChars(n int) {
	g := p.Grid
	row := g.cur[g.CursorRow]
	c := g.CursorCol
	if c >= g.Cols {
		return
	}
	if n > g.Cols-c {
		n = g.Cols - c
	}
	copy(row[c+n:], row[c:g.Cols-n])
	for i := c; i < c+n; i++ {
		row[i] = g.blank()
	}
}

func (p *Parser) deleteChars(n int) {
	g := p.Grid
	row := g.cur[g.CursorRow]
	c := g.CursorCol
	if c >= g.Cols {
		return
	}
	if n > g.Cols-c {
		n = g.Cols - c
	}
	copy(row[c:], row[c+n:])
	for i := g.Cols - n; i < g.Cols; i++ {
		row[i] = g.blank()
	}
}

func (p *Parser) eraseChars(n int) {
	g := p.Grid
	row := g.cur[g.CursorRow]
	c := g.CursorCol
	for i := c; i < c+n && i < g.Cols; i++ {
		row[i] = g.blank()
	}
}

// dsr answers CSI 6n (cursor position report) via Respond; other DSR
// requests are acknowledged with nothing, matching xterm's behaviour for
// codes it doesn't specifically implement.
func (p *Parser) dsr() {
	if p.arg(0, 0) == 6 && p.Respond != nil {
		g := p.Grid
		p.Respond([]byte(csiReply(g.CursorRow+1, g.CursorCol+1)))
	}
}

// da answers CSI c (primary device attributes) with a minimal, honest
// VT100-class response — enough for applications that merely probe "is this
// a real terminal" without claiming DEC private extensions this Parser
// doesn't implement (like real terminal emulators, e.g. VT52-only vs.
// ANSI-capable, do differentiate).
func (p *Parser) da() {
	if p.Respond != nil {
		p.Respond([]byte("\x1b[?1;2c"))
	}
}

func csiReply(row, col int) string {
	return "\x1b[" + itoa(row) + ";" + itoa(col) + "R"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// dispatchPrivate handles CSI ? Pm h/l (DECSET/DECRST) and any other `?`
// -private final byte this Parser recognises.
func (p *Parser) dispatchPrivate(final byte) {
	switch final {
	case 'h':
		p.setPrivateModes(true)
	case 'l':
		p.setPrivateModes(false)
	}
}

func (p *Parser) setPrivateModes(on bool) {
	g := p.Grid
	for _, m := range p.params {
		switch m {
		case 1:
			if p.Type.AppCursor {
				p.Modes.AppCursor = on
			}
		case 7:
			p.Modes.AutoWrap = on
		case 25:
			g.CursorVisible = on
		case 47, 1047:
			if p.Type.AltScreen {
				p.setAltScreen(on)
			}
		case 1049:
			if p.Type.AltScreen {
				if on {
					g.SaveCursor()
				}
				p.setAltScreen(on)
				if !on {
					g.RestoreCursor()
				}
			}
		case 1000, 1002, 1003:
			// The tracking mode (which events get reported), independent of
			// 1006's encoding choice below — real apps enable both (e.g.
			// ncurses' xterm mouse support turns on 1000+1006 together), and
			// conflating them into one field previously meant enabling 1006
			// after 1000 silently forgot that click tracking was on at all.
			if p.Type.MouseReport {
				if on {
					p.Modes.MouseReport = m
				} else {
					p.Modes.MouseReport = 0
				}
			}
		case 1006:
			if p.Type.MouseReport {
				p.Modes.MouseSGR = on
			}
		case 2004:
			if p.Type.BracketedPaste {
				p.Modes.BracketedPaste = on
			}
		}
	}
}

func (p *Parser) setAltScreen(on bool) {
	if on {
		p.Grid.EnterAltScreen()
	} else {
		p.Grid.ExitAltScreen()
	}
}

// sgr applies CSI ... m (Select Graphic Rendition): text attributes and
// 8/16/256/truecolor foreground and background, clamped to what the
// Parser's Type actually supports.
// sgr applies one SGR (CSI ... m) sequence's parameters to curFg/curBg/
// curAttr, then mirrors the resulting colours into Grid.DefaultFg/DefaultBg
// so a subsequent erase (EL/ED) or scroll fills with the now-current SGR
// background rather than whatever it was at Term startup — real terminals
// erase with the active background (e.g. a full-width status/menu bar: set
// a background colour, then erase-to-end-of-line to paint the rest of the
// row in it), and Grid.blank's doc comment already promised this even
// though nothing previously kept DefaultBg in sync to make it true.
func (p *Parser) sgr() {
	if len(p.params) == 0 {
		p.params = []int{0}
	}
	for i := 0; i < len(p.params); i++ {
		n := p.params[i]
		switch {
		case n == 0:
			p.curFg, p.curBg, p.curAttr = Color{}, Color{}, 0
		case n == 1:
			p.curAttr |= AttrBold
		case n == 2:
			p.curAttr |= AttrFaint
		case n == 3:
			p.curAttr |= AttrItalic
		case n == 4:
			p.curAttr |= AttrUnderline
		case n == 5:
			p.curAttr |= AttrBlink
		case n == 7:
			p.curAttr |= AttrReverse
		case n == 8:
			p.curAttr |= AttrConceal
		case n == 9:
			p.curAttr |= AttrStrikethrough
		case n == 22:
			p.curAttr &^= AttrBold | AttrFaint
		case n == 23:
			p.curAttr &^= AttrItalic
		case n == 24:
			p.curAttr &^= AttrUnderline
		case n == 25:
			p.curAttr &^= AttrBlink
		case n == 27:
			p.curAttr &^= AttrReverse
		case n == 28:
			p.curAttr &^= AttrConceal
		case n == 29:
			p.curAttr &^= AttrStrikethrough
		case n >= 30 && n <= 37:
			p.curFg = p.clamped(Color{Mode: ColorIndexed, Index: uint8(n - 30)})
		case n == 38:
			c, adv := p.extendedColor(i)
			p.curFg = p.clamped(c)
			i += adv
		case n == 39:
			p.curFg = Color{}
		case n >= 40 && n <= 47:
			p.curBg = p.clamped(Color{Mode: ColorIndexed, Index: uint8(n - 40)})
		case n == 48:
			c, adv := p.extendedColor(i)
			p.curBg = p.clamped(c)
			i += adv
		case n == 49:
			p.curBg = Color{}
		case n >= 90 && n <= 97:
			p.curFg = p.clamped(Color{Mode: ColorIndexed, Index: uint8(n - 90 + 8)})
		case n >= 100 && n <= 107:
			p.curBg = p.clamped(Color{Mode: ColorIndexed, Index: uint8(n - 100 + 8)})
		}
	}
	p.Grid.DefaultFg, p.Grid.DefaultBg = p.curFg, p.curBg
}

// extendedColor parses the 256-colour (38/48;5;N) or truecolor (38/48;2;R;G;B)
// forms starting at params[i+1], returning the colour and how many extra
// params it consumed.
func (p *Parser) extendedColor(i int) (Color, int) {
	if i+1 >= len(p.params) {
		return Color{}, 0
	}
	switch p.params[i+1] {
	case 5:
		if i+2 < len(p.params) {
			return Color{Mode: ColorIndexed, Index: uint8(p.params[i+2])}, 2
		}
	case 2:
		if i+4 < len(p.params) {
			return Color{Mode: ColorRGB,
				R: uint8(p.params[i+2]), G: uint8(p.params[i+3]), B: uint8(p.params[i+4])}, 4
		}
	}
	return Color{}, 1
}

func (p *Parser) clamped(c Color) Color {
	if out, ok := p.Type.clampColor(c); ok {
		return out
	}
	return Color{}
}

func (p *Parser) osc(b byte) {
	switch b {
	case 0x07: // BEL also terminates OSC (xterm convention, in addition to ST)
		p.finishOSC()
		p.state = stGround
	case 0x1b:
		p.state = stEscape
		p.awaitingST = true // next byte must be '\' (ST); handled generically in escape()
		p.finishOSC()
	default:
		p.strBuf = append(p.strBuf, b)
	}
}

func (p *Parser) finishOSC() {
	s := string(p.strBuf)
	// "Ps;Pt": dispatch on Ps, ignoring anything we don't recognise.
	sep := strings.IndexByte(s, ';')
	if sep < 0 {
		return
	}
	ps := s[:sep]
	pt := s[sep+1:]
	switch ps {
	case "0", "1", "2":
		if p.SetTitle != nil {
			p.SetTitle(pt)
		}
	case "52":
		p.oscClipboard(pt)
	case "8":
		p.oscHyperlink(pt)
	case "9":
		if p.Notify != nil {
			p.Notify(pt)
		}
	case "777":
		if p.OSC777 != nil {
			p.OSC777(pt)
		}
	}
}

// oscClipboard handles OSC 52: "Pc;Pd". Pd is base64-encoded data, or "?" to
// request the current selection.
func (p *Parser) oscClipboard(pt string) {
	sep := strings.IndexByte(pt, ';')
	if sep < 0 {
		return
	}
	sel := pt[:sep]
	data := pt[sep+1:]
	if data == "?" {
		if p.RequestClipboard != nil {
			p.RequestClipboard(sel)
		}
		return
	}
	if p.SetClipboard != nil {
		p.SetClipboard(sel, data)
	}
}

// oscHyperlink handles OSC 8: "params;uri". An empty URI ends the current
// hyperlink; the params may themselves be empty ("8;;uri").
func (p *Parser) oscHyperlink(pt string) {
	sep := strings.IndexByte(pt, ';')
	if sep < 0 {
		p.SetHyperlink("", "")
		p.curLink = ""
		return
	}
	params := pt[:sep]
	uri := pt[sep+1:]
	if p.SetHyperlink != nil {
		p.SetHyperlink(params, uri)
	}
	if uri == "" {
		p.curLink = ""
	} else {
		p.curLink = uri
	}
}
