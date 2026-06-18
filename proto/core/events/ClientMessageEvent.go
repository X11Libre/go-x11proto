package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_code"
)

// ClientMessageEvent is an X11 ClientMessage (event code 33). Data carries the
// 20-byte payload as five 32-bit values - the Format-32 layout used by EWMH /
// ICCCM client messages (e.g. _NET_WM_STATE). Encode serialises it into the
// 32-byte wire form expected by request.SendEventRequest.
type ClientMessageEvent struct {
	GenericEvent
	Window      base.WINDOW
	MessageType base.ATOM // the message_type atom, e.g. _NET_WM_STATE
	Format      byte      // 8, 16 or 32; 0 is treated as 32
	Data        [5]base.CARD32
}

func (e *ClientMessageEvent) ReceiverWindow() base.WINDOW {
	return e.Window
}

// Parse fills the event from rbuf, which is positioned just after the generic
// header (type/detail/sequence). The format byte is carried in the generic
// detail field; the 20-byte payload is read as five 32-bit values.
func (e *ClientMessageEvent) Parse(rbuf base.ReadBuffer) error {
	e.Format = byte(e.Detail)
	e.Window = rbuf.XID()
	e.MessageType = base.ATOM(rbuf.CARD32())
	for i := range e.Data {
		e.Data[i] = rbuf.CARD32()
	}
	return rbuf.LastError
}

func ParseEvent_ClientMessage(gev GenericEvent, rbuf base.ReadBuffer) (*ClientMessageEvent, error) {
	ev := ClientMessageEvent{GenericEvent: gev}
	err := ev.Parse(rbuf)
	return &ev, err
}

// Encode serialises the event into its 32-byte wire representation in the given
// byte order (be = big-endian). Only the Format-32 data layout is emitted.
func (e *ClientMessageEvent) Encode(be bool) [32]byte {
	format := e.Format
	if format == 0 {
		format = 32
	}
	wb := base.MakeWriteBuffer(be)
	wb.WriteCARD8(event_code.ClientMessage)
	wb.WriteCARD8(base.CARD8(format))
	wb.WriteCARD16(0) // sequence number - filled in by the server
	wb.WriteXID(base.XID(e.Window))
	wb.WriteATOM(e.MessageType)
	for _, d := range e.Data {
		wb.WriteCARD32(d)
	}
	var out [32]byte
	copy(out[:], wb.PayloadBytes())
	return out
}
