package xts

import (
	"os"
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
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

// TestTkFontTTFBoxDrawing checks that '┌' is hand-drawn against its actual
// cell rectangle (see boxdraw.go) rather than rasterized from the font: ink
// must reach the down and right quadrants (its two arms) but not the
// up-left quadrant (no arms there). This is a regression guard for the
// double-line spacing bug found during development, where strokes landed
// either bunched into a corner or spread past the cell's own bounds.
func TestTkFontTTFBoxDrawing(t *testing.T) {
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
	face, err := ttf.Open(&tk, rdr, testTTFPath, 24, 96)
	must(t, err, "ttf.Open")
	defer face.Close()

	cellH := face.Height()
	cellW := face.Advance('┌')
	if cellW <= 0 || cellH <= 0 {
		t.Fatalf("degenerate cell size %dx%d", cellW, cellH)
	}

	const margin = 10
	w, h := cellW+2*margin, cellH+2*margin
	depth := screen(c).RootDepth
	pm, err := tk.CreatePixmap(depth, c.DefaultRoot(), base.CARD16(w), base.CARD16(h))
	must(t, err, "CreatePixmap")
	defer pm.Free()

	gc, err := tk.CreateGCFor(pm.XID, c.DefaultWhitePixel(), c.DefaultWhitePixel(), 0)
	must(t, err, "CreateGCFor")
	defer gc.Free()
	must(t, pm.FillRect(gc.XID, 0, 0, base.CARD16(w), base.CARD16(h)), "FillRect white")

	fmtID, err := rdr.StandardFormat(depth, false)
	must(t, err, "StandardFormat")
	pic, err := rdr.PictureFor(pm.Drawable, fmtID, tk_render.PictureValues{})
	must(t, err, "PictureFor")
	defer pic.Free()

	cellTop := margin
	baseline := cellTop + face.Ascent()
	if _, err := face.DrawString(pic, margin, baseline, "┌", [3]byte{0, 0, 0}); err != nil {
		t.Fatalf("DrawString: %v", err)
	}

	img, err := rpc.GetImage(c, request.ImageFormatZPixmap, pm.XID, 0, 0, base.CARD16(w), base.CARD16(h), 0xFFFFFFFF)
	must(t, err, "GetImage")

	inkAt := func(px, py int) bool {
		i := (py*w + px) * 4
		if i < 0 || i+2 >= len(img.Data) {
			t.Fatalf("sample point (%d,%d) out of bounds for %dx%d image", px, py, w, h)
		}
		return img.Data[i] != 0xff || img.Data[i+1] != 0xff || img.Data[i+2] != 0xff
	}

	cx, cy := margin+cellW/2, cellTop+cellH/2
	if !inkAt(cx, cy+cellH/4) {
		t.Error("'┌' has no ink below center — down arm missing or too short")
	}
	if !inkAt(cx+cellW/4, cy) {
		t.Error("'┌' has no ink right of center — right arm missing or too short")
	}
	if inkAt(cx-cellW/3, cy-cellH/3) {
		t.Error("'┌' has ink in the up-left quadrant — should be empty (no up/left arms)")
	}
}
