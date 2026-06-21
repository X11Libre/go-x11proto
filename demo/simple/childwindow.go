package main

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

type ChildWindow struct {
	tk_core.Window
	Gc_black *tk_core.GC
	font     base.FONT
}

func (w *ChildWindow) Init() {
	w.Window.SetWindowHandler(w)
	w.Window.Create()
	w.Window.Map()
}

func (w *ChildWindow) HandleWindowEvent(ev events.Event) bool {
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
		errPanic(err, "ChildWindow => CreateGC1()")
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
			"child win",
		)
	}
	return true
}
