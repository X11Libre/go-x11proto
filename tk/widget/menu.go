package widget

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

// MenuItem is one entry of a Menu:
//   - a normal item has a Label and an OnClick;
//   - an item with a non-nil Submenu cascades to a child menu (OnClick ignored);
//   - an item with Separator set is a non-selectable divider (Label ignored).
//
// Accel is an optional accelerator shown right-aligned (e.g. "Ctrl+O"). If it
// parses (see parseAccel), MenuBar.HandleKey / Menu.HandleKey will fire the
// item's OnClick when a matching key is pressed.
type MenuItem struct {
	Label     string
	OnClick   func()
	Submenu   []MenuItem
	Separator bool
	Accel     string
}

// menu layout metrics (pixels, sized for the "fixed" font).
const (
	menuItemHeight = 20
	menuSepHeight  = 7
	menuPadX       = 8
	menuTextBase   = 14
	menuCharW      = 7
	menuArrowW     = 14 // reserved column for the submenu arrow
	menuAccelGap   = 3 * menuCharW
)

// Menu is a popup menu with separators and cascading submenus. Pop it up with
// Popup (e.g. as a context menu on a right-click, or from a MenuBar); the
// top-level menu grabs the pointer and drives the whole cascade via root
// coordinates. It is click-to-open: the menu stays up after the opening click,
// hovering a submenu item opens it to the right, clicking a leaf runs its
// OnClick, and a click outside dismisses the cascade.
//
// The embedded Window's Conn must be set before Init; Parent defaults to root.
type Menu struct {
	tk_core.Window
	Items []MenuItem

	// TearOff adds a dashed handle at the top of the menu; clicking it detaches
	// a persistent, window-manager-managed copy that stays open (like the old
	// GTK tear-off menus). Set it before Init / Popup.
	TearOff bool

	gc   *tk_core.GC // black-on-white text + separator/highlight fill
	gcHi *tk_core.GC // white-on-black text for the highlighted item
	font base.FONT

	itemY     []int // itemY[i] = top of item i; itemY[len] = total height
	hasSub    bool  // any item has a submenu (reserves the arrow column)
	hi        int   // highlighted selectable item, -1 = none
	parent    *Menu // menu that opened this one (nil = top of cascade)
	child     *Menu // currently open submenu of this menu
	childItem int   // item index that opened child, -1 = none
	subs      map[int]*Menu
	rx, ry    base.INT16 // window position in root coordinates
	detached  bool       // a torn-off, persistent copy (own frame, no grab)

	// drag state for a detached menu: the handle row doubles as a title bar —
	// dragging it moves the window, a plain click on it re-attaches (closes).
	dragging       bool
	dragMoved      bool
	dragRX, dragRY base.INT16 // pointer root position at drag start
	dragWX, dragWY base.INT16 // window position at drag start

	// onBarHover, when set on the top menu (by a MenuBar), is called with the
	// pointer's root position while it is outside the whole cascade — letting the
	// bar switch to another menu when the pointer slides over its titles.
	onBarHover func(rx, ry base.INT16)

	// press tracking (top-of-cascade only): a release selects only when it
	// matches a press on the same item, so the click that opened the menu (whose
	// press landed outside, before the grab) does not select anything.
	pressMenu *Menu
	pressIdx  int
}

// Init creates the (initially unmapped) override-redirect menu window.
func (m *Menu) Init() error {
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

func (m *Menu) layout() {
	m.itemY = make([]int, len(m.Items)+1)
	y, hasSub := 0, false
	if m.tearHandle() {
		y = tearRowH // reserve the tear-off / drag handle row at the top
	}
	for i, it := range m.Items {
		m.itemY[i] = y
		if it.Separator {
			y += menuSepHeight
		} else {
			y += menuItemHeight
		}
		if it.Submenu != nil {
			hasSub = true
		}
	}
	m.itemY[len(m.Items)] = y
	m.hasSub = hasSub

	maxLabel, maxAccel := 1, 0
	for _, it := range m.Items {
		if it.Separator {
			continue
		}
		if len(it.Label) > maxLabel {
			maxLabel = len(it.Label)
		}
		if len(it.Accel) > maxAccel {
			maxAccel = len(it.Accel)
		}
	}
	pix := maxLabel*menuCharW + 2*menuPadX
	if maxAccel > 0 {
		pix += menuAccelGap + maxAccel*menuCharW
	}
	if hasSub {
		pix += menuArrowW
	}
	m.W = base.CARD16(pix)
	m.H = base.CARD16(y)
}

// Popup maps the menu with its top-left at root coordinates (rootX, rootY) and
// grabs the pointer; the menu is now the top of the cascade.
func (m *Menu) Popup(rootX, rootY base.INT16) error {
	m.parent = nil
	m.rx, m.ry = rootX, rootY
	m.hi = -1
	m.pressMenu = nil
	m.pressIdx = -1
	if err := m.Move(rootX, rootY); err != nil {
		return err
	}
	if err := m.Raise(); err != nil {
		return err
	}
	if err := m.Map(); err != nil {
		return err
	}
	_, err := rpc.GrabPointer(m.Conn.X11Conn, &request.GrabPointerRequest{
		GrabWindow:   m.XID,
		OwnerEvents:  false,
		EventMask:    base.CARD16(event_mask.ButtonPress | event_mask.ButtonRelease | event_mask.PointerMotion),
		PointerMode:  request.GrabModeAsync,
		KeyboardMode: request.GrabModeAsync,
	})
	return err
}

// Close dismisses the whole cascade (call on the top menu).
func (m *Menu) Close() { m.topMenu().closeAll() }

func (m *Menu) topMenu() *Menu {
	for m.parent != nil {
		m = m.parent
	}
	return m
}

func (top *Menu) closeAll() {
	top.closeChild()
	_ = rpc.UngrabPointer(top.Conn.X11Conn, 0)
	_ = top.Unmap()
	top.hi = -1
}

func (m *Menu) closeChild() {
	if m.child != nil {
		m.child.closeChild()
		_ = m.child.Unmap()
		m.child = nil
		m.childItem = -1
	}
}

// subOf returns (creating on first use) the Menu for item i's submenu.
func (m *Menu) subOf(i int) *Menu {
	if m.subs == nil {
		m.subs = map[int]*Menu{}
	}
	if s, ok := m.subs[i]; ok {
		return s
	}
	s := &Menu{Items: m.Items[i].Submenu}
	s.Drawable.Conn = m.Conn
	if err := s.Init(); err != nil {
		return nil
	}
	m.subs[i] = s
	return s
}

func (m *Menu) openChild(i int) {
	sub := m.subOf(i)
	if sub == nil {
		return
	}
	sub.parent = m
	sub.hi = -1
	sub.rx = m.rx + base.INT16(int(m.W)-2)
	sub.ry = m.ry + base.INT16(m.itemY[i])
	_ = sub.Move(sub.rx, sub.ry)
	_ = sub.Raise()
	_ = sub.Map()
	m.child = sub
	m.childItem = i
}

func (m *Menu) containsRoot(rx, ry base.INT16) bool {
	return int(rx) >= int(m.rx) && int(rx) < int(m.rx)+int(m.W) &&
		int(ry) >= int(m.ry) && int(ry) < int(m.ry)+int(m.H)
}

// deepestContaining returns the deepest open menu whose rectangle holds the
// point, or nil if the point is over none of them.
func (top *Menu) deepestContaining(rx, ry base.INT16) *Menu {
	var found *Menu
	for m := top; m != nil; m = m.child {
		if m.containsRoot(rx, ry) {
			found = m
		}
	}
	return found
}

func (m *Menu) itemAtRoot(rx, ry base.INT16) int {
	if !m.containsRoot(rx, ry) {
		return -1
	}
	yy := int(ry) - int(m.ry)
	if m.tearHandle() && yy < tearRowH {
		return tearIndex
	}
	for i := range m.Items {
		if yy >= m.itemY[i] && yy < m.itemY[i+1] {
			return i
		}
	}
	return -1
}

func (m *Menu) selectable(i int) bool {
	return i >= 0 && i < len(m.Items) && !m.Items[i].Separator
}

func (m *Menu) draw() {
	m.ClearArea(0, 0, 0, 0, false)
	if m.tearHandle() {
		// a dashed handle across the top (tear-off in a popup, drag/re-attach
		// title bar in a detached menu)
		for x := 5; x < int(m.W)-4; x += 6 {
			m.FillRect(m.gc.XID, base.INT16(x), tearRowH/2, 3, 1)
		}
	}
	for i, it := range m.Items {
		y0 := base.INT16(m.itemY[i])
		if it.Separator {
			m.FillRect(m.gc.XID, 4, y0+base.INT16(menuSepHeight/2), m.W-8, 1)
			continue
		}
		text := m.gc
		if i == m.hi {
			m.FillRect(m.gc.XID, 0, y0, m.W, menuItemHeight)
			text = m.gcHi
		}
		m.PutText8(text.XID, menuPadX, y0+menuTextBase, it.Label)
		if it.Accel != "" {
			right := int(m.W) - menuPadX
			if m.hasSub {
				right -= menuArrowW
			}
			ax := right - len(it.Accel)*menuCharW
			m.PutText8(text.XID, base.INT16(ax), y0+menuTextBase, it.Accel)
		}
		if it.Submenu != nil {
			m.PutText8(text.XID, base.INT16(int(m.W)-menuArrowW), y0+menuTextBase, ">")
		}
	}
}

func (top *Menu) handleMotion(rx, ry base.INT16) {
	cur := top.deepestContaining(rx, ry)
	if cur == nil {
		// outside the cascade: let a menu bar switch menus if the pointer is
		// over one of its other titles.
		if top.onBarHover != nil {
			top.onBarHover(rx, ry)
		}
		return // otherwise keep the cascade as-is
	}
	i := cur.itemAtRoot(rx, ry)
	newHi := -1
	if cur.selectable(i) {
		newHi = i
	}
	if cur.hi != newHi {
		cur.hi = newHi
		cur.draw()
	}
	if newHi >= 0 && cur.Items[newHi].Submenu != nil {
		if cur.childItem != newHi {
			cur.closeChild()
			cur.openChild(newHi)
		}
	} else {
		cur.closeChild()
	}
}

func (top *Menu) handlePress(rx, ry base.INT16) {
	cur := top.deepestContaining(rx, ry)
	if cur == nil {
		top.closeAll() // press outside the cascade dismisses it
		return
	}
	idx := cur.itemAtRoot(rx, ry)
	if idx == tearIndex { // pressing the tear-off handle detaches immediately
		top.closeAll()
		cur.tearOff()
		return
	}
	top.pressMenu = cur
	top.pressIdx = idx
}

func (top *Menu) handleRelease(rx, ry base.INT16) {
	cur := top.deepestContaining(rx, ry)
	pressMenu, pressIdx := top.pressMenu, top.pressIdx
	top.pressMenu, top.pressIdx = nil, -1

	// releasing over the tear-off handle detaches, so the classic
	// press-on-title, drag-to-handle, release gesture works too (its opening
	// press landed on the bar, not the menu, so it isn't press-tracked).
	if cur != nil && cur.itemAtRoot(rx, ry) == tearIndex {
		top.closeAll()
		cur.tearOff()
		return
	}

	// select only on a full click on the same leaf item; ignore the release of
	// the click that opened the menu (its press was outside the grab) so the
	// menu stays up until an item is actually clicked.
	if cur == nil || cur != pressMenu {
		return
	}
	i := cur.itemAtRoot(rx, ry)
	if i != pressIdx || !cur.selectable(i) || cur.Items[i].Submenu != nil {
		return
	}
	action := cur.Items[i].OnClick
	top.closeAll()
	if action != nil {
		action()
	}
}

// HandleWindowEvent is the tk WindowHandler. Each menu window draws itself on
// Expose; pointer events (delivered to the top via the grab) drive the cascade.
func (m *Menu) HandleWindowEvent(ev events.Event) bool {
	if m.detached {
		return m.handleDetached(ev)
	}
	switch e := ev.(type) {
	case *events.ExposeEvent:
		m.draw()
	case *events.MotionEvent:
		m.topMenu().handleMotion(base.INT16(e.RootX), base.INT16(e.RootY))
	case *events.ButtonReleaseEvent:
		m.topMenu().handleRelease(base.INT16(e.RootX), base.INT16(e.RootY))
	case *events.ButtonPressEvent:
		m.topMenu().handlePress(base.INT16(e.RootX), base.INT16(e.RootY))
	}
	return true
}
