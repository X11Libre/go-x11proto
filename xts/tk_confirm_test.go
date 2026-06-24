package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/dialog"
	"github.com/X11Libre/go-x11proto/tk/font"
	"github.com/X11Libre/go-x11proto/tk/keyboard"
)

// keycodeFor scans the server keymap for a keycode whose logical key matches.
func keycodeFor(km *keyboard.Map, minkc, maxkc int, want keyboard.Key) (base.CARD8, bool) {
	for kc := minkc; kc <= maxkc; kc++ {
		if km.Lookup(base.CARD8(kc), 0).Key == want {
			return base.CARD8(kc), true
		}
	}
	return 0, false
}

// TestTkConfirm drives the Confirm dialog with synthetic Enter / Escape key
// presses and checks the right callback fires.
func TestTkConfirm(t *testing.T) {
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)
	tkp := &tk

	f, err := font.Open(c, "fixed")
	must(t, err, "font.Open")
	defer f.Close(c)

	km, err := keyboard.Load(c)
	must(t, err, "keyboard.Load")
	enter, ok := keycodeFor(km, int(c.Setup.MinKeycode), int(c.Setup.MaxKeycode), keyboard.KeyEnter)
	if !ok {
		t.Skip("no Enter key in this layout")
	}
	esc, ok := keycodeFor(km, int(c.Setup.MinKeycode), int(c.Setup.MaxKeycode), keyboard.KeyEscape)
	if !ok {
		t.Skip("no Escape key in this layout")
	}

	newDlg := func() (*dialog.Confirm, *string) {
		got := new(string)
		d := &dialog.Confirm{
			Window:  tk_core.Window{Drawable: tk_core.Drawable{Conn: tkp}, X: 0, Y: 0, W: 300, H: 90},
			Font:    f,
			Keymap:  km,
			Message: "Discard changes?",
			OnYes:   func() { *got = "yes" },
			OnNo:    func() { *got = "no" },
		}
		must(t, d.Init(), "Confirm.Init")
		return d, got
	}

	key := func(kc base.CARD8) *events.KeyPressEvent {
		e := &events.KeyPressEvent{}
		e.Key = kc
		return e
	}

	d1, got1 := newDlg()
	d1.HandleWindowEvent(key(enter))
	if *got1 != "yes" {
		t.Errorf("Enter -> %q, want yes", *got1)
	}
	must(t, d1.Destroy(), "destroy d1")

	d2, got2 := newDlg()
	d2.HandleWindowEvent(key(esc))
	if *got2 != "no" {
		t.Errorf("Escape -> %q, want no", *got2)
	}
	must(t, d2.Destroy(), "destroy d2")
}
