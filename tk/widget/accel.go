package widget

import (
	"strings"

	"github.com/X11Libre/go-x11proto/tk/keyboard"
)

// accel is a parsed menu accelerator: a set of required modifiers plus a base
// keysym (letters normalised to lower case).
type accel struct {
	ctrl, shift, alt bool
	keysym           uint32
}

// parseAccel parses a "Ctrl+Shift+S"-style accelerator. Recognised modifier
// words (case-insensitive): ctrl/control, shift, alt/meta. The remaining token
// is the key: a single character (letter/digit/punctuation). It returns ok=false
// for an unparseable or key-less spec.
func parseAccel(s string) (accel, bool) {
	var a accel
	for _, part := range strings.Split(s, "+") {
		p := strings.TrimSpace(part)
		switch strings.ToLower(p) {
		case "ctrl", "control":
			a.ctrl = true
		case "shift":
			a.shift = true
		case "alt", "meta":
			a.alt = true
		default:
			r := []rune(p)
			if len(r) != 1 {
				return accel{}, false // unsupported key token (e.g. "F1")
			}
			a.keysym = normLetter(uint32(r[0]))
		}
	}
	if a.keysym == 0 {
		return accel{}, false
	}
	return a, true
}

// match reports whether key event k triggers this accelerator.
func (a accel) match(k keyboard.Event) bool {
	return a.ctrl == k.Ctrl && a.shift == k.Shift && a.alt == k.Alt &&
		a.keysym == normLetter(k.Keysym)
}

// normLetter folds an ASCII upper-case letter keysym to lower case so an
// accelerator matches regardless of the Shift-folded keysym the server reports.
func normLetter(ks uint32) uint32 {
	if ks >= 'A' && ks <= 'Z' {
		return ks + ('a' - 'A')
	}
	return ks
}

// findAccel walks an item tree (descending into submenus) and returns the
// OnClick of the first item whose accelerator matches k, or nil.
func findAccel(items []MenuItem, k keyboard.Event) func() {
	for i := range items {
		it := &items[i]
		if it.Separator {
			continue
		}
		if it.Submenu != nil {
			if f := findAccel(it.Submenu, k); f != nil {
				return f
			}
			continue
		}
		if it.Accel == "" || it.OnClick == nil {
			continue
		}
		if a, ok := parseAccel(it.Accel); ok && a.match(k) {
			return it.OnClick
		}
	}
	return nil
}

// HandleKey fires the OnClick of the menu item whose accelerator matches k
// (searching the menu and its submenus) and reports whether one matched. Wire
// it from a focused widget's key handler, e.g. TextView.OnKey.
func (m *Menu) HandleKey(k keyboard.Event) bool {
	if f := findAccel(m.Items, k); f != nil {
		f()
		return true
	}
	return false
}

// HandleKey fires the matching accelerator across all of the bar's menus.
func (b *MenuBar) HandleKey(k keyboard.Event) bool {
	for _, en := range b.entries {
		if f := findAccel(en.menu.Items, k); f != nil {
			f()
			return true
		}
	}
	return false
}
