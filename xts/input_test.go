package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

func TestPointerKeyboardQueries(t *testing.T) {
	c := connect(t)
	defer c.Close()
	root := c.DefaultRoot()

	qp, err := rpc.QueryPointer(c, root)
	must(t, err, "QueryPointer")
	if qp.Root != root {
		t.Errorf("QueryPointer root = %d, want %d", qp.Root, root)
	}
	if _, err := rpc.GetPointerControl(c); err != nil {
		t.Errorf("GetPointerControl: %v", err)
	}
	if _, err := rpc.GetKeyboardControl(c); err != nil {
		t.Errorf("GetKeyboardControl: %v", err)
	}
	km, err := rpc.GetKeyboardMapping(c, c.Setup.MinKeycode, c.Setup.MaxKeycode-c.Setup.MinKeycode+1)
	must(t, err, "GetKeyboardMapping")
	if km.KeysymsPerKeycode == 0 {
		t.Error("GetKeyboardMapping: KeysymsPerKeycode is 0")
	}
	if _, err := rpc.GetModifierMapping(c); err != nil {
		t.Errorf("GetModifierMapping: %v", err)
	}
	if _, err := rpc.GetPointerMapping(c); err != nil {
		t.Errorf("GetPointerMapping: %v", err)
	}
	qk, err := rpc.QueryKeymap(c)
	must(t, err, "QueryKeymap")
	if len(qk.Keys) != 32 {
		t.Errorf("QueryKeymap keys = %d bytes, want 32", len(qk.Keys))
	}
	if _, err := rpc.GetScreenSaver(c); err != nil {
		t.Errorf("GetScreenSaver: %v", err)
	}
	if _, err := rpc.QueryBestSize(c, request.BestSizeTile, root, 16, 16); err != nil {
		t.Errorf("QueryBestSize: %v", err)
	}
}

func TestFocusCoordsMisc(t *testing.T) {
	c := connect(t)
	defer c.Close()
	root := c.DefaultRoot()

	must(t, rpc.SetInputFocus(c, request.RevertToPointerRoot, root, 0), "SetInputFocus")
	tc, err := rpc.TranslateCoordinates(c, root, root, 10, 10)
	must(t, err, "TranslateCoordinates")
	if tc.DstX != 10 || tc.DstY != 10 {
		t.Errorf("TranslateCoordinates dst = (%d,%d), want (10,10)", tc.DstX, tc.DstY)
	}
	must(t, rpc.WarpPointer(c, &request.WarpPointerRequest{DstWindow: root, DstX: 5, DstY: 5}), "WarpPointer")
	must(t, rpc.Bell(c, 0), "Bell")
	must(t, rpc.NoOperation(c), "NoOperation")
}

func TestGrabs(t *testing.T) {
	c := connect(t)
	defer c.Close()
	root := c.DefaultRoot()
	w := createWin(t, c, request.CW_EVENT_MASK, &request.CreateWindowRequest{Width: 100, Height: 100})
	must(t, rpc.MapWindow(c, w), "MapWindow")

	gp, err := rpc.GrabPointer(c, &request.GrabPointerRequest{GrabWindow: w,
		PointerMode: request.GrabModeAsync, KeyboardMode: request.GrabModeAsync})
	must(t, err, "GrabPointer")
	if gp.Status != request.GrabStatusSuccess {
		t.Errorf("GrabPointer status = %d, want success", gp.Status)
	}
	must(t, rpc.UngrabPointer(c, 0), "UngrabPointer")

	gk, err := rpc.GrabKeyboard(c, true, w, 0, request.GrabModeAsync, request.GrabModeAsync)
	must(t, err, "GrabKeyboard")
	if gk.Status != request.GrabStatusSuccess {
		t.Errorf("GrabKeyboard status = %d, want success", gk.Status)
	}
	must(t, rpc.UngrabKeyboard(c, 0), "UngrabKeyboard")

	must(t, rpc.GrabButton(c, &request.GrabButtonRequest{GrabWindow: root, Button: 0, Modifiers: 0x8000,
		PointerMode: request.GrabModeAsync, KeyboardMode: request.GrabModeAsync}), "GrabButton")
	must(t, rpc.UngrabButton(c, 0, root, 0x8000), "UngrabButton")
	must(t, rpc.GrabKey(c, &request.GrabKeyRequest{GrabWindow: root, Key: 24,
		PointerMode: request.GrabModeAsync, KeyboardMode: request.GrabModeAsync}), "GrabKey")
	must(t, rpc.UngrabKey(c, 24, root, 0), "UngrabKey")
	must(t, rpc.AllowEvents(c, request.AllowAsyncPointer, 0), "AllowEvents")
	must(t, rpc.GrabServer(c), "GrabServer")
	must(t, rpc.UngrabServer(c), "UngrabServer")
	must(t, rpc.DestroyWindow(c, w), "DestroyWindow")
}

func TestExtensionsAndHosts(t *testing.T) {
	c := connect(t)
	defer c.Close()
	if _, err := rpc.ListExtensions(c); err != nil {
		t.Errorf("ListExtensions: %v", err)
	}
	if ext, err := rpc.QueryExtension(c, "BIG-REQUESTS"); err != nil {
		t.Errorf("QueryExtension: %v", err)
	} else if !ext.Present {
		t.Error("QueryExtension: BIG-REQUESTS reported absent (should be present)")
	}
	if _, err := rpc.ListHosts(c); err != nil {
		t.Errorf("ListHosts: %v", err)
	}
}
