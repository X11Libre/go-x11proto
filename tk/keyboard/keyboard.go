// Package keyboard turns the raw keycodes carried by KeyPress/KeyRelease events
// into keysyms and Unicode runes, applying the X11 Shift/CapsLock case rules.
// It loads the server's keyboard mapping once (GetKeyboardMapping over the
// setup's keycode range) and translates events against it, so a toolkit can do
// text input without hard-coding a layout.
package keyboard

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// Modifier masks as they appear in the State field of key events.
const (
	maskShift   base.CARD16 = 1 << 0
	maskLock    base.CARD16 = 1 << 1 // CapsLock
	maskControl base.CARD16 = 1 << 2
	maskMod1    base.CARD16 = 1 << 3 // usually Alt
)

// Event is a decoded key press: the resolved keysym plus, for convenience, the
// printable rune (0 if none) and a logical Key for editing/navigation keys.
type Event struct {
	Keycode base.CARD8
	Keysym  uint32
	Rune    rune // printable character, or 0
	Key     Key  // logical editing key, or KeyNone
	Shift   bool
	Ctrl    bool
	Alt     bool
}

// Printable reports whether the event should insert text: it has a rune, is not
// a recognised editing key, and no Control/Alt modifier is held.
func (e Event) Printable() bool {
	return e.Rune != 0 && e.Key == KeyNone && !e.Ctrl && !e.Alt
}

// Map is a snapshot of the server keyboard mapping.
type Map struct {
	minKeycode base.CARD8
	perCode    int
	keysyms    []base.CARD32
}

// Load fetches the keyboard mapping for the whole keycode range advertised in
// the connection setup.
func Load(conn *core.X11Conn) (*Map, error) {
	first := conn.Setup.MinKeycode
	count := int(conn.Setup.MaxKeycode) - int(first) + 1
	if count <= 0 {
		count = 1
	}
	rep, err := rpc.GetKeyboardMapping(conn, first, base.CARD8(count))
	if err != nil {
		return nil, err
	}
	return &Map{
		minKeycode: first,
		perCode:    int(rep.KeysymsPerKeycode),
		keysyms:    rep.Keysyms,
	}, nil
}

// group returns the (unshifted, shifted) keysym pair for a keycode, resolving a
// missing shifted symbol per the core protocol: for a single alphabetic symbol
// the pair is (lower, upper); otherwise the shifted symbol mirrors the
// unshifted one.
func (m *Map) group(keycode base.CARD8) (k0, k1 uint32) {
	if m.perCode == 0 || keycode < m.minKeycode {
		return xkNoSymbol, xkNoSymbol
	}
	base0 := (int(keycode) - int(m.minKeycode)) * m.perCode
	if base0 < 0 || base0 >= len(m.keysyms) {
		return xkNoSymbol, xkNoSymbol
	}
	k0 = uint32(m.keysyms[base0])
	if m.perCode > 1 {
		k1 = uint32(m.keysyms[base0+1])
	}
	if k1 == xkNoSymbol {
		if keysymIsLetter(k0) {
			return lowerKeysym(k0), upperKeysym(k0)
		}
		return k0, k0
	}
	return k0, k1
}

// Lookup translates a keycode and modifier state into an Event.
func (m *Map) Lookup(keycode base.CARD8, state base.CARD16) Event {
	k0, k1 := m.group(keycode)
	shift := state&maskShift != 0
	lock := state&maskLock != 0

	var ks uint32
	switch {
	case !shift && !lock:
		ks = k0
	case !shift && lock:
		// CapsLock affects letters only.
		if keysymIsLetter(k0) {
			ks = k1
		} else {
			ks = k0
		}
	case shift && !lock:
		ks = k1
	default: // shift && lock: for letters the two cancel out
		if keysymIsLetter(k0) {
			ks = k0
		} else {
			ks = k1
		}
	}

	ev := Event{
		Keycode: keycode,
		Keysym:  ks,
		Key:     specialKey(ks),
		Shift:   shift,
		Ctrl:    state&maskControl != 0,
		Alt:     state&maskMod1 != 0,
	}
	if ev.Key == KeyNone {
		ev.Rune = keysymToRune(ks)
	}
	return ev
}
