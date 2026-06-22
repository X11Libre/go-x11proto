package xts

import (
	"errors"
	"testing"
	"time"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/errorcode"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/setup"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// expectError sends req and asserts the server rejects it with the given X
// error code (see proto/core/errorcode).
func expectError(t *testing.T, c *core.X11Conn, req base.Request, code byte, what string) {
	t.Helper()
	err := c.CheckRequest(req)
	if err == nil {
		t.Errorf("%s: expected %s, got success", what, errorcode.Name(code))
		return
	}
	var re *core.RequestError
	if !errors.As(err, &re) {
		t.Errorf("%s: expected %s, got non-protocol error: %v", what, errorcode.Name(code), err)
		return
	}
	if byte(re.Code) != code {
		t.Errorf("%s: expected %s, got %s", what, errorcode.Name(code), errorcode.Name(byte(re.Code)))
	}
}

func connect(t *testing.T, be bool) *core.X11Conn {
	// retry a few times: a freshly started Xvfb (or a busy one) occasionally
	// drops a connection during the handshake.
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		var conn *core.X11Conn
		conn, err = core.NewConn("", be)
		if err == nil {
			return conn
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("connect failed after retries: %+v", err)
	return nil
}

func connectLE(t *testing.T) *core.X11Conn {
	return connect(t, false)
}

func connectBE(t *testing.T) *core.X11Conn {
	return connect(t, true)
}

func waitForEvent(t *testing.T, conn *core.X11Conn, chk func(*testing.T, events.Event) bool) {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	evChan := conn.Events()
	for {
		select {
		case ev, ok := <-evChan:
			if !ok {
				t.Fatalf("error reading events")
			}

			if chk(t, ev) {
				timer.Stop()
				return
			}

		case <-timer.C:
			t.Fatalf("Timeout!\n")
		}
	}
}

func must(t *testing.T, err error, what string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

func screen(c *core.X11Conn) *setup.XSetupScreen {
	return &c.Setup.Screens[0]
}

// newGC creates a graphics context (white on black) on the root drawable.
func newGC(t *testing.T, c *core.X11Conn) base.GC {
	t.Helper()
	gc, err := rpc.CreateGC1(c, c.DefaultWhitePixel(), c.DefaultBlackPixel(), 0)
	must(t, err, "CreateGC1")
	return gc
}

// newPixmap creates a pixmap of the root depth.
func newPixmap(t *testing.T, c *core.X11Conn, w, h base.CARD16) base.PIXMAP {
	t.Helper()
	pm, err := rpc.CreatePixmap(c, screen(c).RootDepth, c.DefaultRoot(), w, h)
	must(t, err, "CreatePixmap")
	return pm
}
