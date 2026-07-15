// Package clipboard implements text transfer over X11 selections (PRIMARY and
// CLIPBOARD). It covers both roles: owning a selection and serving conversion
// requests, and requesting the current owner's text. Targets handled are
// UTF8_STRING, STRING and TARGETS, which is what GTK/Qt and xterm expect.
package clipboard

import (
	"fmt"
	"time"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/atoms"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

const propModeReplace = 0

// Clipboard transfers text for one named selection using a client-owned window.
type Clipboard struct {
	conn *core.X11Conn
	win  base.WINDOW

	sel     base.ATOM // PRIMARY or CLIPBOARD
	utf8    base.ATOM // UTF8_STRING
	targets base.ATOM // TARGETS
	prop    base.ATOM // property used to receive converted data

	text  string
	owned bool

	// OnPaste, if set, is called with the requested text when an asynchronous
	// RequestText completes (driven by the event loop via HandleX11WindowEvent).
	OnPaste func(string)
}

// New creates a Clipboard for the given selection name ("PRIMARY" or
// "CLIPBOARD"), using win to own the selection and receive transfers.
func New(conn *core.X11Conn, win base.WINDOW, selection string) (*Clipboard, error) {
	sel, err := rpc.InternAtom(conn, selection)
	if err != nil {
		return nil, err
	}
	utf8, err := rpc.InternAtom(conn, "UTF8_STRING")
	if err != nil {
		return nil, err
	}
	tgts, err := rpc.InternAtom(conn, "TARGETS")
	if err != nil {
		return nil, err
	}
	prop, err := rpc.InternAtom(conn, "GOX11_CLIPBOARD_RECV")
	if err != nil {
		return nil, err
	}
	return &Clipboard{conn: conn, win: win, sel: sel, utf8: utf8, targets: tgts, prop: prop}, nil
}

// Own takes ownership of the selection and stores text to serve on request.
//
// It does NOT verify the takeover with a GetSelectionOwner round-trip: that
// request's reply is matched on the connection's sequence number, which is
// unreliable here (and a mismatch would make Own return early with c.text
// left empty — serving a blank selection). SetSelectionOwner is sufficient:
// if it succeeds the server routes future requests to us.
func (c *Clipboard) Own(text string) error {
	if err := rpc.SetSelectionOwner(c.conn, c.win, c.sel, 0); err != nil {
		return err
	}
	c.text = text
	c.owned = true
	return nil
}

// Owned reports whether this client currently owns the selection.
func (c *Clipboard) Owned() bool { return c.owned }

// Text returns the text this client is serving (valid while Owned).
func (c *Clipboard) Text() string { return c.text }

// Serve answers a SelectionRequest as the owner: it writes the requested data to
// the requestor's property and replies with a SelectionNotify. Unsupported
// targets are refused (reply property None).
func (c *Clipboard) Serve(req *events.SelectionRequestEvent) error {
	property := req.Property
	if property == 0 { // obsolete client: use the target as the property
		property = req.Target
	}
	refused := false

	switch req.Target {
	case c.utf8, atoms.STRING:
		if err := rpc.ChangeProperty8(c.conn, propModeReplace, req.Requestor,
			property, req.Target, toCARD8([]byte(c.text))); err != nil {
			return err
		}
	case c.targets:
		data := []base.CARD32{base.CARD32(c.targets), base.CARD32(c.utf8), atoms.STRING}
		if err := rpc.ChangeProperty32(c.conn, propModeReplace, req.Requestor,
			property, 4 /*ATOM*/, data); err != nil {
			return err
		}
	default:
		refused = true
		property = 0
	}

	notify := &events.SelectionNotifyEvent{
		Timestamp: req.Timestamp,
		Requestor: req.Requestor,
		Selection: req.Selection,
		Target:    req.Target,
		Property:  property,
	}
	_, err := c.conn.Send(&request.SendEventRequest{
		Destination: req.Requestor,
		EventMask:   0, // selection events go to the requestor regardless of mask
		Event:       notify.Encode(c.conn.BE),
	})
	if err == nil && refused {
		return nil
	}
	return err
}

// RequestText asks the current owner to convert the selection to UTF8_STRING.
// The result arrives asynchronously as a SelectionNotify; deliver events to
// HandleX11WindowEvent and set OnPaste to receive it.
//
// It does not pre-check GetSelectionOwner: an absent owner simply yields a
// SelectionNotify with property None, which HandleX11WindowEvent reports as an
// empty paste. Skipping the round-trip avoids a fragile sequence-matched reply
// (whose mismatch would otherwise make the whole request fail).
func (c *Clipboard) RequestText() (bool, error) {
	return true, rpc.ConvertSelection(c.conn, c.win, c.sel, c.utf8, c.prop, 0)
}

// GetText synchronously fetches the owner's text: it issues the conversion and
// then reads the connection's event channel until the SelectionNotify arrives
// (or timeout). Like xsettings' timestamp probe it consumes events directly, so
// call it outside the application's own event loop. Returns ("", false, nil)
// when there is no owner or the conversion is refused.
func (c *Clipboard) GetText(timeout time.Duration) (string, bool, error) {
	ok, err := c.RequestText()
	if err != nil || !ok {
		return "", false, err
	}
	deadline := time.After(timeout)
	for {
		select {
		case ev := <-c.conn.Events():
			ne, ok := ev.(*events.SelectionNotifyEvent)
			if !ok || ne.Requestor != c.win || ne.Selection != c.sel {
				continue
			}
			if ne.Property == 0 {
				return "", false, nil // refused
			}
			data, err := c.readProperty()
			if err != nil {
				return "", false, err
			}
			return string(data), true, nil
		case <-deadline:
			return "", false, fmt.Errorf("clipboard: timed out waiting for selection")
		}
	}
}

// HandleX11WindowEvent serves requests (as owner), notes lost ownership, and
// completes asynchronous pastes (as requestor). Register it as the window
// handler to use the clipboard from an event loop.
func (c *Clipboard) HandleX11WindowEvent(window base.WINDOW, ev events.Event) bool {
	switch e := ev.(type) {
	case *events.SelectionRequestEvent:
		if e.Owner == c.win && e.Selection == c.sel {
			_ = c.Serve(e)
		}
	case *events.SelectionClearEvent:
		if e.Selection == c.sel {
			c.owned = false
		}
	case *events.SelectionNotifyEvent:
		if e.Requestor == c.win && e.Selection == c.sel && c.OnPaste != nil {
			if e.Property == 0 {
				c.OnPaste("")
				return true
			}
			if data, err := c.readProperty(); err == nil {
				c.OnPaste(string(data))
			}
		}
	}
	return true
}

// readProperty reads (and deletes) the whole receive property.
func (c *Clipboard) readProperty() ([]byte, error) {
	var out []byte
	off := base.CARD32(0)
	for {
		rep, err := rpc.GetProperty(c.conn, true, c.win, c.prop, 0, off, 0x4000)
		if err != nil {
			return nil, err
		}
		out = append(out, rep.Value...)
		if rep.BytesAfter == 0 {
			break
		}
		off += base.CARD32(len(rep.Value) / 4)
	}
	return out, nil
}

// EventMask is the mask a clipboard window should select so it observes the
// property changes used during transfers.
const EventMask = base.CARD32(event_mask.PropertyChange)

func toCARD8(b []byte) []base.CARD8 {
	out := make([]base.CARD8, len(b))
	for i, v := range b {
		out[i] = base.CARD8(v)
	}
	return out
}
