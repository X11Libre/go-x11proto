package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
)

// TestTkMenu smoke-tests the MenuBar/Menu widgets: build a bar with two menus,
// create them (override-redirect popups), pop one up and close it. Pointer-
// driven selection can't be simulated headless, so this checks the wire
// operations (create/grab/map/unmap) don't error.
func TestTkMenu(t *testing.T) {
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)

	parent := tk_core.Window{
		Drawable:  tk_core.Drawable{Conn: &tk},
		ParentXID: c.DefaultRoot(),
		W:         300, H: 200,
		EventMask: event_mask.Exposure,
	}
	must(t, parent.Create(), "parent.Create")
	must(t, parent.Map(), "parent.Map")

	bar := &tk_widget.MenuBar{
		Window: tk_core.Window{
			Drawable:  tk_core.Drawable{Conn: &tk},
			ParentXID: parent.XID,
			X:         0, Y: 0, W: 300,
		},
	}
	fileMenu := bar.AddMenu("File", []tk_widget.MenuItem{
		{Label: "Open", OnClick: func() {}},
		{Label: "Quit", OnClick: func() {}},
	})
	bar.AddMenu("Help", []tk_widget.MenuItem{{Label: "About"}})
	must(t, bar.Init(), "MenuBar.Init")

	must(t, fileMenu.Popup(40, 40), "Menu.Popup")
	fileMenu.Close()

	must(t, parent.Destroy(), "parent.Destroy")
}

// TestTkContextMenu smoke-tests a standalone context menu with separators and
// nested (multi-layer) submenus: build it, pop it up, close it.
func TestTkContextMenu(t *testing.T) {
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)

	ctx := &tk_widget.Menu{
		Items: []tk_widget.MenuItem{
			{Label: "New", OnClick: func() {}},
			{Separator: true},
			{Label: "Recent", Submenu: []tk_widget.MenuItem{
				{Label: "a.txt"},
				{Separator: true},
				{Label: "Clear", OnClick: func() {}},
			}},
			{Label: "Options", Submenu: []tk_widget.MenuItem{
				{Label: "Theme", Submenu: []tk_widget.MenuItem{ // third layer
					{Label: "Light"},
					{Label: "Dark"},
				}},
			}},
			{Separator: true},
			{Label: "Quit", OnClick: func() {}},
		},
	}
	ctx.Drawable.Conn = &tk
	must(t, ctx.Init(), "context Menu.Init")
	must(t, ctx.Popup(80, 80), "context Menu.Popup")
	ctx.Close()
}
