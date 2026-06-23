package xts

import (
	"testing"

	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
)

// TestTkTextView creates a TextView against the live server, loads text and
// draws it, exercising the real GC/font/draw path in both byte orders. The
// editing logic itself is covered by offline unit tests in tk/widget.
func TestTkTextView(t *testing.T) {
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)
	tkp := &tk

	f, err := font.Open(c, "fixed")
	must(t, err, "font.Open")
	defer f.Close(c)

	tv := &tk_widget.TextView{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: tkp},
			X:        0, Y: 0, W: 300, H: 200,
		},
		Font: f,
	}
	must(t, tv.Init(), "TextView.Init")

	tv.SetText("first line\nsecond line\nthird")
	if got := tv.Text(); got != "first line\nsecond line\nthird" {
		t.Errorf("Text = %q", got)
	}
	if tv.LineCount() != 3 {
		t.Errorf("LineCount = %d, want 3", tv.LineCount())
	}
	must(t, tv.Draw(), "TextView.Draw")

	// scrolling API stays in range
	tv.ScrollTo(100)
	if tv.TopLine() < 0 || tv.TopLine() >= tv.LineCount() {
		t.Errorf("TopLine out of range: %d", tv.TopLine())
	}

	must(t, tv.Destroy(), "Destroy")
}
