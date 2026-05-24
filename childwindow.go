package main

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

type ChildWindow struct {
	tk_core.Window
	Gc_black base.GC
	font     base.FONT
}

func (w *ChildWindow) Init() {
	w.Window.SetWindowHandler(w)
	w.Window.Create()
	w.Window.Map()
}

func (w *ChildWindow) HandleWindowEvent(ev events.Event) bool {
	if w.font == 0 {
		fontid, err := w.Window.Conn.GetFont("fixed")
		errPanic(err, "OpenFont")
		w.font = fontid
	}

	if w.Gc_black == 0 {
		gcid, _ := rpc.CreateGC1(
			w.Window.Conn.X11Conn,
			w.Window.Conn.X11Conn.DefaultBlackPixel(),
			w.Window.Conn.X11Conn.DefaultWhitePixel(),
			w.font,
		)
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
			"child win",
		)
	}
	return true
}
