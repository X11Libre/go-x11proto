package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type CrossingEvent struct {
	GenericEvent
	Type        base.CARD8
	Timestamp   base.CARD32
	RootWindow  base.WINDOW
	EventWindow base.WINDOW
	ChildWindow base.WINDOW
	RootX       base.CARD16
	RootY       base.CARD16
	EventX      base.CARD16
	EventY      base.CARD16
	State       base.CARD16
	Mode        base.CARD8
	Flags       base.CARD8 // BOOL
}

func (e *CrossingEvent) ReceiverWindow() base.WINDOW {
	return e.EventWindow
}

func (e *CrossingEvent) Parse(rbuf base.ReadBuffer) error {
	e.Type = e.GenericEvent.Detail
	e.Timestamp = rbuf.CARD32()
	e.RootWindow = rbuf.XID()
	e.EventWindow = rbuf.XID()
	e.ChildWindow = rbuf.XID()
	e.RootX = rbuf.CARD16()
	e.RootY = rbuf.CARD16()
	e.EventX = rbuf.CARD16()
	e.EventY = rbuf.CARD16()
	e.State = rbuf.CARD16()
	e.Mode = rbuf.CARD8()
	e.Flags = rbuf.CARD8()
	return rbuf.LastError
}

type EnterEvent struct {
	CrossingEvent
}

type LeaveEvent struct {
	CrossingEvent
}

func ParseEvent_EnterNotify(gev GenericEvent, rbuf base.ReadBuffer) (*EnterEvent, error) {
	ev := EnterEvent{CrossingEvent: CrossingEvent{GenericEvent: gev}}
	err := ev.Parse(rbuf)
	return &ev, err
}

func ParseEvent_LeaveNotify(gev GenericEvent, rbuf base.ReadBuffer) (*LeaveEvent, error) {
	ev := LeaveEvent{CrossingEvent: CrossingEvent{GenericEvent: gev}}
	err := ev.Parse(rbuf)
	return &ev, err
}
