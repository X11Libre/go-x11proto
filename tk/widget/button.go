package widget

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font"
)

// Button is a labelled push button with a 1px border: the label is centred and
// the button draws inverted (white-on-black) while pressed. OnButtonPress fires
// on release inside the button.
//
// Fill the embedded Window (Parent/X/Y/W/H) and Label before Init; if Label is
// empty the Window's Name is used. Font is optional — a "fixed" font is opened
// when it is nil.
type Button struct {
	tk_core.Window
	Label         string
	Font          *font.Font
	OnButtonPress func()

	gc      *tk_core.GC // black on white (normal text, pressed fill)
	gcHi    *tk_core.GC // white on black (pressed text)
	ownFont bool
	down    bool
}

// Init creates and maps the button, opening a default font if none was set.
func (w *Button) Init() error {
	w.EventMask |= event_mask.ButtonPress | event_mask.ButtonRelease | event_mask.Exposure
	w.SetBorderPixel = true
	w.BorderPixel = w.Conn.X11Conn.DefaultBlackPixel()
	if w.BorderWidth == 0 {
		w.BorderWidth = 1
	}

	w.Window.SetWindowHandler(w)
	if err := w.Window.Create(); err != nil {
		return err
	}

	if w.Font == nil {
		f, err := font.Open(w.Conn.X11Conn, "fixed")
		if err != nil {
			return err
		}
		w.Font, w.ownFont = f, true
	}

	black := w.Conn.X11Conn.DefaultBlackPixel()
	white := w.Conn.X11Conn.DefaultWhitePixel()
	var err error
	if w.gc, err = w.Conn.CreateGC1(black, white, w.Font.ID); err != nil {
		return err
	}
	if w.gcHi, err = w.Conn.CreateGC1(white, black, w.Font.ID); err != nil {
		return err
	}
	return w.Window.Map()
}

func (w *Button) text() string {
	if w.Label != "" {
		return w.Label
	}
	return w.Window.Name
}

// Repaint draws the button in its current (up/down) state.
func (w *Button) Repaint() {
	if w.gc == nil {
		return
	}
	_ = w.ClearArea(0, 0, 0, 0, false)
	label := w.text()
	tw, th := w.Font.Measure(0, label)
	x := (int(w.W) - tw) / 2
	y := (int(w.H) - th) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	text := w.gc
	if w.down {
		_ = w.FillRect(w.gc.XID, 0, 0, w.W, w.H) // black fill
		text = w.gcHi
	}
	_ = w.Font.DrawText(w.Drawable, text.XID, base.INT16(x), base.INT16(y), 0, label)
}

func (w *Button) HandleWindowEvent(ev events.Event) bool {
	switch ev.(type) {
	case *events.ExposeEvent:
		w.Repaint()
	case *events.ButtonPressEvent:
		w.down = true
		w.Repaint()
	case *events.ButtonReleaseEvent:
		w.down = false
		w.Repaint()
		if w.OnButtonPress != nil {
			w.OnButtonPress()
		}
	}
	return true
}
