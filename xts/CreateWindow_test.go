package xts

import (
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	"testing"
)

func TestCreateWindow1(t *testing.T) {
	conn := connect(t)
	defer conn.Close()

	window, err := rpc.CreateWindow1(
		conn,
		conn.DefaultRoot(),
		50,
		50,
		500,
		300,
		0xFFFFFF,
	)

	if err != nil {
		t.Fatalf("CreateWindow1 call failed: %s", err)
	}

	if err = rpc.MapWindow(conn, window); err != nil {
		t.Fatalf("MapWindow() call failed: %s", err)
	}

	waitForEvent(t, conn,
		func(t *testing.T, ev events.Event) bool {
			switch ev.(type) {
			case *events.MapEvent:
				t.Logf("received MapNotify: %+v\n", ev)
				return true
			}
			return false
		})
}
