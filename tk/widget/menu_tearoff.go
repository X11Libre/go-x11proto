package widget

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

const (
	tearRowH  = 10 // height of the tear-off / drag handle row
	tearIndex = -2 // itemAtRoot / localIndex result for the handle row
	dragSlop  = 3  // movement past which a handle press counts as a drag
)

// tearHandle reports whether the menu shows the dashed handle row: a tear-off
// popup (TearOff) or a detached copy (where the handle is the title/drag bar).
func (m *Menu) tearHandle() bool { return m.TearOff || m.detached }

// tearOff detaches a persistent copy of this menu: an override-redirect window
// with no WM decoration and no pointer grab, carrying its own dashed handle bar.
// Dragging the bar moves it; a plain click on the bar re-attaches (closes it).
// Leaf items still run their OnClick; submenu cascades are not offered.
func (m *Menu) tearOff() {
	d := &Menu{Items: m.Items, detached: true}
	d.Drawable.Conn = m.Conn
	_ = d.initDetached(m.rx, m.ry)
}

// initDetached creates and maps the torn-off window at the given root position.
func (m *Menu) initDetached(rootX, rootY base.INT16) error {
	m.hi = -1
	m.childItem = -1
	if m.font.Invalid() {
		f, err := m.Conn.GetFont("fixed")
		if err != nil {
			return err
		}
		m.font = f
	}
	m.layout()

	black := m.Conn.X11Conn.DefaultBlackPixel()
	white := m.Conn.X11Conn.DefaultWhitePixel()
	m.X, m.Y = rootX, rootY
	m.EventMask = event_mask.Exposure | event_mask.ButtonPress |
		event_mask.ButtonRelease | event_mask.PointerMotion
	m.SetBackPixel = true
	m.BackPixel = white
	m.SetBorderPixel = true
	m.BorderPixel = black
	m.BorderWidth = 1

	m.SetWindowHandler(m)
	if err := m.Create(); err != nil {
		return err
	}
	if err := m.SetOverrideRedirect(true); err != nil { // our own frame, no WM
		return err
	}
	var err error
	if m.gc, err = m.Conn.CreateGC1(black, white, m.font); err != nil {
		return err
	}
	if m.gcHi, err = m.Conn.CreateGC1(white, black, m.font); err != nil {
		return err
	}
	return m.Map()
}

// localIndex maps a window-local y to an item index, or tearIndex for the
// handle row (torn-off menus get events in window coordinates, holding no grab
// except while dragging).
func (m *Menu) localIndex(y int) int {
	if m.tearHandle() && y < tearRowH {
		return tearIndex
	}
	for i := range m.Items {
		if y >= m.itemY[i] && y < m.itemY[i+1] {
			return i
		}
	}
	return -1
}

// handleDetached is the event handler for a torn-off menu.
func (m *Menu) handleDetached(ev events.Event) bool {
	switch e := ev.(type) {
	case *events.ExposeEvent:
		m.draw()
	case *events.ButtonPressEvent:
		if e.Key == 1 && m.localIndex(int(e.EventY)) == tearIndex {
			m.startDrag(base.INT16(e.RootX), base.INT16(e.RootY))
		}
	case *events.MotionEvent:
		if m.dragging {
			m.dragTo(base.INT16(e.RootX), base.INT16(e.RootY))
		} else {
			m.hover(int(e.EventY))
		}
	case *events.ButtonReleaseEvent:
		if m.dragging {
			m.endDrag()
		} else if i := m.localIndex(int(e.EventY)); m.selectable(i) && m.Items[i].Submenu == nil {
			if f := m.Items[i].OnClick; f != nil {
				f() // run the action; the torn-off menu stays open
			}
		}
	}
	return true
}

func (m *Menu) hover(y int) {
	nh := -1
	if i := m.localIndex(y); m.selectable(i) {
		nh = i
	}
	if m.hi != nh {
		m.hi = nh
		m.draw()
	}
}

// startDrag grabs the pointer and records the drag origin (a press on the
// handle bar). It resolves to a move or, if the pointer barely moves, a
// re-attach on release.
func (m *Menu) startDrag(rx, ry base.INT16) {
	m.dragging = true
	m.dragMoved = false
	m.dragRX, m.dragRY = rx, ry
	m.dragWX, m.dragWY = m.X, m.Y
	_, _ = rpc.GrabPointer(m.Conn.X11Conn, &request.GrabPointerRequest{
		GrabWindow:   m.XID,
		OwnerEvents:  false,
		EventMask:    base.CARD16(event_mask.ButtonRelease | event_mask.PointerMotion),
		PointerMode:  request.GrabModeAsync,
		KeyboardMode: request.GrabModeAsync,
	})
}

func (m *Menu) dragTo(rx, ry base.INT16) {
	dx, dy := int(rx)-int(m.dragRX), int(ry)-int(m.dragRY)
	if abs(dx)+abs(dy) > dragSlop {
		m.dragMoved = true
	}
	m.X = m.dragWX + base.INT16(dx)
	m.Y = m.dragWY + base.INT16(dy)
	_ = m.Move(m.X, m.Y)
}

func (m *Menu) endDrag() {
	m.dragging = false
	_ = rpc.UngrabPointer(m.Conn.X11Conn, 0)
	if !m.dragMoved { // a click on the handle (no drag) re-attaches: close it
		_ = m.Destroy()
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
