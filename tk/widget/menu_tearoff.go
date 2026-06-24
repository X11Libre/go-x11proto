package widget

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

const (
	tearRowH  = 10 // height of the tear-off handle row
	tearIndex = -2 // itemAtRoot result for the tear-off handle
)

// tearOff detaches a persistent copy of this menu: a normal (not
// override-redirect) window the window manager decorates, with no pointer grab,
// that stays open. Leaf items still run their OnClick (and the menu stays up);
// submenu cascades are not offered in a torn-off menu.
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
	m.Name = "Menu"

	m.SetWindowHandler(m)
	if err := m.Create(); err != nil {
		return err
	}
	var err error
	if m.gc, err = m.Conn.CreateGC1(black, white, m.font); err != nil {
		return err
	}
	if m.gcHi, err = m.Conn.CreateGC1(white, black, m.font); err != nil {
		return err
	}
	m.wmDelete, _ = m.Window.EnableWMDelete()
	return m.Map()
}

// itemAtLocal maps a window-local y to an item index (torn-off menus get events
// in window coordinates since they hold no grab).
func (m *Menu) itemAtLocal(y int) int {
	for i := range m.Items {
		if y >= m.itemY[i] && y < m.itemY[i+1] {
			return i
		}
	}
	return -1
}

// handleDetached is the event handler for a torn-off menu: highlight on hover,
// run leaf items on click (staying open), and close on the WM delete request.
func (m *Menu) handleDetached(ev events.Event) bool {
	if tk_core.IsWMDelete(ev, m.wmDelete) {
		_ = m.Destroy()
		return true
	}
	switch e := ev.(type) {
	case *events.ExposeEvent:
		m.draw()
	case *events.MotionEvent:
		nh := -1
		if i := m.itemAtLocal(int(e.EventY)); m.selectable(i) {
			nh = i
		}
		if m.hi != nh {
			m.hi = nh
			m.draw()
		}
	case *events.ButtonReleaseEvent:
		i := m.itemAtLocal(int(e.EventY))
		if m.selectable(i) && m.Items[i].Submenu == nil {
			if f := m.Items[i].OnClick; f != nil {
				f()
			}
		}
	}
	return true
}
