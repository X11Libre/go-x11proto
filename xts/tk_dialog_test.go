package xts

import (
	"os"
	"testing"

	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/dialog"
	"github.com/X11Libre/go-x11proto/tk/font"
)

// TestTkFilePicker creates a FilePicker against the live server, points it at a
// temp directory and draws it, exercising the real GC/font/draw path in both
// byte orders. Navigation logic is unit-tested offline in tk/dialog.
func TestTkFilePicker(t *testing.T) {
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)
	tkp := &tk

	f, err := font.Open(c, "fixed")
	must(t, err, "font.Open")
	defer f.Close(c)

	tmp := t.TempDir()
	must(t, os.WriteFile(tmp+"/readme.txt", []byte("x"), 0o644), "write file")
	must(t, os.Mkdir(tmp+"/sub", 0o755), "mkdir")

	chosen := ""
	fp := &dialog.FilePicker{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: tkp}, X: 0, Y: 0, W: 400, H: 300,
		},
		Font:     f,
		OnAccept: func(path string) { chosen = path },
	}
	must(t, fp.Init(), "FilePicker.Init")
	must(t, fp.Open(tmp), "FilePicker.Open")

	if fp.CurrentDir() != tmp {
		t.Errorf("CurrentDir = %q, want %q", fp.CurrentDir(), tmp)
	}
	must(t, fp.Draw(), "FilePicker.Draw")
	_ = chosen // selection is driven by input events, covered offline

	must(t, fp.Destroy(), "Destroy")
}
