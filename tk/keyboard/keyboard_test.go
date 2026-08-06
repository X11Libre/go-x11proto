package keyboard

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
)

// synthMap builds a Map from a keycode->keysyms table (2 keysyms per code),
// mimicking what GetKeyboardMapping returns, so the translation rules can be
// tested offline.
func synthMap(t *testing.T, perCode int, table map[base.CARD8][]uint32) *Map {
	t.Helper()
	const min = base.CARD8(8)
	maxKc := min
	for kc := range table {
		if kc > maxKc {
			maxKc = kc
		}
	}
	n := int(maxKc) - int(min) + 1
	syms := make([]base.CARD32, n*perCode)
	for kc, ks := range table {
		off := (int(kc) - int(min)) * perCode
		for i := 0; i < perCode && i < len(ks); i++ {
			syms[off+i] = base.CARD32(ks[i])
		}
	}
	return &Map{minKeycode: min, perCode: perCode, keysyms: syms}
}

func TestKeysymToRune(t *testing.T) {
	cases := []struct {
		ks   uint32
		want rune
	}{
		{0x61, 'a'},       // Latin-1 lower
		{0x41, 'A'},       // Latin-1 upper
		{0x20, ' '},       // space
		{0x7e, '~'},       // tilde
		{0xe4, 'ä'},       // Latin-1 high
		{0x01000100, 'Ā'}, // direct Unicode block (U+0100)
		{xkReturn, 0},     // function key, no rune
		{xkBackSpace, 0},  // function key, no rune
		{xkNoSymbol, 0},   // nothing
	}
	for _, c := range cases {
		if got := keysymToRune(c.ks); got != c.want {
			t.Errorf("keysymToRune(%#x) = %q, want %q", c.ks, got, c.want)
		}
	}
}

func TestLookupLetterCaseRules(t *testing.T) {
	const kcA = base.CARD8(38) // 'a'
	m := synthMap(t, 2, map[base.CARD8][]uint32{
		kcA: {0x61, 0x41}, // a / A
	})
	cases := []struct {
		name  string
		state base.CARD16
		want  rune
	}{
		{"plain", 0, 'a'},
		{"shift", maskShift, 'A'},
		{"caps", maskLock, 'A'},
		{"shift+caps", maskShift | maskLock, 'a'}, // cancel out for letters
	}
	for _, c := range cases {
		ev := m.Lookup(kcA, c.state)
		if ev.Rune != c.want {
			t.Errorf("%s: rune = %q, want %q", c.name, ev.Rune, c.want)
		}
		if ev.Key != KeyNone {
			t.Errorf("%s: Key = %v, want KeyNone", c.name, ev.Key)
		}
	}
}

func TestLookupDigitCaseRules(t *testing.T) {
	const kc1 = base.CARD8(10) // 1 / !
	m := synthMap(t, 2, map[base.CARD8][]uint32{
		kc1: {0x31, 0x21}, // '1' / '!'
	})
	cases := []struct {
		state base.CARD16
		want  rune
	}{
		{0, '1'},
		{maskShift, '!'},
		{maskLock, '1'},             // CapsLock must not affect non-letters
		{maskShift | maskLock, '!'}, // shift still shifts
	}
	for _, c := range cases {
		if got := m.Lookup(kc1, c.state).Rune; got != c.want {
			t.Errorf("state=%d: rune = %q, want %q", c.state, got, c.want)
		}
	}
}

func TestLookupMissingShiftedLetter(t *testing.T) {
	// Only one keysym given for a letter: shifted form must be derived.
	const kcA = base.CARD8(38)
	m := synthMap(t, 2, map[base.CARD8][]uint32{
		kcA: {0x61}, // only 'a'
	})
	if got := m.Lookup(kcA, maskShift).Rune; got != 'A' {
		t.Errorf("derived shift = %q, want 'A'", got)
	}
	if got := m.Lookup(kcA, 0).Rune; got != 'a' {
		t.Errorf("plain = %q, want 'a'", got)
	}
}

func TestLookupSpecialKeys(t *testing.T) {
	tbl := map[base.CARD8][]uint32{
		36:  {xkReturn},
		22:  {xkBackSpace},
		119: {xkDelete},
		113: {xkLeft},
		114: {xkRight},
		110: {xkHome},
		115: {xkEnd},
		106: {xkInsert},
		67:  {xkF1},
		76:  {xkF12},
	}
	m := synthMap(t, 1, tbl)
	want := map[base.CARD8]Key{
		36: KeyEnter, 22: KeyBackspace, 119: KeyDelete,
		113: KeyLeft, 114: KeyRight, 110: KeyHome, 115: KeyEnd,
		106: KeyInsert, 67: KeyF1, 76: KeyF12,
	}
	for kc, k := range want {
		ev := m.Lookup(kc, 0)
		if ev.Key != k {
			t.Errorf("keycode %d: Key = %v, want %v", kc, ev.Key, k)
		}
		if ev.Rune != 0 {
			t.Errorf("keycode %d: special key produced rune %q", kc, ev.Rune)
		}
		if ev.Printable() {
			t.Errorf("keycode %d: special key reported Printable", kc)
		}
	}
}

func TestPrintableWithModifiers(t *testing.T) {
	const kcA = base.CARD8(38)
	m := synthMap(t, 2, map[base.CARD8][]uint32{kcA: {0x61, 0x41}})
	if !m.Lookup(kcA, 0).Printable() {
		t.Error("plain 'a' should be printable")
	}
	if m.Lookup(kcA, maskControl).Printable() {
		t.Error("Ctrl+a must not be printable")
	}
	if ev := m.Lookup(kcA, maskControl); !ev.Ctrl || ev.Keysym != 0x61 {
		t.Errorf("Ctrl+a: Ctrl=%v keysym=%#x, want Ctrl=true keysym=0x61", ev.Ctrl, ev.Keysym)
	}
}

func TestLookupPasteShortcuts(t *testing.T) {
	// The keyboard-paste detection in tk/term handleKey relies on these
	// exact lookup results: Ctrl+V must surface as rune 'v' with Ctrl set
	// (so it can be intercepted before EncodeKey turns it into 0x16), and
	// Shift+Insert must surface as KeyInsert with Shift set (so it can be
	// intercepted before EncodeKey swallows it).
	const kcV = base.CARD8(55)
	m := synthMap(t, 2, map[base.CARD8][]uint32{
		kcV: {0x76, 0x56}, // 'v' / 'V'
		106: {xkInsert},
	})
	ev := m.Lookup(kcV, maskControl)
	if ev.Rune != 'v' || !ev.Ctrl {
		t.Errorf("Ctrl+V: rune=%q Ctrl=%v, want rune 'v' Ctrl=true", ev.Rune, ev.Ctrl)
	}
	ev = m.Lookup(kcV, maskControl|maskShift)
	if ev.Rune != 'V' || !ev.Ctrl || !ev.Shift {
		t.Errorf("Ctrl+Shift+V: rune=%q Ctrl=%v Shift=%v, want rune 'V' Ctrl+Shift=true", ev.Rune, ev.Ctrl, ev.Shift)
	}
	ev = m.Lookup(106, maskShift)
	if ev.Key != KeyInsert || !ev.Shift {
		t.Errorf("Shift+Insert: Key=%v Shift=%v, want KeyInsert Shift=true", ev.Key, ev.Shift)
	}
}
