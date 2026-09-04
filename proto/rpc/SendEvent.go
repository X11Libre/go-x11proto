package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

// SendEvent delivers the given 32-byte event to the destination window. With
// propagate false the event is sent only to dest; the event mask is used to
// pick overflow propagation targets. protocol extensions (ICCCM/EWMH and
// XEmbed) send ClientMessages this way. Pass requests.NoEventMask (0) for the
// event mask when targeting a specific window directly.
//
// The Event must be the fully encoded 32-byte wire form; use
// events.ClientMessage...Encode to build it from a structured event.
func SendEvent(c *core.X11Conn, propagate bool, dest base.WINDOW, eventMask base.CARD32, event [32]byte) error {
	_, err := c.Send(&request.SendEventRequest{
		Propagate:   propagate,
		Destination: dest,
		EventMask:   eventMask,
		Event:       event,
	})
	return err
}
