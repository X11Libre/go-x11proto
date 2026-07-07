package term

import (
	"github.com/X11Libre/go-x11proto/tk/keyboard"
)

// EncodeKey turns a decoded key event into the byte sequence a real terminal
// would send to its PTY. appCursor selects CSI (\x1b[A) vs. SS3 (\x1bOA)
// arrow-key codes, matching DECSET/DECRST 1 (see ModeState.AppCursor) — the
// one place key encoding actually depends on terminal mode; every other key
// here sends the same bytes regardless of mode, matching xterm's own
// behaviour. Returns nil for an event that shouldn't send anything.
func EncodeKey(e keyboard.Event, appCursor bool) []byte {
	if e.Ctrl && e.Key == keyboard.KeyNone {
		if b, ok := ctrlByte(e.Rune); ok {
			return []byte{b}
		}
	}
	switch e.Key {
	case keyboard.KeyEnter:
		return []byte{'\r'}
	case keyboard.KeyBackspace:
		return []byte{0x7f}
	case keyboard.KeyDelete:
		return []byte("\x1b[3~")
	case keyboard.KeyTab:
		if e.Shift {
			return []byte("\x1b[Z")
		}
		return []byte{'\t'}
	case keyboard.KeyEscape:
		return []byte{0x1b}
	case keyboard.KeyLeft:
		return arrow('D', appCursor)
	case keyboard.KeyRight:
		return arrow('C', appCursor)
	case keyboard.KeyUp:
		return arrow('A', appCursor)
	case keyboard.KeyDown:
		return arrow('B', appCursor)
	case keyboard.KeyHome:
		return []byte("\x1b[1~")
	case keyboard.KeyEnd:
		return []byte("\x1b[4~")
	case keyboard.KeyPageUp:
		return []byte("\x1b[5~")
	case keyboard.KeyPageDown:
		return []byte("\x1b[6~")
	}
	if n := e.Key - keyboard.KeyF1; n >= 0 && int(n) < len(fkeySeq) {
		return fkeySeq[n]
	}
	if e.Printable() {
		return []byte(string(e.Rune))
	}
	return nil
}

// fkeySeq holds F1-F12's byte sequences, indexed by (Key - KeyF1). This is
// the xterm/vt220 convention every terminfo "xterm*" entry advertises (kf1=
// \EOP, ..., kf12=\E[24~) — F1-F4 are SS3 letters (the old ANSI/VT100 "PF"
// keys), F5-F12 are CSI ~-terminated numbers with gaps at 16 and 22 (VT220
// assigned those slots to Help/Do, which most keyboards don't have and
// xterm never reused). mc and anything else keying off $TERM=xterm* expects
// exactly this.
var fkeySeq = [12][]byte{
	[]byte("\x1bOP"), []byte("\x1bOQ"), []byte("\x1bOR"), []byte("\x1bOS"),
	[]byte("\x1b[15~"), []byte("\x1b[17~"), []byte("\x1b[18~"), []byte("\x1b[19~"),
	[]byte("\x1b[20~"), []byte("\x1b[21~"), []byte("\x1b[23~"), []byte("\x1b[24~"),
}

func arrow(final byte, appCursor bool) []byte {
	if appCursor {
		return []byte{0x1b, 'O', final}
	}
	return []byte{0x1b, '[', final}
}

// ctrlByte computes the control code for Ctrl+<rune>, the small, well-defined
// subset every terminal supports: letters map to 0x01-0x1a (Ctrl+A..Ctrl+Z,
// case-insensitive), plus the four punctuation keys ECMA-48 assigns codes
// 0x1c-0x1f to.
func ctrlByte(r rune) (byte, bool) {
	switch {
	case r >= 'a' && r <= 'z':
		return byte(r-'a') + 1, true
	case r >= 'A' && r <= 'Z':
		return byte(r-'A') + 1, true
	case r == '[':
		return 0x1b, true
	case r == '\\':
		return 0x1c, true
	case r == ']':
		return 0x1d, true
	case r == '^':
		return 0x1e, true
	case r == '_':
		return 0x1f, true
	}
	return 0, false
}

// bracketPaste wraps s in the bracketed-paste markers if enabled, so the
// receiving application (a shell/readline/editor that opted in via DECSET
// 2004) can tell pasted text apart from typed text — e.g. to avoid
// auto-indent mangling a pasted code block.
func bracketPaste(s string, enabled bool) []byte {
	if !enabled {
		return []byte(s)
	}
	return append(append([]byte("\x1b[200~"), s...), "\x1b[201~"...)
}
