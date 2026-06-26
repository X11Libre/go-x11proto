package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/atoms"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// TestConnCloseEventRace exercises the close-vs-send race on the connection's
// event channel. Each iteration floods the server with property changes on a
// window that selected PropertyChangeMask; every change echoes back a
// PropertyNotify that readLoop forwards onto eventCh. The events are not
// drained, so the buffer fills and readLoop is actively sending when Close() is
// called concurrently.
//
// Before the fix (Close() closed eventCh while readLoop still sent on it) this
// panics with "send on closed channel" and trips the race detector. After it
// (readLoop is the sole closer, closing eventCh only once it has returned) it is
// clean. Run under `go test -race`.
func TestConnCloseEventRace(t *testing.T) {
	for i := 0; i < 40; i++ {
		c := connect(t)

		w := createWin(t, c, request.CW_EVENT_MASK,
			&request.CreateWindowRequest{Width: 1, Height: 1,
				EventMask: event_mask.PropertyChange})

		for j := 0; j < 400; j++ {
			// PropModeReplace = 0; each change generates a PropertyNotify.
			_ = rpc.ChangeProperty8(c, 0, w, atoms.WM_NAME, atoms.STRING,
				[]base.CARD8{base.CARD8(j)})
		}

		// Close while PropertyNotify deliveries are still in flight on readLoop.
		c.Close()
	}
}
