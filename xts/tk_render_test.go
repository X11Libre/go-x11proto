package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_render "github.com/X11Libre/go-x11proto/tk/render"
)

// TestTkRender exercises the tk RENDER abstraction end to end.
func TestTkRender(t *testing.T) {
	c := connectLE(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)

	rdr, err := tk_render.Open(&tk)
	if err != nil {
		t.Skipf("RENDER not available: %v", err)
	}
	maj, min, err := rdr.Version()
	must(t, err, "Version")
	t.Logf("RENDER %d.%d", maj, min)

	argb, err := rdr.ARGB32()
	must(t, err, "ARGB32")

	const sz = 16
	pm, err := tk.CreatePixmap(32, c.DefaultRoot(), sz, sz)
	must(t, err, "CreatePixmap")

	pic, err := rdr.PictureFor(pm.Drawable, argb, tk_render.PictureValues{
		ValueMask: tk_render.CPRepeat, Repeat: tk_render.RepeatNormal,
	})
	must(t, err, "PictureFor")

	must(t, pic.FillRect(tk_render.OpSrc, tk_render.Color{Red: 0xffff, Alpha: 0xffff}, 0, 0, sz, sz), "FillRect")

	img, err := rpc.GetImage(c, request.ImageFormatZPixmap, pm.XID, 0, 0, sz, sz, 0xFFFFFFFF)
	must(t, err, "GetImage")
	if len(img.Data) < 4 || img.Data[2] != 0xff || img.Data[3] != 0xff {
		t.Errorf("filled pixel = %v, want B,G,R,A = 0,0,0xff,0xff", img.Data[:min4(img.Data)])
	}

	// composite onto a second picture
	pm2, err := tk.CreatePixmap(32, c.DefaultRoot(), sz, sz)
	must(t, err, "CreatePixmap2")
	pic2, err := rdr.PictureFor(pm2.Drawable, argb, tk_render.PictureValues{})
	must(t, err, "PictureFor2")
	must(t, pic2.Composite(tk_render.OpOver, pic, nil, 0, 0, 0, 0, 0, 0, sz, sz), "Composite")

	must(t, pic.Change(tk_render.PictureValues{ValueMask: tk_render.CPRepeat, Repeat: tk_render.RepeatPad}), "Change")

	must(t, pic.Free(), "Free")
	must(t, pic2.Free(), "Free2")
	must(t, pm.Free(), "pm.Free")
	must(t, pm2.Free(), "pm2.Free")
}

func min4(b []byte) int {
	if len(b) < 4 {
		return len(b)
	}
	return 4
}
