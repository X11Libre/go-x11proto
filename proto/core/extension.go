package core

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

// Extension is the per-connection identity of an X11 extension as returned by
// QueryExtension: the request major opcode assigned to it by this server, and
// the base values its events and errors are numbered from. The numbers are
// connection-specific, so extension code must look them up at runtime (they are
// not fixed by the protocol).
//
// Typical use:
//
//	xt, _ := conn.QueryExtension("XKEYBOARD")
//	if xt.Present {
//	        conn.RegisterExtensionEvents(xt.FirstEvent, NumXkbEvents, parseXkbEvent)
//	        conn.RegisterExtensionErrors(xt.FirstError, NumXkbErrors, "XKB")
//	}
//	// build a request:
//	//   w.SetExtOpcode(xt.MajorOpcode, MINOR_XKB_FOO)
type Extension struct {
	Name        string
	Present     bool
	MajorOpcode base.CARD8
	FirstEvent  base.CARD8
	FirstError  base.CARD8
}

// ExtEventParser turns a raw 32-byte extension event into an events.Event.
// It has the same shape as events.ParseEvent.
type ExtEventParser func(data []byte, be bool) (events.Event, error)

type extEventRange struct {
	first  base.CARD8
	last   base.CARD8
	parser ExtEventParser
}

type extErrorRange struct {
	first base.CARD8
	last  base.CARD8
	name  string
}

// QueryExtension queries the named extension and caches the result for the
// lifetime of the connection. The returned Extension is non-nil even when the
// extension is absent (check Present).
func (c *X11Conn) QueryExtension(name string) (*Extension, error) {
	c.extMu.Lock()
	if ext, ok := c.extensions[name]; ok {
		c.extMu.Unlock()
		return ext, nil
	}
	c.extMu.Unlock()

	reply, err := c.SendAndWait(&request.QueryExtensionRequest{Name: name})
	if err != nil {
		return nil, err
	}
	rep := request.QueryExtensionReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	ext := &Extension{
		Name:        name,
		Present:     rep.Present,
		MajorOpcode: rep.MajorOpcode,
		FirstEvent:  rep.FirstEvent,
		FirstError:  rep.FirstError,
	}

	c.extMu.Lock()
	if c.extensions == nil {
		c.extensions = make(map[string]*Extension)
	}
	c.extensions[name] = ext
	c.extMu.Unlock()
	return ext, nil
}

// RegisterExtensionEvents registers parser for an extension's events, which
// occupy the codes [first, first+count). Events whose code falls in that range
// are decoded with parser instead of the core event parser.
func (c *X11Conn) RegisterExtensionEvents(first base.CARD8, count int, parser ExtEventParser) {
	if count <= 0 || parser == nil {
		return
	}
	c.extMu.Lock()
	c.extEvents = append(c.extEvents, extEventRange{
		first:  first,
		last:   first + base.CARD8(count-1),
		parser: parser,
	})
	c.extMu.Unlock()
}

// RegisterExtensionErrors registers a human-readable name for an extension's
// error codes [first, first+count), used when logging X errors.
func (c *X11Conn) RegisterExtensionErrors(first base.CARD8, count int, name string) {
	if count <= 0 {
		return
	}
	c.extMu.Lock()
	c.extErrors = append(c.extErrors, extErrorRange{
		first: first,
		last:  first + base.CARD8(count-1),
		name:  name,
	})
	c.extMu.Unlock()
}

// extEventParser returns the registered parser for an extension event code, or
// nil if the code is not in any registered range.
func (c *X11Conn) extEventParser(code base.CARD8) ExtEventParser {
	c.extMu.Lock()
	defer c.extMu.Unlock()
	for _, r := range c.extEvents {
		if code >= r.first && code <= r.last {
			return r.parser
		}
	}
	return nil
}

// extErrorName returns the registered extension name for an error code, or ""
// if the code is not in any registered extension range.
func (c *X11Conn) extErrorName(code base.CARD8) string {
	c.extMu.Lock()
	defer c.extMu.Unlock()
	for _, r := range c.extErrors {
		if code >= r.first && code <= r.last {
			return r.name
		}
	}
	return ""
}
