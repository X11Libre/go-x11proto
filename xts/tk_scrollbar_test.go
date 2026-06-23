package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/core/events"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
)

// TestTkScrollbar wires a Scrollbar to a TextView against the live server and
// drives a synthetic click on the lower track, checking the bar scrolls the
// view. Exercises the real draw path in both byte orders; thumb geometry is
// unit-tested offline.
func TestTkScrollbar(t *testing.T) {
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)
	tkp := &tk

	f, err := font.Open(c, "fixed")
	must(t, err, "font.Open")
	defer f.Close(c)

	tv := &tk_widget.TextView{
		Window: tk_core.Window{Drawable: tk_core.Drawable{Conn: tkp}, X: 0, Y: 0, W: 280, H: 200},
		Font:   f,
	}
	must(t, tv.Init(), "TextView.Init")

	lines := ""
	for i := 0; i < 100; i++ {
		lines += "some line of text\n"
	}
	tv.SetText(lines)

	sb := &tk_widget.Scrollbar{
		Window: tk_core.Window{Drawable: tk_core.Drawable{Conn: tkp}, X: 280, Y: 0, W: 16, H: 200},
	}
	sb.OnScroll = func(top int) { tv.ScrollTo(top) }
	must(t, sb.Init(), "Scrollbar.Init")
	sb.SetRange(tv.LineCount(), tv.VisibleLines(), tv.TopLine())

	if tv.TopLine() != 0 {
		t.Fatalf("initial TopLine = %d, want 0", tv.TopLine())
	}

	// Click near the bottom of the track -> page/scroll down.
	press := &events.ButtonPressEvent{}
	press.EventY = 190
	press.Key = 1 // button 1
	sb.HandleWindowEvent(press)

	if tv.TopLine() == 0 {
		t.Error("clicking the lower track did not scroll the view")
	}

	must(t, sb.Destroy(), "Scrollbar.Destroy")
	must(t, tv.Destroy(), "TextView.Destroy")
}
