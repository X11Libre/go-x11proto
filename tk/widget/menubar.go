package widget

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

const (
	menuBarHeight = 22
	menuBarPadX   = 10
)

type barEntry struct {
	title  string
	menu   *Menu
	x0, x1 int // title hit box within the bar
}

// MenuBar is a horizontal strip of menu titles; pressing a title pops up its
// Menu just below it. Build it with AddMenu, then Init.
//
// The embedded Window's Conn/Parent/X/Y/W must be set before Init; the height is
// fixed (menuBarHeight).
type MenuBar struct {
	tk_core.Window

	// TearOff makes every menu on the bar detachable (a dashed handle at the
	// top; click it to tear off a persistent copy). Set it before Init — e.g.
	// from the desktop theme: bar.TearOff = theme.Load(conn).TearOffMenus.
	// Individual menus can still opt in on their own via Menu.TearOff.
	TearOff bool

	gc      *tk_core.GC
	font    base.FONT
	entries []*barEntry

	openIdx            int // index of the currently open menu, -1 = none
	barRootX, barRootY int // the bar's top-left in root coordinates
}

// AddMenu appends a titled menu and returns it (so callers can keep a handle).
func (b *MenuBar) AddMenu(title string, items []MenuItem) *Menu {
	m := &Menu{Items: items}
	b.entries = append(b.entries, &barEntry{title: title, menu: m})
	return m
}

// Init creates the bar window, lays out the titles and creates the menus.
func (b *MenuBar) Init() error {
	b.openIdx = -1
	if b.font.Invalid() {
		f, err := b.Conn.GetFont("fixed")
		if err != nil {
			return err
		}
		b.font = f
	}
	b.H = menuBarHeight
	b.EventMask = event_mask.Exposure | event_mask.ButtonPress
	b.SetBackPixel = true
	b.BackPixel = b.Conn.X11Conn.DefaultWhitePixel()

	b.SetWindowHandler(b)
	if err := b.Create(); err != nil {
		return err
	}

	black := b.Conn.X11Conn.DefaultBlackPixel()
	white := b.Conn.X11Conn.DefaultWhitePixel()
	var err error
	if b.gc, err = b.Conn.CreateGC1(black, white, b.font); err != nil {
		return err
	}

	x := menuBarPadX
	for _, en := range b.entries {
		w := len(en.title) * menuCharW
		en.x0 = x - 4
		en.x1 = x + w + 4
		x = en.x1 + menuBarPadX
		en.menu.Drawable.Conn = b.Conn
		if b.TearOff {
			en.menu.TearOff = true // bar-wide: make every menu detachable
		}
		if err := en.menu.Init(); err != nil {
			return err
		}
	}
	return b.Map()
}

func (b *MenuBar) draw() {
	for _, en := range b.entries {
		b.PutText8(b.gc.XID, base.INT16(en.x0+4), menuBarHeight-7, en.title)
	}
}

// HandleWindowEvent is the tk WindowHandler for the bar.
func (b *MenuBar) HandleWindowEvent(ev events.Event) bool {
	switch e := ev.(type) {
	case *events.ExposeEvent:
		b.draw()
	case *events.ButtonPressEvent:
		for i, en := range b.entries {
			if int(e.EventX) >= en.x0 && int(e.EventX) < en.x1 {
				// the bar's root origin = press-root minus press-in-window.
				b.barRootX = int(e.RootX) - int(e.EventX)
				b.barRootY = int(e.RootY) - int(e.EventY)
				b.openMenu(i)
				break
			}
		}
	}
	return true
}

// openMenu pops up entry i's menu just below its title and arms hover-switching.
func (b *MenuBar) openMenu(i int) {
	en := b.entries[i]
	en.menu.onBarHover = b.hoverSwitch
	b.openIdx = i
	_ = en.menu.Popup(base.INT16(b.barRootX+en.x0), base.INT16(b.barRootY+menuBarHeight))
}

// hoverSwitch is called by the open menu when the pointer is over the bar but
// outside the cascade; if it is over a different title, switch to that menu.
func (b *MenuBar) hoverSwitch(rx, ry base.INT16) {
	lx, ly := int(rx)-b.barRootX, int(ry)-b.barRootY
	if ly < 0 || ly >= menuBarHeight {
		return // not over the bar strip
	}
	for i, en := range b.entries {
		if lx >= en.x0 && lx < en.x1 {
			if i != b.openIdx {
				if b.openIdx >= 0 {
					b.entries[b.openIdx].menu.Close()
				}
				b.openMenu(i)
			}
			return
		}
	}
}
