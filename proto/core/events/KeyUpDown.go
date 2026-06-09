package events

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

/*
KeyPress, KeyRelease, ButtonPress, ButtonRelease all have the
same physical layout
*/
type KeyUpDownEvent struct {
	GenericEvent
	Key         base.CARD8
	Timestamp   base.CARD32
	RootWindow  base.WINDOW
	EventWindow base.WINDOW
	ChildWindow base.WINDOW
	RootX       base.CARD16
	RootY       base.CARD16
	EventX      base.CARD16
	EventY      base.CARD16
	State       base.CARD16
	SameScreen  base.CARD8
}

func (e *KeyUpDownEvent) ReceiverWindow() base.WINDOW {
	return e.EventWindow
}

func (e *KeyUpDownEvent) Parse(rbuf base.ReadBuffer) error {
	e.Key = e.GenericEvent.Detail
	e.Timestamp = rbuf.CARD32()
	e.RootWindow = rbuf.XID()
	e.EventWindow = rbuf.XID()
	e.ChildWindow = rbuf.XID()
	e.RootX = rbuf.CARD16()
	e.RootY = rbuf.CARD16()
	e.EventX = rbuf.CARD16()
	e.EventY = rbuf.CARD16()
	e.State = rbuf.CARD16()
	e.SameScreen = rbuf.CARD8()
	rbuf.CARD8() // ignored
	return rbuf.LastError
}

type KeyPressEvent struct {
	KeyUpDownEvent
}

type KeyReleaseEvent struct {
	KeyUpDownEvent
}

type ButtonPressEvent struct {
	KeyUpDownEvent
}

type ButtonReleaseEvent struct {
	KeyUpDownEvent
}

func ParseEvent_KeyPress(gev GenericEvent, rbuf base.ReadBuffer) (*KeyPressEvent, error) {
	ev := KeyPressEvent{KeyUpDownEvent: KeyUpDownEvent{GenericEvent: gev}}
	err := ev.Parse(rbuf)
	return &ev, err
}

func ParseEvent_KeyRelease(gev GenericEvent, rbuf base.ReadBuffer) (*KeyReleaseEvent, error) {
	ev := KeyReleaseEvent{KeyUpDownEvent: KeyUpDownEvent{GenericEvent: gev}}
	err := ev.Parse(rbuf)
	return &ev, err
}

func ParseEvent_ButtonPress(gev GenericEvent, rbuf base.ReadBuffer) (*ButtonPressEvent, error) {
	ev := ButtonPressEvent{KeyUpDownEvent: KeyUpDownEvent{GenericEvent: gev}}
	err := ev.Parse(rbuf)
	return &ev, err
}

func ParseEvent_ButtonRelease(gev GenericEvent, rbuf base.ReadBuffer) (*ButtonReleaseEvent, error) {
	ev := ButtonReleaseEvent{KeyUpDownEvent: KeyUpDownEvent{GenericEvent: gev}}
	err := ev.Parse(rbuf)
	return &ev, err
}
