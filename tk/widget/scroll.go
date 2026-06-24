package widget

import "github.com/X11Libre/go-x11proto/proto/base"

// X11 delivers wheel and touchpad two-finger scrolling as button events:
// buttons 4/5 are vertical (up/down), 6/7 horizontal. Widgets treat 4/5 as a
// scroll gesture.
const (
	btnWheelUp    base.CARD8 = 4
	btnWheelDown  base.CARD8 = 5
	btnWheelLeft  base.CARD8 = 6
	btnWheelRight base.CARD8 = 7
)

// wheelStepLines is how many lines one wheel/touchpad notch scrolls.
const wheelStepLines = 3
