package main

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
	"log"
)

type MyWindow struct {
	tk_core.Window
	Gc_black base.GC
	font     base.FONT
	Win2     ChildWindow
	Button   tk_widget.Button
}

func (w *MyWindow) Init() {
	w.Window.SetWindowHandler(w)
	w.Window.Create()
	w.Window.Map()

	w.Win2 = ChildWindow{
		Window: tk_core.Window{
			Parent:    &w.Window,
			Conn:      w.Window.Conn,
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
			Parent:    &w.Window,
			Conn:      w.Window.Conn,
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
		//		log.Printf("KeyPressEvent: %+v\n", ev)
	case *events.KeyReleaseEvent:
		//		log.Printf("KeyPressEvent: %+v\n", ev)
	default:
		//		log.Printf("MyWindow::HandleWindowEvent: %T %+v\n", ev, ev)
	}
	return true
}
