package main

import (
	_ "embed"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
	"github.com/X11Libre/go-x11proto/tk/xpm"
	"log"
	"os"
)

//go:embed xlogo.xpm
var xlogoXPM []byte

type MyWindow struct {
	tk_core.Window
	Gc_black *tk_core.GC
	font     base.FONT
	bgPixmap base.PIXMAP
	Menu     tk_widget.MenuBar
	Ctx      *tk_widget.Menu // right-click context menu
	Win2     ChildWindow
	Button   tk_widget.Button
}

func (w *MyWindow) Init() {
	w.Window.SetWindowHandler(w)
	w.Window.Create()

	// Upload background pixmap before mapping so it's visible immediately.
	conn := w.Window.Conn.X11Conn
	if img, err := xpm.DecodeBytes(xlogoXPM); err == nil {
		if pixmap, err := img.Upload(conn, conn.DefaultRoot()); err == nil {
			w.Window.SetBackgroundPixmap(pixmap)
			w.bgPixmap = pixmap
		}
	}

	w.Window.Map()

	// popup menu bar across the top
	w.Menu = tk_widget.MenuBar{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: w.Window.Conn},
			Parent:   &w.Window,
			X:        0, Y: 0, W: 500,
		},
	}
	w.Menu.AddMenu("File", []tk_widget.MenuItem{
		{Label: "Open", OnClick: func() { log.Printf("menu: File / Open\n") }},
		{Label: "Save", OnClick: func() { log.Printf("menu: File / Save\n") }},
		{Label: "Quit", OnClick: func() { log.Printf("menu: File / Quit\n"); os.Exit(0) }},
	})
	w.Menu.AddMenu("Edit", []tk_widget.MenuItem{
		{Label: "Cut", OnClick: func() { log.Printf("menu: Edit / Cut\n") }},
		{Label: "Copy", OnClick: func() { log.Printf("menu: Edit / Copy\n") }},
		{Label: "Paste", OnClick: func() { log.Printf("menu: Edit / Paste\n") }},
	})
	w.Menu.AddMenu("Help", []tk_widget.MenuItem{
		{Label: "About", OnClick: func() { log.Printf("menu: Help / About\n") }},
	})
	errPanic(w.Menu.Init(), "create menu bar")

	// right-click context menu with separators and nested submenus
	w.Ctx = &tk_widget.Menu{
		Items: []tk_widget.MenuItem{
			{Label: "New", OnClick: func() { log.Printf("ctx: New\n") }},
			{Label: "Open", OnClick: func() { log.Printf("ctx: Open\n") }},
			{Separator: true},
			{Label: "Recent", Submenu: []tk_widget.MenuItem{
				{Label: "alpha.txt", OnClick: func() { log.Printf("ctx: Recent / alpha\n") }},
				{Label: "beta.txt", OnClick: func() { log.Printf("ctx: Recent / beta\n") }},
				{Separator: true},
				{Label: "Clear", OnClick: func() { log.Printf("ctx: Recent / Clear\n") }},
			}},
			{Label: "Options", Submenu: []tk_widget.MenuItem{
				{Label: "Theme", Submenu: []tk_widget.MenuItem{ // a third layer
					{Label: "Light", OnClick: func() { log.Printf("ctx: Theme / Light\n") }},
					{Label: "Dark", OnClick: func() { log.Printf("ctx: Theme / Dark\n") }},
				}},
				{Label: "Verbose", OnClick: func() { log.Printf("ctx: Verbose\n") }},
			}},
			{Separator: true},
			{Label: "Quit", OnClick: func() { os.Exit(0) }},
		},
	}
	w.Ctx.Drawable.Conn = w.Window.Conn
	errPanic(w.Ctx.Init(), "create context menu")

	w.Win2 = ChildWindow{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{
				Conn: w.Window.Conn,
			},
			Parent:    &w.Window,
			Name:      "HELLO WORLD EXAMPLE",
			X:         100,
			Y:         100,
			W:         300,
			H:         150,
			EventMask: 0xFFFFFF,
		},
	}
	w.Win2.Init()

	w.Button = tk_widget.Button{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{
				Conn: w.Window.Conn,
			},
			Parent:    &w.Window,
			Name:      "button test",
			X:         100,
			Y:         300,
			W:         80,
			H:         30,
			EventMask: 0xFFFFFF,
		},
		OnButtonPress: func() {
			log.Printf("==> button pressed\n")
		},
	}
	w.Button.Init()
}

func (w *MyWindow) HandleWindowEvent(ev events.Event) bool {

	if w.font.Invalid() {
		fontid, err := w.Window.Conn.GetFont("fixed")
		errPanic(err, "OpenFont")
		w.font = fontid
	}

	if w.Gc_black == nil {
		gcid, err := w.Window.Conn.CreateGC1(
			w.Window.Conn.X11Conn.DefaultBlackPixel(),
			w.Window.Conn.X11Conn.DefaultWhitePixel(),
			w.font,
		)
		errPanic(err, "MyWindow => CreateGC1()")
		w.Gc_black = gcid
	}

	switch e := ev.(type) {
	case *events.ExposeEvent:
		w.FillRects(
			w.Gc_black.XID,
			[]base.Rectangle{
				{X: 5, Y: 60, Width: 50, Height: 50},
				{X: 60, Y: 60, Width: 50, Height: 50},
			},
		)
		w.PutText8(
			w.Gc_black.XID,
			30,
			30,
			"hello foo bar (right-click for a context menu)",
		)
	case *events.ButtonPressEvent:
		if e.Key == 3 { // right button: pop up the context menu at the pointer
			w.Ctx.Popup(base.INT16(e.RootX), base.INT16(e.RootY))
		}
	case *events.KeyPressEvent:
	case *events.KeyReleaseEvent:
	default:
	}
	return true
}
