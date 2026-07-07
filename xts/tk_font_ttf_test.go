package xts

import (
	"os"
	"testing"

	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font/ttf"
	tk_render "github.com/X11Libre/go-x11proto/tk/render"
)

// testTTFPath is a system-installed monospace font. Environments without it
// (minimal CI images) skip rather than fail — this is a round-trip sanity
// check of the rendering technique, not a font-availability test.
const testTTFPath = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"

// TestTkFontTTF opens an antialiased TrueType face and draws text onto a
// pixmap via RENDER — verifying via GetImage that glyphs actually land
// (non-background pixels appear), the same shape of check TestTkFont does
// for the core-font backend.
func TestTkFontTTF(t *testing.T) {
	if _, err := os.Stat(testTTFPath); err != nil {
		t.Skipf("test font not installed: %v", err)
	}
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)

	rdr, err := tk_render.Open(&tk)
	if err != nil {
		t.Skipf("RENDER not available: %v", err)
	}

	face, err := ttf.Open(&tk, rdr, testTTFPath, 14, 96)
	must(t, err, "ttf.Open")
	defer face.Close()

	if face.Height() <= 0 {
		t.Errorf("Height = %d, want > 0", face.Height())
	}
	if adv := face.Advance('M'); adv <= 0 {
		t.Errorf("Advance('M') = %d, want > 0", adv)
	}

	const w, h = 100, 24
	depth := screen(c).RootDepth
	pm, err := tk.CreatePixmap(depth, c.DefaultRoot(), w, h)
	must(t, err, "CreatePixmap")
	defer pm.Free()

	gc, err := tk.CreateGCFor(pm.XID, c.DefaultWhitePixel(), c.DefaultWhitePixel(), 0)
	must(t, err, "CreateGCFor")
	defer gc.Free()
	must(t, pm.FillRect(gc.XID, 0, 0, w, h), "FillRect white")

	fmtID, err := rdr.StandardFormat(depth, false)
	must(t, err, "StandardFormat")
	pic, err := rdr.PictureFor(pm.Drawable, fmtID, tk_render.PictureValues{})
	must(t, err, "PictureFor")
	defer pic.Free()

	if _, err := face.DrawString(pic, 2, 16, "Hi", [3]byte{0, 0, 0}); err != nil {
		t.Fatalf("DrawString: %v", err)
	}

	img, err := rpc.GetImage(c, request.ImageFormatZPixmap, pm.XID, 0, 0, w, h, 0xFFFFFFFF)
	must(t, err, "GetImage")
	nonWhite := 0
	for _, b := range img.Data {
		if b != 0xff {
			nonWhite++
		}
	}
	if nonWhite == 0 {
		t.Error("DrawString produced no visible pixels")
	}

	// Drawing the same rune twice must hit the glyph cache and still
	// composite correctly (not just "not crash").
	before := nonWhite
	if _, err := face.DrawString(pic, 2, 16, "H", [3]byte{0, 0, 0}); err != nil {
		t.Fatalf("DrawString (cached glyph): %v", err)
	}
	img2, err := rpc.GetImage(c, request.ImageFormatZPixmap, pm.XID, 0, 0, w, h, 0xFFFFFFFF)
	must(t, err, "GetImage 2")
	nonWhite2 := 0
	for _, b := range img2.Data {
		if b != 0xff {
			nonWhite2++
		}
	}
	if nonWhite2 < before {
		t.Errorf("re-drawing a cached glyph lost pixels: %d -> %d", before, nonWhite2)
	}
}
