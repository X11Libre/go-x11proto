package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

func TestColors(t *testing.T) {
	c := connect(t)
	defer c.Close()
	cmap := screen(c).Colormap

	col, err := rpc.AllocColor(c, cmap, 0xFFFF, 0, 0)
	must(t, err, "AllocColor")

	q, err := rpc.QueryColors(c, cmap, []base.CARD32{col.Pixel})
	must(t, err, "QueryColors")
	if len(q) != 1 {
		t.Errorf("QueryColors returned %d colors, want 1", len(q))
	}
	must(t, rpc.FreeColors(c, cmap, 0, []base.CARD32{col.Pixel}), "FreeColors")
}

func TestColormapLifecycle(t *testing.T) {
	c := connect(t)
	defer c.Close()
	mid, err := rpc.CreateColormap(c, request.ColormapAllocNone, c.DefaultRoot(), screen(c).RootVisual)
	must(t, err, "CreateColormap")
	must(t, rpc.InstallColormap(c, mid), "InstallColormap")
	maps, err := rpc.ListInstalledColormaps(c, c.DefaultRoot())
	must(t, err, "ListInstalledColormaps")
	if len(maps) == 0 {
		t.Error("ListInstalledColormaps returned none")
	}
	must(t, rpc.UninstallColormap(c, mid), "UninstallColormap")
	must(t, rpc.FreeColormap(c, mid), "FreeColormap")
}
