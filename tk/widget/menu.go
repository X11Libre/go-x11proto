package widget

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

// MenuItem is one entry of a Menu. An item with a nil OnClick (or empty Label)
// is selectable but does nothing.
type MenuItem struct {
	Label   string
	OnClick func()
}

// menu layout metrics (in pixels, sized for the "fixed" font).
const (
	menuItemHeight = 20
	menuPadX       = 8
	menuTextBase   = 14 // text baseline offset within an item
	menuCharW      = 7  // approximate glyph advance, for sizing
)

// Menu is a popup menu: an override-redirect window listing items vertically.
// Popup maps it at a root position and grabs the pointer (classic press-drag-
// release behaviour): moving over an item highlights it, releasing on it runs
// its OnClick, and releasing/pressing elsewhere just closes the menu.
//
// The embedded Window's Conn must be set before Init; Parent defaults to root.
type Menu struct {
	tk_core.Window
	Items []MenuItem

	gc   *tk_core.GC // black-on-white text (and the highlight fill)
	gcHi *tk_core.GC // white-on-black text for the highlighted item
	font base.FONT
	hi   int // highlighted item index, -1 = none
	open bool
}

// Init creates the (initially unmapped) override-redirect menu window.
func (m *Menu) Init() error {
	m.hi = -1
	if m.font.Invalid() {
		f, err := m.Conn.GetFont("fixed")
		if err != nil {
			return err
		}
		m.font = f
	}

	m.W = base.CARD16(menuWidth(m.Items))
	m.H = base.CARD16(menuItemHeight * len(m.Items))
	m.EventMask = event_mask.Exposure | event_mask.ButtonPress |
		event_mask.ButtonRelease | event_mask.PointerMotion
	m.SetBackPixel = true
	m.BackPixel = m.Conn.X11Conn.DefaultWhitePixel()
	m.SetBorderPixel = true
	m.BorderPixel = m.Conn.X11Conn.DefaultBlackPixel()
	m.BorderWidth = 1

	m.SetWindowHandler(m)
	if err := m.Create(); err != nil {
		return err
	}
	if err := m.SetOverrideRedirect(true); err != nil {
		return err
	}

	black := m.Conn.X11Conn.DefaultBlackPixel()
	white := m.Conn.X11Conn.DefaultWhitePixel()
	var err error
	if m.gc, err = m.Conn.CreateGC1(black, white, m.font); err != nil {
		return err
	}
	if m.gcHi, err = m.Conn.CreateGC1(white, black, m.font); err != nil {
		return err
	}
	return nil
}

func menuWidth(items []MenuItem) int {
	max := 1
	for _, it := range items {
		if n := len(it.Label); n > max {
			max = n
		}
	}
	return max*menuCharW + 2*menuPadX
}

// Popup maps the menu with its top-left at root coordinates (rootX, rootY) and
// grabs the pointer so the menu receives all subsequent pointer events.
func (m *Menu) Popup(rootX, rootY base.INT16) error {
	if err := m.Move(rootX, rootY); err != nil {
		return err
	}
	if err := m.Raise(); err != nil {
		return err
	}
	if err := m.Map(); err != nil {
		return err
	}
	m.open = true
	m.hi = -1
	_, err := rpc.GrabPointer(m.Conn.X11Conn, &request.GrabPointerRequest{
		GrabWindow:   m.XID,
		OwnerEvents:  false,
		EventMask:    base.CARD16(event_mask.ButtonPress | event_mask.ButtonRelease | event_mask.PointerMotion),
		PointerMode:  request.GrabModeAsync,
		KeyboardMode: request.GrabModeAsync,
	})
	return err
}

// Close ungrabs the pointer and unmaps the menu.
func (m *Menu) Close() {
	if !m.open {
		return
	}
	m.open = false
	_ = rpc.UngrabPointer(m.Conn.X11Conn, 0)
	_ = m.Unmap()
}

// itemAt returns the item index at window coordinates (x,y), or -1 if outside.
func (m *Menu) itemAt(x, y base.CARD16) int {
	if int(x) >= int(m.W) || int(y) >= int(m.H) {
		return -1
	}
	i := int(y) / menuItemHeight
	if i < 0 || i >= len(m.Items) {
		return -1
	}
	return i
}

func (m *Menu) draw() {
	for i, it := range m.Items {
		y0 := base.INT16(i * menuItemHeight)
		if i == m.hi {
			m.FillRect(m.gc.XID, 0, y0, m.W, menuItemHeight) // black bar
			m.PutText8(m.gcHi.XID, menuPadX, y0+menuTextBase, it.Label)
		} else {
			m.PutText8(m.gc.XID, menuPadX, y0+menuTextBase, it.Label)
		}
	}
}

// HandleWindowEvent is the tk WindowHandler for the menu.
func (m *Menu) HandleWindowEvent(ev events.Event) bool {
	switch e := ev.(type) {
	case *events.ExposeEvent:
		m.draw()
	case *events.MotionEvent:
		if idx := m.itemAt(e.EventX, e.EventY); idx != m.hi {
			m.hi = idx
			m.ClearArea(0, 0, 0, 0, false)
			m.draw()
		}
	case *events.ButtonReleaseEvent:
		idx := m.itemAt(e.EventX, e.EventY)
		m.Close()
		if idx >= 0 && m.Items[idx].OnClick != nil {
			m.Items[idx].OnClick()
		}
	case *events.ButtonPressEvent:
		if m.itemAt(e.EventX, e.EventY) < 0 {
			m.Close() // a press outside the menu dismisses it
		}
	}
	return true
}
