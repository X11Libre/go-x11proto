package main

import (
	_ "embed"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
	"github.com/X11Libre/go-x11proto/tk/xpm"
	"log"
)

//go:embed xlogo.xpm
var xlogoXPM []byte

type MyWindow struct {
	tk_core.Window
	Gc_black *tk_core.GC
	font     base.FONT
	bgPixmap base.PIXMAP
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

	switch ev.(type) {
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
			"hello foo bar",
		)
	case *events.KeyPressEvent:
	case *events.KeyReleaseEvent:
	default:
	}
	return true
}
