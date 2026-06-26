package widget

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

const (
	tearRowH  = 14 // height of the tear-off handle row (popup)
	titleBarH = 18 // height of the detached menu's title bar
	tearIndex = -2 // itemAtRoot / localIndex result for the top (handle) row
)

// topRowH is the height reserved at the top of the menu: the dashed tear-off
// handle on a TearOff popup, or the title bar on a detached copy.
func (m *Menu) topRowH() int {
	switch {
	case m.detached:
		return titleBarH
	case m.TearOff:
		return tearRowH
	}
	return 0
}

// tearOff detaches a persistent copy of this menu: a managed but undecorated
// window with its own title bar (a drag grip plus a close button) and no
// pointer grab. Drag the grip to move it, click the close button to re-attach
// (close). Leaf items still run their OnClick; submenu cascades aren't offered.
func (m *Menu) tearOff() {
	d := &Menu{Items: m.Items, detached: true}
	d.Drawable.Conn = m.Conn
	// offset a little so it visibly pops out as its own window rather than
	// sitting exactly where the popup was (which looks like nothing happened).
	_ = d.initDetached(m.rx+24, m.ry+24)
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
	// A managed window with decorations turned off (via _MOTIF_WM_HINTS): we
	// draw our own frame, but the window manager still routes button clicks to
	// us — an override-redirect window's clicks get swallowed by the WM's
	// passive button grabs (only motion gets through).
	if motif, err := m.Conn.InternAtom("_MOTIF_WM_HINTS"); err == nil {
		// flags=DECORATIONS(2), decorations=0 → no title bar / border
		_ = rpc.ChangeProperty32(m.Conn.X11Conn, 0, m.XID, motif, motif,
			[]base.CARD32{2, 0, 0, 0, 0})
	}
	// WM_NORMAL_HINTS with USPosition: tell the WM to honour our requested
	// position instead of running its own placement (which would drop the menu
	// in the middle of the screen, leaving our m.X/m.Y out of sync so the first
	// drag step snaps it back).
	if nh, err := m.Conn.InternAtom("WM_NORMAL_HINTS"); err == nil {
		sz, _ := m.Conn.InternAtom("WM_SIZE_HINTS")
		_ = rpc.ChangeProperty32(m.Conn.X11Conn, 0, m.XID, nh, sz, []base.CARD32{
			1,                                      // flags: USPosition
			base.CARD32(rootX), base.CARD32(rootY), // x, y (obsolete, for old WMs)
			base.CARD32(m.W), base.CARD32(m.H), // width, height (obsolete)
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // min/max/inc/aspect/base/gravity
		})
	}
	m.wmDelete, _ = m.Window.EnableWMDelete()
	var err error
	if m.gc, err = m.Conn.CreateGC1(black, white, m.font); err != nil {
		return err
	}
	if m.gcHi, err = m.Conn.CreateGC1(white, black, m.font); err != nil {
		return err
	}
	if err := m.Map(); err != nil {
		return err
	}
	if err := m.Raise(); err != nil { // sit above the main window
		return err
	}
	// Grab the focus right away: under a click-to-focus WM the first click would
	// otherwise be swallowed to focus the window instead of hitting an item.
	return rpc.SetInputFocus(m.Conn.X11Conn, 2 /*RevertToParent*/, m.XID, 0)
}

// drawTitleBar paints the detached menu's title bar: a dark bar with a dashed
// drag grip on the left and a close (×) button on the right.
func (m *Menu) drawTitleBar() {
	w := int(m.W)
	cx0 := w - titleBarH

	m.FillRect(m.gc.XID, 0, 0, m.W, titleBarH) // dark bar
	// drag grip: white dashes on the left, up to the close button
	for x := 5; x < cx0-4; x += 7 {
		m.FillRect(m.gcHi.XID, base.INT16(x), titleBarH/2-1, 4, 2)
	}
	// close button: white box when hovered, then a contrasting ×
	xgc := m.gcHi
	if m.closeHot {
		m.FillRect(m.gcHi.XID, base.INT16(cx0), 0, titleBarH, titleBarH)
		xgc = m.gc
	}
	const pad = 5
	m.PolySegment(xgc.XID, []base.Segment{
		{X1: base.INT16(cx0 + pad), Y1: pad, X2: base.INT16(w - pad), Y2: titleBarH - pad},
		{X1: base.INT16(w - pad), Y1: pad, X2: base.INT16(cx0 + pad), Y2: titleBarH - pad},
	})
}

// inCloseButton reports whether window-local (x, y) is over the close button.
func (m *Menu) inCloseButton(x, y int) bool {
	return y >= 0 && y < titleBarH && x >= int(m.W)-titleBarH
}

// localIndex maps a window-local y to an item index, or tearIndex for the top
// row (torn-off menus get events in window coordinates, except while dragging).
func (m *Menu) localIndex(y int) int {
	if h := m.topRowH(); h > 0 && y < h {
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
	case *events.ClientMessageEvent:
		if tk_core.IsWMDelete(e, m.wmDelete) { // WM close button, if WM ignored our hints
			_ = m.Destroy()
		}
	case *events.ButtonPressEvent:
		if e.Key != 1 {
			break
		}
		x, y := int(e.EventX), int(e.EventY)
		switch {
		case m.inCloseButton(x, y):
			_ = m.Destroy() // close (re-attach)
		case y < titleBarH:
			m.startDrag(base.INT16(e.RootX), base.INT16(e.RootY))
		}
	case *events.MotionEvent:
		if m.dragging {
			m.dragTo(base.INT16(e.RootX), base.INT16(e.RootY))
		} else {
			m.hoverDetached(int(e.EventX), int(e.EventY))
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

// hoverDetached updates the highlighted item and the close-button hover state.
func (m *Menu) hoverDetached(x, y int) {
	hot := m.inCloseButton(x, y)
	nh := -1
	if !hot {
		if i := m.localIndex(y); m.selectable(i) {
			nh = i
		}
	}
	if nh != m.hi || hot != m.closeHot {
		m.hi, m.closeHot = nh, hot
		m.draw()
	}
}

// startDrag grabs the pointer and records the drag origin (a press on the title
// bar), so dragging moves the window even when the pointer leaves it.
func (m *Menu) startDrag(rx, ry base.INT16) {
	m.dragging = true
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
	m.X = m.dragWX + (rx - m.dragRX)
	m.Y = m.dragWY + (ry - m.dragRY)
	_ = m.Move(m.X, m.Y)
}

func (m *Menu) endDrag() {
	m.dragging = false
	_ = rpc.UngrabPointer(m.Conn.X11Conn, 0)
}
