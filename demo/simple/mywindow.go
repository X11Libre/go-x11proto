package main

import (
	_ "embed"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
	"github.com/X11Libre/go-x11proto/tk/xpm"
	"log"
)

//go:embed xlogo.xpm
var xlogoXPM []byte

type MyWindow struct {
	tk_core.Window
	Gc_black base.GC
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
			rpc.SetWindowBackgroundPixmap(conn, w.Window.XID, pixmap)
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

	if w.Gc_black.Invalid() {
		gcid, err := rpc.CreateGC1(
			w.Window.Conn.X11Conn,
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
			w.Gc_black,
			[]base.Rectangle{
				{5, 60, 50, 50},
				{60, 60, 50, 50},
			},
		)
		w.PutText8(
			w.Gc_black,
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
