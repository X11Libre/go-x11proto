// Package keycodes provides named constants for the physical X11 keycodes
// reported in the detail byte of KeyPress/KeyRelease events, for the standard
// XKB/evdev base layout (keycode = Linux evdev code + 8).
package keycodes

import "github.com/X11Libre/go-x11proto/proto/base"

const (
	Q      base.CARD8 = 24
	C      base.CARD8 = 54
	F      base.CARD8 = 41
	G      base.CARD8 = 42
	F1     base.CARD8 = 67
	Space  base.CARD8 = 65
	Return base.CARD8 = 36

	Up    base.CARD8 = 111
	Down  base.CARD8 = 116
	Left  base.CARD8 = 113
	Right base.CARD8 = 114

	// vi-style movement keys
	H base.CARD8 = 43
	J base.CARD8 = 44
	K base.CARD8 = 45
	L base.CARD8 = 46

	// "+" and "-" sit at different physical positions across keyboard layouts,
	// so each has several keycodes worth accepting.
	PlusMain  base.CARD8 = 35 // main-row "+" (DE "]")
	PlusKP    base.CARD8 = 86 // keypad "+"
	PlusEqual base.CARD8 = 21 // US "=/+"
	MinusMain base.CARD8 = 61 // main-row "-" (DE "/")
	MinusKP   base.CARD8 = 82 // keypad "-"
	MinusUS   base.CARD8 = 20 // US "-/_"
)

// Modifier masks for the State field of KeyPress/KeyRelease events.
const (
	ShiftMask base.CARD16 = 1 << 0
)
