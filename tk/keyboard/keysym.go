package keyboard

import "unicode"

// X11 keysyms relevant to text editing. The function-key block lives in
// 0xff00-0xffff; printable Latin-1 keysyms equal their Unicode code point, and
// the 0x01000000 block encodes Unicode directly (keysym - 0x01000000).
const (
	xkNoSymbol  = 0x000000
	xkBackSpace = 0xff08
	xkTab       = 0xff09
	xkReturn    = 0xff0d
	xkEscape    = 0xff1b
	xkHome      = 0xff50
	xkLeft      = 0xff51
	xkUp        = 0xff52
	xkRight     = 0xff53
	xkDown      = 0xff54
	xkPageUp    = 0xff55
	xkPageDown  = 0xff56
	xkEnd       = 0xff57
	xkKPEnter   = 0xff8d
	xkDelete    = 0xffff

	// xkF1 (0xffbe) through xkF12 (0xffc9) are contiguous, so specialKey
	// below maps the whole range with one arithmetic check instead of 12
	// cases.
	xkF1  = 0xffbe
	xkF12 = 0xffc9
)

// Key is a logical editing/navigation key, decoupled from the raw keysym.
type Key int

const (
	KeyNone Key = iota
	KeyBackspace
	KeyDelete
	KeyEnter
	KeyTab
	KeyEscape
	KeyLeft
	KeyRight
	KeyUp
	KeyDown
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)

// specialKey maps a keysym to a logical editing key, or KeyNone if the keysym
// is not one (e.g. a printable character).
func specialKey(ks uint32) Key {
	switch ks {
	case xkBackSpace:
		return KeyBackspace
	case xkDelete:
		return KeyDelete
	case xkReturn, xkKPEnter:
		return KeyEnter
	case xkTab:
		return KeyTab
	case xkEscape:
		return KeyEscape
	case xkLeft:
		return KeyLeft
	case xkRight:
		return KeyRight
	case xkUp:
		return KeyUp
	case xkDown:
		return KeyDown
	case xkHome:
		return KeyHome
	case xkEnd:
		return KeyEnd
	case xkPageUp:
		return KeyPageUp
	case xkPageDown:
		return KeyPageDown
	}
	if ks >= xkF1 && ks <= xkF12 {
		return KeyF1 + Key(ks-xkF1)
	}
	return KeyNone
}

// keysymToRune converts a keysym to the Unicode rune it represents, or 0 when it
// has no printable character (function keys, modifiers, unmapped symbols).
func keysymToRune(ks uint32) rune {
	// Latin-1: keysyms 0x20-0x7e and 0xa0-0xff are the code point itself.
	if (ks >= 0x20 && ks <= 0x7e) || (ks >= 0xa0 && ks <= 0xff) {
		return rune(ks)
	}
	// Direct Unicode block.
	if ks >= 0x01000000 && ks <= 0x0110ffff {
		return rune(ks - 0x01000000)
	}
	return 0
}

// keysymIsLetter reports whether the keysym maps to an alphabetic rune (used for
// the CapsLock case rule, which only applies to letters).
func keysymIsLetter(ks uint32) bool {
	r := keysymToRune(ks)
	return r != 0 && unicode.IsLetter(r)
}

// upperKeysym / lowerKeysym return the case-folded keysym for a letter keysym,
// or the keysym unchanged when it is not a letter.
func upperKeysym(ks uint32) uint32 {
	if r := keysymToRune(ks); r != 0 {
		if u := unicode.ToUpper(r); u != r {
			return runeToKeysym(u)
		}
	}
	return ks
}

func lowerKeysym(ks uint32) uint32 {
	if r := keysymToRune(ks); r != 0 {
		if l := unicode.ToLower(r); l != r {
			return runeToKeysym(l)
		}
	}
	return ks
}

// runeToKeysym is the inverse of keysymToRune for the ranges we care about.
func runeToKeysym(r rune) uint32 {
	if (r >= 0x20 && r <= 0x7e) || (r >= 0xa0 && r <= 0xff) {
		return uint32(r)
	}
	return uint32(r) + 0x01000000
}
