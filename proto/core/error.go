package core

import (
	"fmt"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/errorcode"
)

// RequestError is an X11 protocol error the server reported for a specific
// request. It is delivered to the request's caller (SendAndWait / CheckRequest)
// so callers can inspect the error Code (see proto/core/errorcode).
type RequestError struct {
	Code        base.CARD8  // errorcode.BadWindow, BadValue, ...
	Sequence    base.CARD16 // sequence number of the offending request
	MajorOpcode base.CARD8
	MinorOpcode base.CARD16
	BadID       base.CARD32 // the bad resource id / value, when applicable
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("x11 %s (code %d), seq=%d, opcode=%d.%d, id=%d",
		errorcode.Name(byte(e.Code)), e.Code, e.Sequence, e.MajorOpcode, e.MinorOpcode, e.BadID)
}
