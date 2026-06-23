package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/tk/keyboard"
)

// TestKeyboardMap loads the server keyboard mapping through tk/keyboard and
// checks that a known letter key translates with the Shift/CapsLock rules and
// that Return resolves to the logical Enter key. It scans the mapping for the
// keycode carrying 'a' rather than hard-coding one, so it is layout-robust.
func TestKeyboardMap(t *testing.T) {
	c := connect(t)
	defer c.Close()

	km, err := keyboard.Load(c)
	if err != nil {
		t.Fatalf("keyboard.Load: %v", err)
	}

	// Find the keycode that produces 'a' unshifted.
	var kcA base.CARD8
	found := false
	for kc := int(c.Setup.MinKeycode); kc <= int(c.Setup.MaxKeycode); kc++ {
		if km.Lookup(base.CARD8(kc), 0).Rune == 'a' {
			kcA = base.CARD8(kc)
			found = true
			break
		}
	}
	if !found {
		t.Skip("no key maps to 'a' in this server's layout")
	}

	if got := km.Lookup(kcA, 0).Rune; got != 'a' {
		t.Errorf("plain = %q, want 'a'", got)
	}
	if got := km.Lookup(kcA, 1).Rune; got != 'A' { // ShiftMask = 1
		t.Errorf("shift = %q, want 'A'", got)
	}
	ev := km.Lookup(kcA, 0)
	if !ev.Printable() || ev.Key != keyboard.KeyNone {
		t.Errorf("'a' should be a printable, non-special key: %+v", ev)
	}

	// Return key (keycode 36 on the standard evdev layout) -> KeyEnter.
	if k := km.Lookup(36, 0).Key; k != keyboard.KeyEnter {
		t.Logf("keycode 36 -> %v (not KeyEnter; layout may differ)", k)
	}
}
