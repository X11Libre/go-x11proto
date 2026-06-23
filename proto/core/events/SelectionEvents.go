package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_code"
)

// SelectionClearEvent (code 29) tells the previous owner of a selection that it
// has lost ownership (a new client called SetSelectionOwner).
type SelectionClearEvent struct {
	GenericEvent
	Timestamp base.CARD32
	Owner     base.WINDOW // the window that lost ownership
	Selection base.ATOM
}

func (e *SelectionClearEvent) ReceiverWindow() base.WINDOW { return e.Owner }

func (e *SelectionClearEvent) Parse(rbuf base.ReadBuffer) error {
	e.Timestamp = rbuf.CARD32()
	e.Owner = rbuf.XID()
	e.Selection = base.ATOM(rbuf.CARD32())
	return rbuf.LastError
}

func ParseEvent_SelectionClear(gev GenericEvent, rbuf base.ReadBuffer) (*SelectionClearEvent, error) {
	ev := SelectionClearEvent{GenericEvent: gev}
	return &ev, ev.Parse(rbuf)
}

// SelectionRequestEvent (code 30) is delivered to a selection owner asking it to
// convert the selection to Target and store the result in Property on the
// Requestor window, then reply with a SelectionNotify.
type SelectionRequestEvent struct {
	GenericEvent
	Timestamp base.CARD32
	Owner     base.WINDOW
	Requestor base.WINDOW
	Selection base.ATOM
	Target    base.ATOM
	Property  base.ATOM
}

func (e *SelectionRequestEvent) ReceiverWindow() base.WINDOW { return e.Owner }

func (e *SelectionRequestEvent) Parse(rbuf base.ReadBuffer) error {
	e.Timestamp = rbuf.CARD32()
	e.Owner = rbuf.XID()
	e.Requestor = rbuf.XID()
	e.Selection = base.ATOM(rbuf.CARD32())
	e.Target = base.ATOM(rbuf.CARD32())
	e.Property = base.ATOM(rbuf.CARD32())
	return rbuf.LastError
}

func ParseEvent_SelectionRequest(gev GenericEvent, rbuf base.ReadBuffer) (*SelectionRequestEvent, error) {
	ev := SelectionRequestEvent{GenericEvent: gev}
	return &ev, ev.Parse(rbuf)
}

// SelectionNotifyEvent (code 31) is sent by an owner (via SendEvent) to a
// requestor when a ConvertSelection has been served. Property is the property
// the data was stored in, or None (0) if the conversion was refused.
type SelectionNotifyEvent struct {
	GenericEvent
	Timestamp base.CARD32
	Requestor base.WINDOW
	Selection base.ATOM
	Target    base.ATOM
	Property  base.ATOM
}

func (e *SelectionNotifyEvent) ReceiverWindow() base.WINDOW { return e.Requestor }

func (e *SelectionNotifyEvent) Parse(rbuf base.ReadBuffer) error {
	e.Timestamp = rbuf.CARD32()
	e.Requestor = rbuf.XID()
	e.Selection = base.ATOM(rbuf.CARD32())
	e.Target = base.ATOM(rbuf.CARD32())
	e.Property = base.ATOM(rbuf.CARD32())
	return rbuf.LastError
}

func ParseEvent_SelectionNotify(gev GenericEvent, rbuf base.ReadBuffer) (*SelectionNotifyEvent, error) {
	ev := SelectionNotifyEvent{GenericEvent: gev}
	return &ev, ev.Parse(rbuf)
}

// Encode serialises the event into its 32-byte wire form for SendEventRequest,
// which is how a selection owner replies to a request.
func (e *SelectionNotifyEvent) Encode(be bool) [32]byte {
	wb := base.MakeWriteBuffer(be)
	wb.WriteCARD8(event_code.SelectionNotify)
	wb.WriteCARD8(0)  // unused
	wb.WriteCARD16(0) // sequence - filled by the server
	wb.WriteCARD32(e.Timestamp)
	wb.WriteXID(base.XID(e.Requestor))
	wb.WriteCARD32(base.CARD32(e.Selection))
	wb.WriteCARD32(base.CARD32(e.Target))
	wb.WriteCARD32(base.CARD32(e.Property))
	var out [32]byte
	copy(out[:], wb.PayloadBytes())
	return out
}
