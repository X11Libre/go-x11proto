package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type PropertyEvent struct {
	GenericEvent
	Window    base.WINDOW
	Atom      base.WINDOW
	Timestamp base.CARD32
	Deleted   bool
}

func (e *PropertyEvent) ReceiverWindow() base.WINDOW {
	return e.Window
}

func (e *PropertyEvent) Parse(rbuf base.ReadBuffer) error {
	e.Window = rbuf.XID()
	e.Atom = rbuf.XID()
	e.Timestamp = rbuf.CARD32()
	e.Deleted = rbuf.Bool()
	return rbuf.LastError
}

func ParseEvent_PropertyNotify(gev GenericEvent, rbuf base.ReadBuffer) (*PropertyEvent, error) {
	ev := PropertyEvent{GenericEvent: gev}
	err := ev.Parse(rbuf)
	return &ev, err
}
