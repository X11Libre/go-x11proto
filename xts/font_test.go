package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

func TestFontListing(t *testing.T) {
	c := connectLE(t)
	defer c.Close()
	// these must not error even when the server has no font path
	if _, err := rpc.GetFontPath(c); err != nil {
		t.Errorf("GetFontPath: %v", err)
	}
	if _, err := rpc.ListFonts(c, 50, "*"); err != nil {
		t.Errorf("ListFonts: %v", err)
	}
	if _, err := rpc.ListFontsWithInfo(c, 50, "*"); err != nil {
		t.Errorf("ListFontsWithInfo: %v", err)
	}
}

func TestFontOpen(t *testing.T) {
	c := connectLE(t)
	defer c.Close()
	fonts, err := rpc.ListFonts(c, 1, "*")
	must(t, err, "ListFonts")
	if len(fonts) == 0 {
		t.Skip("no fonts available on this server")
	}
	f, err := rpc.OpenFont(c, fonts[0])
	must(t, err, "OpenFont")
	if _, err := rpc.QueryFont(c, f); err != nil {
		t.Errorf("QueryFont: %v", err)
	}
	if _, err := rpc.QueryTextExtents(c, f, []base.CARD16{'H', 'i'}); err != nil {
		t.Errorf("QueryTextExtents: %v", err)
	}
	must(t, rpc.CloseFont(c, f), "CloseFont")
}
