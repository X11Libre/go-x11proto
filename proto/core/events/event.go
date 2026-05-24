package events

import (
	"fmt"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_code"
	"log"
)

type Event interface {
	GetType() base.CARD8
	ReceiverWindow() base.WINDOW
}

type GenericEvent struct {
	RawType  base.CARD8
	Type     base.CARD8
	Detail   base.CARD8
	Sequence base.CARD16
	IsClient bool
}

func (ev GenericEvent) GetType() base.CARD8 {
	return ev.Type
}

func ParseEvent(data []byte, be bool) (Event, error) {
	rbuf := base.MakeReadBuffer(data, be)

	gev := GenericEvent{
		RawType:  rbuf.CARD8(),
		Detail:   rbuf.CARD8(),
		Sequence: rbuf.CARD16(),
	}
	gev.Type = gev.RawType & 0x7F
	gev.IsClient = gev.RawType != gev.Type

	switch gev.Type {
	case event_code.Expose:
		return ParseEvent_Expose(gev, rbuf)
	case event_code.KeyPress:
		return ParseEvent_KeyPress(gev, rbuf)
	case event_code.KeyRelease:
		return ParseEvent_KeyRelease(gev, rbuf)
	case event_code.ButtonPress:
		return ParseEvent_ButtonPress(gev, rbuf)
	case event_code.ButtonRelease:
		return ParseEvent_ButtonRelease(gev, rbuf)
	case event_code.EnterNotify:
		return ParseEvent_EnterNotify(gev, rbuf)
	case event_code.LeaveNotify:
		return ParseEvent_LeaveNotify(gev, rbuf)
	case event_code.MotionNotify:
		return ParseEvent_MotionNotify(gev, rbuf)
	case event_code.MapNotify:
		return ParseEvent_MapNotify(gev, rbuf)
	case event_code.PropertyNotify:
		return ParseEvent_PropertyNotify(gev, rbuf)
	case event_code.ConfigureNotify:
		return ParseEvent_ConfigureNotify(gev, rbuf)
	case event_code.FocusIn:
		return ParseEvent_FocusInNotify(gev, rbuf)
	case event_code.FocusOut:
		return ParseEvent_FocusOutNotify(gev, rbuf)
	case event_code.ReparentNotify:
		return ParseEvent_ReparentNotify(gev, rbuf)
	case event_code.KeymapNotify:
		return ParseEvent_KeymapNotify(gev, rbuf)
	case event_code.VisibilityNotify:
		return ParseEvent_VisibilityNotify(gev, rbuf)
	case event_code.ColormapNotify:
		return ParseEvent_ColormapNotify(gev, rbuf)
	case event_code.CreateNotify:
		return ParseEvent_CreateNotify(gev, rbuf)
	default:
		log.Printf("unknown event %d\n", gev.Type)
		return nil, fmt.Errorf("unknown event %d", gev.Type)
	}
}
