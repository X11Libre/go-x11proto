package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/ext/render"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// TestRender exercises the RENDER extension end to end against the server.
func TestRender(t *testing.T) {
	c := connectLE(t)
	defer c.Close()

	rdr, err := render.Query(c)
	if err != nil {
		t.Skipf("RENDER not available: %v", err)
	}

	maj, min, err := rdr.QueryVersion()
	must(t, err, "QueryVersion")
	t.Logf("RENDER version %d.%d", maj, min)

	fmts, err := rdr.QueryPictFormats()
	must(t, err, "QueryPictFormats")
	argb := fmts.FindFormat(32, true)
	if argb == nil {
		t.Fatal("no ARGB32 (depth 32 + alpha) picture format reported")
	}
	if rgb := fmts.FindFormat(24, false); rgb == nil {
		t.Error("no depth-24 direct format reported")
	}

	// ARGB32 picture on a depth-32 pixmap
	const sz = 16
	pm, err := rpc.CreatePixmap(c, 32, c.DefaultRoot(), sz, sz)
	must(t, err, "CreatePixmap")
	pic, err := rdr.CreatePicture(pm, argb.ID, render.PictureValues{
		ValueMask: render.CPRepeat, Repeat: render.RepeatNormal,
	})
	must(t, err, "CreatePicture")

	// fill it opaque red, then read it back to confirm the request round-trips
	must(t, rdr.FillRectangles(render.PictOpSrc, pic,
		render.Color{Red: 0xffff, Alpha: 0xffff},
		[]base.Rectangle{{X: 0, Y: 0, Width: sz, Height: sz}}), "FillRectangles")

	img, err := rpc.GetImage(c, request.ImageFormatZPixmap, pm, 0, 0, sz, sz, 0xFFFFFFFF)
	must(t, err, "GetImage")
	if len(img.Data) < 4 {
		t.Fatalf("GetImage returned %d bytes", len(img.Data))
	}
	// standard ARGB32, little-endian ZPixmap: byte order B,G,R,A
	if img.Data[2] != 0xff || img.Data[3] != 0xff {
		t.Errorf("filled pixel = %v, want red+alpha (B,G,R,A = 0,0,0xff,0xff)", img.Data[:4])
	}

	// composite over a second picture
	pm2, err := rpc.CreatePixmap(c, 32, c.DefaultRoot(), sz, sz)
	must(t, err, "CreatePixmap2")
	pic2, err := rdr.CreatePicture(pm2, argb.ID, render.PictureValues{})
	must(t, err, "CreatePicture2")
	must(t, rdr.Composite(render.PictOpOver, pic, 0, pic2, 0, 0, 0, 0, 0, 0, sz, sz), "Composite")

	must(t, rdr.ChangePicture(pic, render.PictureValues{
		ValueMask: render.CPRepeat, Repeat: render.RepeatPad,
	}), "ChangePicture")

	must(t, rdr.FreePicture(pic), "FreePicture")
	must(t, rdr.FreePicture(pic2), "FreePicture2")
	must(t, rpc.FreePixmap(c, pm), "FreePixmap")
	must(t, rpc.FreePixmap(c, pm2), "FreePixmap2")
}
