// Package xembed implements the XEmbed protocol (freedesktop.org/xembed-spec)
// used to embed one X client's window inside another client's window, as a
// reparenting + ClientMessage + property handshake. It covers both sides:
//
//   - the embedder (e.g. a tab manager or panel) that re-parents a client
//     window into its own and tells the client it is now embedded, and
//   - the embedded client (e.g. a terminal) that announces its embedding
//     support via the _XEMBED_INFO property.
//
// The handshake is intentionally thin on top of the generated X11 protocol:
// the heavy lifting (ReparentWindow, ChangeProperty, SendEvent) lives in
// proto/rpc; this package only defines the wire constants and the small
// helpers a tab manager or embedded client needs.
//
// Protocol notes / wire layout
//
// A ClientMessage of type _XEMBED carries the XEmbed message in Data[2], with
// Data[0] reserved for a server timestamp (CurrentTime = 0 is accepted) and
// Data[1]/Data[3:] used per-message as detail parameters. _XEMBED_INFO is a
// property of two CARD32s: [version, flags].
package xembed

import (
	"fmt"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// Atom names (interned on demand via Intern; see Conn.Intern).
const (
	AtomXEmbed     = "_XEMBED"
	AtomXEmbedInfo = "_XEMBED_INFO"
	atomXaCARDINAL = "CARDINAL" // predefined property type XA_CARDINAL type id 6
)

// XEmbed protocol version we speak.
const Version = 0

// _XEMBED_INFO flags.
const (
	FlagMapped = 1 << 0 // client is mapped / wants to stay mapped when embedded
)

// XEMBED_* message ids (Data[2] of a _XEMBED ClientMessage).
const (
	MsgEmbeddedNotify   = 0
	MsgWindowActivate   = 1
	MsgWindowDeactivate = 2
	MsgRequestFocus     = 3
	MsgFocusIn          = 4
	MsgFocusOut         = 5
	MsgFocusNext        = 6
	MsgFocusPrev        = 7
	// 8-9 unused.
	MsgModalityOn            = 10
	MsgModalityOff           = 11
	MsgRegisterAccelerator   = 12
	MsgUnregisterAccelerator = 13
	MsgActivateAccelerator   = 14
)

// XEMBED_FOCUS_* detail values for FocusIn/FocusOut.
const (
	FocusCurrent = 0
	FocusFirst   = 1
	FocusLast    = 2
)

// Conn wraps a connection with the XEmbed atoms interned once, so repeated
// message sends don't re-run InternAtom.
type Conn struct {
	conn       *core.X11Conn
	at         base.ATOM // _XEMBED
	atInfo     base.ATOM // _XEMBED_INFO
	atCardinal base.ATOM // CARDINAL property type
}

// New interns the XEmbed atoms on conn and returns a Conn ready for use.
func New(conn *core.X11Conn) (*Conn, error) {
	at, err := rpc.InternAtom(conn, AtomXEmbed)
	if err != nil {
		return nil, err
	}
	atInfo, err := rpc.InternAtom(conn, AtomXEmbedInfo)
	if err != nil {
		return nil, err
	}
	atCardinal, err := rpc.InternAtom(conn, atomXaCARDINAL)
	if err != nil {
		return nil, err
	}
	return &Conn{conn: conn, at: at, atInfo: atInfo, atCardinal: atCardinal}, nil
}

// At returns the interned _XEMBED atom.
func (c *Conn) At() base.ATOM { return c.at }

// InfoAt returns the interned _XEMBED_INFO atom.
func (c *Conn) InfoAt() base.ATOM { return c.atInfo }

// sendXEmbed delivers a _XEMBED ClientMessage to the given window.
// data is the full 5-word payload; Data[0] is forced to CurrentTime (0).
func (c *Conn) sendXEmbed(win base.WINDOW, data [5]base.CARD32) error {
	ev := events.ClientMessageEvent{
		Window:      win,
		MessageType: c.at,
		Format:      32,
		Data:        data,
	}
	return rpc.SendEvent(c.conn, false, win, 0, ev.Encode(c.conn.BE))
}

// ---- client side (embedded client announcing itself) ----

// SetInfo writes the client's _XEMBED_INFO property: [Version, flags].
// The embedded client calls this on its own window so an embedder can discover
// that it speaks XEmbed and whether it wants to stay mapped.
func (c *Conn) SetInfo(win base.WINDOW, version base.CARD32, flags base.CARD32) error {
	return rpc.ChangeProperty32(c.conn, propModeReplace, win, c.atInfo, c.atCardinal,
		[]base.CARD32{version, flags})
}

// RequestFocus asks the embedder to give keyboard focus to the embedded
// client's window.
func (c *Conn) RequestFocus(win base.WINDOW, time base.CARD32) error {
	return c.sendXEmbed(win, [5]base.CARD32{time, 0, MsgRequestFocus, 0, 0})
}

// ---- embedder side (the container / tab manager) ----

// Embed re-parents the client window into container, sets its _XEMBED_INFO
// (version 0, any requested flags), and announces completion with
// XEMBED_EMBEDDED_NOTIFY. It then maps the client unless flags FlagMapped is
// requested by the client (in which case the client keeps itself mapped).
//
// The window event mask on the embedded window should include
// event_mask.StructureNotify so the embedder gets reparent/expose notices.
func (c *Conn) Embed(container, client base.WINDOW, x, y base.INT16, w, h base.CARD16, clientFlags base.CARD32) error {
	if err := rpc.ReparentWindow(c.conn, client, container, x, y); err != nil {
		return fmt.Errorf("xembed: reparent: %w", err)
	}
	// Advertise our (embedder-chosen) XEmbed_INFO on the embedded window.
	flags := clientFlags
	if err := c.SetInfo(client, Version, flags); err != nil {
		return fmt.Errorf("xembed: set info: %w", err)
	}
	if w != 0 && h != 0 {
		if err := rpc.ConfigureWindow(c.conn, &request.ConfigureWindowRequest{
			Window:    client,
			ValueMask: request.CONFIG_WINDOW_WIDTH | request.CONFIG_WINDOW_HEIGHT,
			Width:     w,
			Height:    h,
		}); err != nil {
			return fmt.Errorf("xembed: resize: %w", err)
		}
	}
	// Tell the embedded client the embedding is complete.
	if err := c.sendXEmbed(container, [5]base.CARD32{0, base.CARD32(client), MsgEmbeddedNotify, Version, 0}); err != nil {
		return fmt.Errorf("xembed: embedded-notify: %w", err)
	}
	// Respect the client's desire to be visible; otherwise we (the embedder)
	// control mapping.
	if flags&FlagMapped == 0 {
		if err := rpc.MapWindow(c.conn, client); err != nil {
			return fmt.Errorf("xembed: map: %w", err)
		}
	}
	return nil
}

// Activate tells the embedded client that its embedder window was activated.
func (c *Conn) Activate(container base.WINDOW, client base.WINDOW, time base.CARD32) error {
	return c.sendXEmbed(container, [5]base.CARD32{time, base.CARD32(client), MsgWindowActivate, 0, 0})
}

// Deactivate tells the embedded client that its embedder window was
// deactivated.
func (c *Conn) Deactivate(container base.WINDOW, client base.WINDOW, time base.CARD32) error {
	return c.sendXEmbed(container, [5]base.CARD32{time, base.CARD32(client), MsgWindowDeactivate, 0, 0})
}

// FocusIn tells the embedded client that keyboard focus has moved into it
// (detail: FocusCurrent, FocusFirst or FocusLast).
func (c *Conn) FocusIn(container base.WINDOW, client base.WINDOW, time base.CARD32, detail base.CARD32) error {
	return c.sendXEmbed(container, [5]base.CARD32{time, base.CARD32(client), MsgFocusIn, detail, 0})
}

// FocusOut tells the embedded client that keyboard focus is leaving it.
func (c *Conn) FocusOut(container base.WINDOW, client base.WINDOW, time base.CARD32) error {
	return c.sendXEmbed(container, [5]base.CARD32{time, base.CARD32(client), MsgFocusOut, 0, 0})
}

// ---- parsing incoming ClientMessages ----

// Message unpacks a received _XEMBED ClientMessage into its id and detail
// fields. ok is false when ev is not a _XEMBED ClientMessage.
func (c *Conn) Message(ev events.Event) (id, detail1, detail2 base.CARD32, ok bool) {
	cm, isCM := ev.(*events.ClientMessageEvent)
	if !isCM || cm.MessageType != c.at {
		return 0, 0, 0, false
	}
	return cm.Data[2], cm.Data[3], cm.Data[4], true
}

// property-replace mode used by ChangeProperty.
const propModeReplace = 0

// ContainerEventMask is the event mask an embedder should select on its
// container window to see embedded-child lifecycle events (reparent/expose,
// child map/unmap, and the embedded client's property changes, which is how a
// client signals it wants focus/mapping).
const ContainerEventMask = event_mask.StructureNotify | event_mask.SubstructureNotify | event_mask.PropertyChange
