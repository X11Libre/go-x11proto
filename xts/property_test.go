package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/atoms"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

func TestAtoms(t *testing.T) {
	c := connectLE(t)
	defer c.Close()

	a, err := rpc.InternAtom(c, "GO_X11_TEST_ATOM")
	must(t, err, "InternAtom")
	if a == 0 {
		t.Fatal("InternAtom returned 0")
	}
	name, err := rpc.GetAtomName(c, a)
	must(t, err, "GetAtomName")
	if name != "GO_X11_TEST_ATOM" {
		t.Errorf("GetAtomName = %q, want GO_X11_TEST_ATOM", name)
	}

	// only-if-exists on a name that does not exist must return None (0)
	reply, err := c.SendAndWait(&request.InternAtomRequest{OnlyIfExist: true, Name: "GO_X11_NO_SUCH_ATOM_42"})
	must(t, err, "InternAtom(onlyIfExists)")
	rep := request.InternAtomReply{}
	must(t, rep.Parse(*reply), "parse InternAtom reply")
	if rep.Atom != 0 {
		t.Errorf("only-if-exists for missing atom = %d, want 0", rep.Atom)
	}
}

func TestProperties(t *testing.T) {
	c := connectLE(t)
	defer c.Close()
	w := createWin(t, c, request.CW_EVENT_MASK, &request.CreateWindowRequest{Width: 10, Height: 10})

	prop, err := rpc.InternAtom(c, "GO_X11_PROP")
	must(t, err, "InternAtom")
	must(t, rpc.ChangePropertyString(c, 0, w, prop, "hello"), "ChangeProperty(string)")

	r, err := rpc.GetProperty(c, false, w, prop, 0, 0, 100)
	must(t, err, "GetProperty")
	if r.Format != 8 || string(r.Value) != "hello" {
		t.Errorf("GetProperty = (fmt %d, %q), want (8, hello)", r.Format, string(r.Value))
	}

	// 32-bit property
	prop32, _ := rpc.InternAtom(c, "GO_X11_PROP32")
	must(t, rpc.ChangeProperty32(c, 0, w, prop32, base.ATOM(6) /*CARDINAL*/, []base.CARD32{10, 20, 30}), "ChangeProperty32")
	r32, err := rpc.GetProperty(c, false, w, prop32, 0, 0, 100)
	must(t, err, "GetProperty32")
	if r32.Format != 32 || r32.ValueLen != 3 {
		t.Errorf("GetProperty32 = (fmt %d, len %d), want (32, 3)", r32.Format, r32.ValueLen)
	}

	props, err := rpc.ListProperties(c, w)
	must(t, err, "ListProperties")
	if !containsAtom(props, prop) {
		t.Errorf("ListProperties %v missing %d", props, prop)
	}
	must(t, rpc.DeleteProperty(c, w, prop), "DeleteProperty")
	must(t, rpc.RotateProperties(c, w, 1, []base.ATOM{prop32}), "RotateProperties")
	_ = atoms.STRING
	must(t, rpc.DestroyWindow(c, w), "DestroyWindow")
}

func containsAtom(s []base.ATOM, a base.ATOM) bool {
	for _, x := range s {
		if x == a {
			return true
		}
	}
	return false
}

func TestSelections(t *testing.T) {
	c := connectLE(t)
	defer c.Close()
	w := createWin(t, c, request.CW_EVENT_MASK, &request.CreateWindowRequest{Width: 10, Height: 10})
	sel, _ := rpc.InternAtom(c, "GO_X11_SEL")

	must(t, rpc.SetSelectionOwner(c, w, sel, 0), "SetSelectionOwner")
	owner, err := rpc.GetSelectionOwner(c, sel)
	must(t, err, "GetSelectionOwner")
	if owner != w {
		t.Errorf("selection owner = %d, want %d", owner, w)
	}
	must(t, rpc.SetSelectionOwner(c, 0, sel, 0), "SetSelectionOwner(release)")
	must(t, rpc.DestroyWindow(c, w), "DestroyWindow")
}
