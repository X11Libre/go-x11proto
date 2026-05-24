package xts

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"testing"
	"time"
)

func connect(t *testing.T, be bool) *core.X11Conn {
	conn, err := core.NewConn("", be)
	if err != nil {
		t.Fatalf("connect failed: %+v", err)
	}
	return conn
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
