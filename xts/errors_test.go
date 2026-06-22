package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/errorcode"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// badOpcodeRequest is a request carrying an unassigned major opcode, used to
// provoke a BadRequest from the server.
type badOpcodeRequest struct{}

func (badOpcodeRequest) WriteInto(w *base.RequestWriter) error {
	w.SetOpcode(200) // well above any core/extension opcode
	return nil
}

// TestProtocolErrors mirrors XTS's error-condition coverage: for the
// reproducible semantic errors, drive a minimal bad request and assert the
// exact X error code. (XTS's BadLength purposes send truncated requests at the
// wire level; the high-level API always emits correct lengths, so those are not
// reproducible here.)
func TestProtocolErrors(t *testing.T) {
	c := connect(t)
	defer c.Close()
	root := c.DefaultRoot()

	// a syntactically valid but unallocated id: a resource argument referring to
	// it yields the matching Bad<Resource> error.
	unused := func() base.XID { return c.NextResourceID() }

	// --- bad resource references ---
	expectError(t, c, &request.GetWindowAttributesRequest{Window: base.WINDOW(unused())},
		errorcode.BadWindow, "GetWindowAttributes(unused id)")
	expectError(t, c, &request.GetGeometryRequest{Drawable: base.DRAWABLE(unused())},
		errorcode.BadDrawable, "GetGeometry(unused id)")
	expectError(t, c, &request.FreePixmapRequest{Pixmap: base.PIXMAP(unused())},
		errorcode.BadPixmap, "FreePixmap(unused id)")
	expectError(t, c, &request.FreeGCRequest{Gc: base.GC(unused())},
		errorcode.BadGC, "FreeGC(unused id)")
	expectError(t, c, &request.FreeColormapRequest{Colormap: base.COLORMAP(unused())},
		errorcode.BadColor, "FreeColormap(unused id)")
	expectError(t, c, &request.FreeCursorRequest{Cursor: base.CURSOR(unused())},
		errorcode.BadCursor, "FreeCursor(unused id)")
	expectError(t, c, &request.CloseFontRequest{Font: base.FONT(unused())},
		errorcode.BadFont, "CloseFont(non-font id)")
	expectError(t, c, &request.GetAtomNameRequest{Atom: 0},
		errorcode.BadAtom, "GetAtomName(atom 0)")

	// --- value / match errors ---
	expectError(t, c, &request.GetImageRequest{Format: 99, Drawable: root, Width: 1, Height: 1, PlaneMask: 0xFFFFFFFF},
		errorcode.BadValue, "GetImage(bad format)")

	pm, err := rpc.CreatePixmap(c, screen(c).RootDepth, root, 8, 8)
	must(t, err, "CreatePixmap")
	expectError(t, c, &request.GetImageRequest{
		Format: request.ImageFormatZPixmap, Drawable: pm,
		X: 0, Y: 0, Width: 100, Height: 100, PlaneMask: 0xFFFFFFFF,
	}, errorcode.BadMatch, "GetImage(rect exceeds drawable)")

	// --- bad id choice: reuse an id already in use ---
	expectError(t, c, &request.CreatePixmapRequest{
		Depth: screen(c).RootDepth, Pid: pm, Drawable: root, Width: 8, Height: 8,
	}, errorcode.BadIDChoice, "CreatePixmap(reused id)")

	must(t, rpc.FreePixmap(c, pm), "FreePixmap cleanup")

	// --- unknown request opcode ---
	expectError(t, c, badOpcodeRequest{}, errorcode.BadRequest, "request with opcode 200")
}
