package xts

import (
	"strings"
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
)

// wheelEvent builds a synthetic scroll button press (button 4 = up, 5 = down).
func wheelEvent(button base.CARD8) *events.ButtonPressEvent {
	e := &events.ButtonPressEvent{}
	e.Key = button
	return e
}

// TestTkWheelScroll checks that wheel/touchpad scroll buttons (4/5) scroll the
// TextView and the Scrollbar.
func TestTkWheelScroll(t *testing.T) {
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)
	tkp := &tk

	f, err := font.Open(c, "fixed")
	must(t, err, "font.Open")
	defer f.Close(c)

	tv := &tk_widget.TextView{
		Window: tk_core.Window{Drawable: tk_core.Drawable{Conn: tkp}, X: 0, Y: 0, W: 280, H: 120},
		Font:   f,
	}
	must(t, tv.Init(), "TextView.Init")
	tv.SetText(strings.Repeat("line\n", 100))

	if tv.TopLine() != 0 {
		t.Fatalf("initial TopLine = %d", tv.TopLine())
	}
	tv.HandleWindowEvent(wheelEvent(5)) // wheel down
	down := tv.TopLine()
	if down <= 0 {
		t.Errorf("wheel-down did not scroll: TopLine = %d", down)
	}
	tv.HandleWindowEvent(wheelEvent(4)) // wheel up
	if up := tv.TopLine(); up >= down {
		t.Errorf("wheel-up did not scroll back: %d (was %d)", up, down)
	}

	// scrollbar bound to a view: a wheel event over it scrolls via OnScroll
	sb := &tk_widget.Scrollbar{Window: tk_core.Window{
		Drawable: tk_core.Drawable{Conn: tkp}, X: 280, Y: 0, W: 16, H: 120}}
	moved := -1
	sb.OnScroll = func(top int) { moved = top }
	must(t, sb.Init(), "Scrollbar.Init")
	sb.SetRange(100, 8, 0)
	sb.HandleWindowEvent(wheelEvent(5))
	if moved <= 0 {
		t.Errorf("scrollbar wheel-down did not fire OnScroll with a positive top: %d", moved)
	}

	must(t, tv.Destroy(), "tv.Destroy")
	must(t, sb.Destroy(), "sb.Destroy")
}
