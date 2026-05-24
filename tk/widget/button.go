package widget

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

type Button struct {
	tk_core.Window
	Gc_up         base.GC
	Gc_down       base.GC
	Font          base.FONT
	OnButtonPress func()
}

func (w *Button) Init() {
	w.Window.SetWindowHandler(w)
	w.Window.Create()
	w.Window.Map()
}

func (w *Button) Repaint(down bool) {
	if down {
		w.PutText8(
			w.Gc_down,
			10,
			15,
			w.Window.Name)
	} else {
		w.PutText8(
			w.Gc_up,
			10,
			15,
			w.Window.Name)
	}
}

func (w *Button) HandleWindowEvent(ev events.Event) bool {
	if w.Font == 0 {
		fontid, err := w.Window.Conn.GetFont("fixed")
		if err != nil {
			panic(err)
		}
		w.Font = fontid
	}

	if w.Gc_up == 0 {
		gcid, _ := rpc.CreateGC1(
			w.Window.Conn.X11Conn,
			w.Window.Conn.X11Conn.DefaultBlackPixel(),
			w.Window.Conn.X11Conn.DefaultWhitePixel(),
			w.Font,
		)
		w.Gc_up = gcid
	}

	if w.Gc_down == 0 {
		gcid, _ := rpc.CreateGC1(
			w.Window.Conn.X11Conn,
			0xFF0000,
			w.Window.Conn.X11Conn.DefaultWhitePixel(),
			w.Font,
		)
		w.Gc_down = gcid
	}

	switch ev.(type) {
	case *events.ExposeEvent:
		w.Repaint(false)
	case *events.ButtonPressEvent:
		w.Repaint(true)
	case *events.ButtonReleaseEvent:
		w.Repaint(false)
		if w.OnButtonPress != nil {
			w.OnButtonPress()
		}
	}
	return true
}
