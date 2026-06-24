package widget

import (
	"testing"

	"github.com/X11Libre/go-x11proto/tk/keyboard"
)

func TestParseAccel(t *testing.T) {
	cases := []struct {
		in              string
		ok              bool
		ctrl, shift, al bool
		ks              uint32
	}{
		{"Ctrl+O", true, true, false, false, 'o'},
		{"ctrl+o", true, true, false, false, 'o'},
		{"Ctrl+Shift+S", true, true, true, false, 's'},
		{"Alt+X", true, false, false, true, 'x'},
		{"Ctrl+Alt+Shift+a", true, true, true, true, 'a'},
		{"S", true, false, false, false, 's'}, // bare key, no modifiers
		{"Ctrl+", false, false, false, false, 0},
		{"Ctrl+F1", false, false, false, false, 0}, // multi-char key unsupported
		{"", false, false, false, false, 0},
	}
	for _, c := range cases {
		a, ok := parseAccel(c.in)
		if ok != c.ok {
			t.Errorf("parseAccel(%q) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if a.ctrl != c.ctrl || a.shift != c.shift || a.alt != c.al || a.keysym != c.ks {
			t.Errorf("parseAccel(%q) = %+v, want ctrl=%v shift=%v alt=%v ks=%q",
				c.in, a, c.ctrl, c.shift, c.al, c.ks)
		}
	}
}

func TestAccelMatch(t *testing.T) {
	ctrlO, _ := parseAccel("Ctrl+O")
	ctrlShiftS, _ := parseAccel("Ctrl+Shift+S")

	// Ctrl+O event: keysym 'o', ctrl only.
	if !ctrlO.match(keyboard.Event{Keysym: 'o', Ctrl: true}) {
		t.Error("Ctrl+O should match keysym 'o' + Ctrl")
	}
	// must not match when Shift is also held.
	if ctrlO.match(keyboard.Event{Keysym: 'o', Ctrl: true, Shift: true}) {
		t.Error("Ctrl+O must not match when Shift is held")
	}
	// must not match without Ctrl.
	if ctrlO.match(keyboard.Event{Keysym: 'o'}) {
		t.Error("Ctrl+O must not match without Ctrl")
	}
	// Ctrl+Shift+S: the server reports the Shift-folded upper-case keysym 'S'.
	if !ctrlShiftS.match(keyboard.Event{Keysym: 'S', Ctrl: true, Shift: true}) {
		t.Error("Ctrl+Shift+S should match keysym 'S' + Ctrl + Shift")
	}
	if ctrlShiftS.match(keyboard.Event{Keysym: 's', Ctrl: true}) {
		t.Error("Ctrl+Shift+S must not match plain Ctrl+s")
	}
}

func TestFindAccelDescendsSubmenus(t *testing.T) {
	hit := ""
	items := []MenuItem{
		{Label: "Open", Accel: "Ctrl+O", OnClick: func() { hit = "open" }},
		{Separator: true},
		{Label: "More", Submenu: []MenuItem{
			{Label: "Save As", Accel: "Ctrl+Shift+S", OnClick: func() { hit = "saveas" }},
		}},
	}
	if f := findAccel(items, keyboard.Event{Keysym: 'o', Ctrl: true}); f != nil {
		f()
	}
	if hit != "open" {
		t.Errorf("Ctrl+O -> %q, want open", hit)
	}
	hit = ""
	if f := findAccel(items, keyboard.Event{Keysym: 'S', Ctrl: true, Shift: true}); f != nil {
		f()
	}
	if hit != "saveas" {
		t.Errorf("Ctrl+Shift+S -> %q, want saveas", hit)
	}
	// no match leaves things untouched
	hit = ""
	if f := findAccel(items, keyboard.Event{Keysym: 'z', Ctrl: true}); f != nil {
		t.Error("Ctrl+Z should not match any item")
	}
}

func TestMenuBarHandleKey(t *testing.T) {
	var b MenuBar
	saved := false
	b.AddMenu("File", []MenuItem{
		{Label: "Save", Accel: "Ctrl+S", OnClick: func() { saved = true }},
	})
	if !b.HandleKey(keyboard.Event{Keysym: 's', Ctrl: true}) {
		t.Error("HandleKey should report Ctrl+S handled")
	}
	if !saved {
		t.Error("Ctrl+S should have fired Save")
	}
	if b.HandleKey(keyboard.Event{Keysym: 'q', Ctrl: true}) {
		t.Error("unbound Ctrl+Q should not be handled")
	}
}

// menuItemTextWidths checks the layout reserves room for the accelerator column.
func TestMenuLayoutWidthWithAccel(t *testing.T) {
	withAccel := &Menu{Items: []MenuItem{{Label: "Open", Accel: "Ctrl+O"}}}
	withAccel.layout()
	noAccel := &Menu{Items: []MenuItem{{Label: "Open"}}}
	noAccel.layout()
	if withAccel.W <= noAccel.W {
		t.Errorf("accel menu width %d should exceed plain width %d", withAccel.W, noAccel.W)
	}
}
