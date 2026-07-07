// Command fonttest demonstrates antialiased TrueType text on go-x11proto via
// tk/font/ttf, side by side with the existing X core bitmap font: it draws
// the same string twice, once through each, so the difference is visible
// directly. See tk/font/ttf's package doc for how the antialiased path
// works (pure-Go rasterization + RENDER compositing, no cgo/libfreetype).
package main

import (
	"log"

	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_font "github.com/X11Libre/go-x11proto/tk/font"
	"github.com/X11Libre/go-x11proto/tk/font/ttf"
	tk_render "github.com/X11Libre/go-x11proto/tk/render"
)

// ttfPath is a system-installed monospace font, good enough for this
// comparison; a real terminal/editor backend will need embedding or
// fontconfig discovery instead of a hardcoded path.
const ttfPath = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"

const sampleText = "The quick Fox 0123 ┌─┐ ╔═╗"

type win struct {
	tk_core.Window
	winPic   *tk_render.Picture
	coreFont *tk_font.Font
	coreGC   *tk_core.GC
	aa       *ttf.Face
}

func (w *win) HandleWindowEvent(ev events.Event) bool {
	if _, ok := ev.(*events.ExposeEvent); ok {
		_ = w.draw()
	}
	return true
}

func (w *win) draw() error {
	if err := w.Window.FillRect(w.coreGC.XID, 0, 0, 600, 160); err != nil {
		return err
	}
	if err := w.coreFont.SetOn(w.coreGC); err != nil {
		return err
	}
	if err := w.Window.Drawable.ImageText8(w.coreGC.XID, 10, 40, "core bitmap font: "+sampleText); err != nil {
		return err
	}
	_, err := w.aa.DrawString(w.winPic, 10, 110, "AA TrueType font: "+sampleText, [3]byte{0x10, 0x10, 0x10})
	return err
}

func main() {
	conn, err := proto.DialBE("")
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()
	tkc := tk_core.MakeTkConn(conn)

	rdr, err := tk_render.Open(&tkc)
	if err != nil {
		log.Fatalf("RENDER extension unavailable: %v", err)
	}

	aa, err := ttf.Open(&tkc, rdr, ttfPath, 18, 96)
	if err != nil {
		log.Fatalf("open TrueType face: %v", err)
	}
	defer aa.Close()

	coreFont, err := tk_font.Open(conn, "fixed")
	if err != nil {
		log.Fatalf("open core font: %v", err)
	}
	coreGC, err := tkc.CreateGC1(tkc.X11Conn.DefaultBlackPixel(), tkc.X11Conn.DefaultWhitePixel(), 0)
	if err != nil {
		log.Fatalf("create GC: %v", err)
	}

	w := &win{
		Window: tk_core.Window{
			Drawable:  tk_core.Drawable{Conn: &tkc},
			Parent:    tkc.GetRoot(),
			Name:      "fonttest: core bitmap vs antialiased TrueType",
			X:         50,
			Y:         50,
			W:         600,
			H:         160,
			EventMask: 0x8000, // ExposureMask
		},
		coreFont: coreFont,
		coreGC:   coreGC,
		aa:       aa,
	}
	w.Window.SetWindowHandler(w)
	if err := w.Window.Create(); err != nil {
		log.Fatalf("create window: %v", err)
	}

	// CreatePicture (like CreateWindow/CreatePixmap) allocates its XID
	// client-side and has no reply, so a format/depth mismatch surfaces only
	// as an async X error, never as this call's return value — the format
	// must be picked correctly up front from the window's real depth, not
	// guessed and "corrected" after the fact.
	geom, err := w.Window.GetGeometry()
	if err != nil {
		log.Fatalf("GetGeometry: %v", err)
	}
	winFmt, err := rdr.StandardFormat(geom.Depth, false)
	if err != nil {
		log.Fatalf("no picture format for window depth %d: %v", geom.Depth, err)
	}
	w.winPic, err = rdr.PictureFor(w.Window.Drawable, winFmt, tk_render.PictureValues{})
	if err != nil {
		log.Fatalf("picture for window: %v", err)
	}

	w.Window.Map()

	conn.SimpleEventLoop()
}
