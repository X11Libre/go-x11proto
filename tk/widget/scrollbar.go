package widget

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

const sbMinThumb = 8 // minimum thumb height in pixels

// Scrollbar is a vertical scrollbar: a track with a draggable thumb whose size
// and position reflect a (Total, Visible, Top) line range. Clicking the track
// pages, dragging the thumb scrolls continuously; either way OnScroll reports
// the new top line. Bind it to a TextView by forwarding OnScroll to
// TextView.ScrollTo and refreshing the range from TextView.OnScroll.
type Scrollbar struct {
	tk_core.Window
	ThumbColor base.CARD32 // defaults to black
	OnScroll   func(top int)

	gc       *tk_core.GC
	total    int
	visible  int
	top      int
	dragging bool
	dragOff  int // pointer offset inside the thumb at drag start
}

// Init creates and maps the scrollbar.
func (s *Scrollbar) Init() error {
	if s.ThumbColor == 0 {
		s.ThumbColor = s.Conn.X11Conn.DefaultBlackPixel()
	}
	s.EventMask |= base.CARD32(event_mask.ButtonPress | event_mask.ButtonRelease |
		event_mask.Button1Motion | event_mask.Exposure | event_mask.StructureNotify)
	if !s.SetBackPixel {
		s.BackPixel = 0xc0c0c0 // light grey track
		s.SetBackPixel = true
	}
	s.Window.SetWindowHandler(s)
	if err := s.Window.Create(); err != nil {
		return err
	}
	gc, err := s.Conn.CreateGC1(s.ThumbColor, s.BackPixel, 0)
	if err != nil {
		return err
	}
	s.gc = gc
	return s.Window.Map()
}

// SetRange updates the line range and repaints. total is the number of lines,
// visible the number on screen, top the first visible line.
func (s *Scrollbar) SetRange(total, visible, top int) {
	s.total, s.visible, s.top = total, visible, top
	_ = s.Draw()
}

// Draw paints the track and thumb.
func (s *Scrollbar) Draw() error {
	if err := s.ClearArea(0, 0, 0, 0, false); err != nil {
		return err
	}
	y, h := thumbGeom(int(s.H), s.total, s.visible, s.top)
	return s.FillRect(s.gc.XID, 0, base.INT16(y), s.W, base.CARD16(h))
}

// HandleWindowEvent handles paging, dragging and repaint.
func (s *Scrollbar) HandleWindowEvent(ev events.Event) bool {
	switch e := ev.(type) {
	case *events.ExposeEvent:
		_ = s.Draw()
	case *events.ConfigureEvent:
		s.W, s.H = e.Width, e.Height
		_ = s.Draw()
	case *events.ButtonPressEvent:
		switch e.Key {
		case btnWheelUp: // touchpad / wheel scroll up
			s.scrollTo(s.top - wheelStepLines)
		case btnWheelDown:
			s.scrollTo(s.top + wheelStepLines)
		default:
			s.press(int(e.EventY))
		}
	case *events.ButtonReleaseEvent:
		s.dragging = false
	case *events.MotionEvent:
		if s.dragging {
			s.dragTo(int(e.EventY))
		}
	}
	return true
}

func (s *Scrollbar) press(y int) {
	ty, th := thumbGeom(int(s.H), s.total, s.visible, s.top)
	switch {
	case y >= ty && y < ty+th: // on the thumb: begin dragging
		s.dragging = true
		s.dragOff = y - ty
	case y < ty: // page up
		s.scrollTo(s.top - max1(s.visible-1))
	default: // page down
		s.scrollTo(s.top + max1(s.visible-1))
	}
}

func (s *Scrollbar) dragTo(y int) {
	top := topForThumbY(int(s.H), s.total, s.visible, y-s.dragOff)
	s.scrollTo(top)
}

func (s *Scrollbar) scrollTo(top int) {
	top = clampTop(s.total, s.visible, top)
	if top == s.top {
		return
	}
	s.top = top
	_ = s.Draw()
	if s.OnScroll != nil {
		s.OnScroll(top)
	}
}

// --- pure geometry (unit-tested offline) ---

// thumbGeom returns the thumb's y position and height within a track of trackH
// pixels for the given line range.
func thumbGeom(trackH, total, visible, top int) (y, h int) {
	if trackH <= 0 {
		return 0, 0
	}
	if total <= visible || total <= 0 {
		return 0, trackH // nothing to scroll: full-height thumb
	}
	h = trackH * visible / total
	if h < sbMinThumb {
		h = sbMinThumb
	}
	if h > trackH {
		h = trackH
	}
	maxTop := total - visible
	y = (trackH - h) * top / maxTop
	if y < 0 {
		y = 0
	}
	if y > trackH-h {
		y = trackH - h
	}
	return y, h
}

// topForThumbY is the inverse of thumbGeom's y: the top line that places the
// thumb at pixel y.
func topForThumbY(trackH, total, visible, y int) int {
	if total <= visible {
		return 0
	}
	_, h := thumbGeom(trackH, total, visible, 0)
	span := trackH - h
	if span <= 0 {
		return 0
	}
	maxTop := total - visible
	top := y * maxTop / span
	return clampTop(total, visible, top)
}

func clampTop(total, visible, top int) int {
	maxTop := max0(total - visible)
	if top < 0 {
		return 0
	}
	if top > maxTop {
		return maxTop
	}
	return top
}
