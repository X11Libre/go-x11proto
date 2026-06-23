package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
)

// *font.Font must satisfy the widget text-renderer interface so it can back
// Labels and the editor.
var _ tk_widget.TextRenderer = (*font.Font)(nil)

// TestTkFont opens a server font, checks its metrics, and draws text onto a
// pixmap — verifying via GetImage that glyphs actually land (non-background
// pixels appear).
func TestTkFont(t *testing.T) {
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)

	f, err := font.Open(c, "fixed")
	must(t, err, "font.Open(fixed)")
	defer f.Close(c)

	if f.Height() <= 0 {
		t.Errorf("Height = %d, want > 0", f.Height())
	}
	if w := f.TextWidth("Hello"); w <= 0 {
		t.Errorf("TextWidth(Hello) = %d, want > 0", w)
	}
	// Width must grow with more characters.
	if f.TextWidth("ii") <= f.TextWidth("i") {
		t.Errorf("TextWidth not monotonic: %d !> %d", f.TextWidth("ii"), f.TextWidth("i"))
	}
	// IndexAtX round-trips through TextWidth for a fixed-ish font.
	if i := f.IndexAtX("Hello", 0); i != 0 {
		t.Errorf("IndexAtX(.,0) = %d, want 0", i)
	}
	if i := f.IndexAtX("Hello", f.TextWidth("Hello")+100); i != 5 {
		t.Errorf("IndexAtX past end = %d, want 5", i)
	}

	const w, h = 80, 20
	pm, err := tk.CreatePixmap(screen(c).RootDepth, c.DefaultRoot(), w, h)
	must(t, err, "CreatePixmap")
	defer pm.Free()

	gc, err := tk.CreateGC1(c.DefaultBlackPixel(), c.DefaultWhitePixel(), f.ID)
	must(t, err, "CreateGC1")
	defer gc.Free()

	// white background, then draw black text with a filled background cell
	must(t, pm.FillRect(gc.XID, 0, 0, w, h), "FillRect") // black fill first
	must(t, gc.SetForeground(c.DefaultWhitePixel()), "fg=white")
	must(t, pm.FillRect(gc.XID, 0, 0, w, h), "FillRect white")
	must(t, gc.SetForeground(c.DefaultBlackPixel()), "fg=black")
	must(t, f.DrawText(pm.Drawable, gc.XID, 2, 2, 0, "Hi"), "DrawText")

	img, err := rpc.GetImage(c, request.ImageFormatZPixmap, pm.XID, 0, 0, w, h, 0xFFFFFFFF)
	must(t, err, "GetImage")
	nonWhite := 0
	for _, b := range img.Data {
		if b != 0xff {
			nonWhite++
		}
	}
	if nonWhite == 0 {
		t.Error("DrawText produced no visible pixels")
	}
}
