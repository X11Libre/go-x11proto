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

// TestTkFilePickerFloating checks the Floating option makes the picker a
// top-level window (parented to the root) rather than a child.
func TestTkFilePickerFloating(t *testing.T) {
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)
	tkp := &tk

	f, err := font.Open(c, "fixed")
	must(t, err, "font.Open")
	defer f.Close(c)

	// a parent window the picker would normally attach to
	parent := &tk_core.Window{Drawable: tk_core.Drawable{Conn: tkp}, X: 0, Y: 0, W: 200, H: 200}
	must(t, parent.Create(), "parent.Create")

	fp := &dialog.FilePicker{
		Window:   tk_core.Window{Drawable: tk_core.Drawable{Conn: tkp}, Parent: parent, X: 20, Y: 20, W: 400, H: 300},
		Font:     f,
		Floating: true,
		Title:    "Open File",
	}
	must(t, fp.Init(), "FilePicker.Init")

	// despite Parent being set, Floating must root it at the screen
	tree, err := fp.QueryTree()
	must(t, err, "QueryTree")
	if tree.Parent != c.DefaultRoot() {
		t.Errorf("floating picker parent = %d, want root %d", tree.Parent, c.DefaultRoot())
	}

	must(t, os.WriteFile(t.TempDir()+"/x", []byte("x"), 0o644), "tmp") // ensure a readable dir exists
	must(t, fp.Destroy(), "Destroy")
	must(t, parent.Destroy(), "parent.Destroy")
}
