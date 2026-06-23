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
	gc      *tk_core.GC
	font    base.FONT
	entries []*barEntry
}

// AddMenu appends a titled menu and returns it (so callers can keep a handle).
func (b *MenuBar) AddMenu(title string, items []MenuItem) *Menu {
	m := &Menu{Items: items}
	b.entries = append(b.entries, &barEntry{title: title, menu: m})
	return m
}

// Init creates the bar window, lays out the titles and creates the menus.
func (b *MenuBar) Init() error {
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
		for _, en := range b.entries {
			if int(e.EventX) >= en.x0 && int(e.EventX) < en.x1 {
				// the bar's root origin = press-root minus press-in-window;
				// drop the menu just below the bar at the title's left edge.
				rootX := base.INT16(int(e.RootX) - int(e.EventX) + en.x0)
				rootY := base.INT16(int(e.RootY) - int(e.EventY) + menuBarHeight)
				_ = en.menu.Popup(rootX, rootY)
				break
			}
		}
	}
	return true
}
