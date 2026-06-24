package widget

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

// rect is a child placement in frame-local coordinates.
type rect struct{ X, Y, W, H int }

// Slot is a child window pinned to a frame edge with a fixed extent (height for
// Top/Bottom, width for Left/Right).
type Slot struct {
	Win    *tk_core.Window
	Extent int
}

// Frame is a border layout: optional Top/Bottom bars span the full width, then
// optional Left/Right bars take the remaining height, and Center fills the
// rest. It re-lays its children whenever it is resized (ConfigureNotify), which
// is exactly the arrangement an editor needs — menubar on top, status line at
// the bottom, scrollbar on the right, text area filling the middle.
//
// Children must already be created (their windows reparented to this frame).
// Fill in the embedded Window before Init.
type Frame struct {
	tk_core.Window
	Top, Bottom *Slot
	Left, Right *Slot
	Center      *tk_core.Window
	OnLayout    func()
	// OnClose, if set, is called when the window manager asks to close the frame
	// (the title-bar close button). Without it the WM would kill the connection.
	OnClose func()

	wmDelete base.ATOM
}

// Init creates and maps the frame and performs the first layout.
func (f *Frame) Init() error {
	f.EventMask |= base.CARD32(event_mask.StructureNotify)
	f.Window.SetWindowHandler(f)
	if err := f.Window.Create(); err != nil {
		return err
	}
	f.wmDelete, _ = f.Window.EnableWMDelete()
	if err := f.Window.Map(); err != nil {
		return err
	}
	f.Relayout(int(f.W), int(f.H))
	return nil
}

// HandleWindowEvent re-lays the children on resize and handles the WM close
// request.
func (f *Frame) HandleWindowEvent(ev events.Event) bool {
	if tk_core.IsWMDelete(ev, f.wmDelete) {
		if f.OnClose != nil {
			f.OnClose()
		}
		return true
	}
	if e, ok := ev.(*events.ConfigureEvent); ok {
		f.W, f.H = e.Width, e.Height
		f.Relayout(int(e.Width), int(e.Height))
	}
	return true
}

// Relayout positions every present child for a frame of size w×h.
func (f *Frame) Relayout(w, h int) {
	top, bottom, left, right, center := computeBorderLayout(w, h,
		extent(f.Top), extent(f.Bottom), extent(f.Left), extent(f.Right))

	place(slotWin(f.Top), top)
	place(slotWin(f.Bottom), bottom)
	place(slotWin(f.Left), left)
	place(slotWin(f.Right), right)
	place(f.Center, center)

	if f.OnLayout != nil {
		f.OnLayout()
	}
}

func extent(s *Slot) int {
	if s == nil || s.Win == nil {
		return 0
	}
	return s.Extent
}

func slotWin(s *Slot) *tk_core.Window {
	if s == nil {
		return nil
	}
	return s.Win
}

func place(w *tk_core.Window, r rect) {
	if w == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	_ = w.MoveResize(base.INT16(r.X), base.INT16(r.Y), base.CARD16(r.W), base.CARD16(r.H))
}

// computeBorderLayout slices a w×h area into the five border regions. Top and
// bottom span the full width; left and right take the height between them;
// center fills what remains. Zero extents drop that region (zero-size rect).
func computeBorderLayout(w, h, topH, bottomH, leftW, rightW int) (top, bottom, left, right, center rect) {
	if topH > h {
		topH = h
	}
	if bottomH > h-topH {
		bottomH = h - topH
	}
	midY := topH
	midH := h - topH - bottomH
	if midH < 0 {
		midH = 0
	}

	top = rect{0, 0, w, topH}
	bottom = rect{0, h - bottomH, w, bottomH}

	if leftW > w {
		leftW = w
	}
	if rightW > w-leftW {
		rightW = w - leftW
	}
	centerW := w - leftW - rightW
	if centerW < 0 {
		centerW = 0
	}

	left = rect{0, midY, leftW, midH}
	right = rect{w - rightW, midY, rightW, midH}
	center = rect{leftW, midY, centerW, midH}
	return top, bottom, left, right, center
}
