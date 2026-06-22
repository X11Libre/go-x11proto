package core

import (
	"time"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

// CheckRequest sends req and returns the X protocol error it generated, or nil
// if it succeeded. It is the way to check requests that have no reply: req is
// followed by a GetInputFocus round-trip, so once that reply arrives the server
// has processed req and any error has been delivered (X processes and reports in
// order). Requests that do have a reply work too; their error is reported the
// same way and a successful reply is discarded.
//
// On success the returned error is nil; on a protocol error it is a
// *RequestError whose Code can be compared against proto/core/errorcode.
func (c *X11Conn) CheckRequest(req base.Request) error {
	c.writeMu.Lock()
	if c.nextSeq == 0 {
		c.nextSeq++
	}
	seq := c.nextSeq
	c.nextSeq++
	pr := &pendingRequest{
		replyCh: make(chan base.ReplyReader, 1),
		errCh:   make(chan error, 1),
		done:    make(chan struct{}),
	}
	c.pendingMu.Lock()
	c.pending[seq] = pr
	c.pendingMu.Unlock()

	if err := c.writeRequest(req, seq); err != nil {
		c.removePending(seq)
		c.writeMu.Unlock()
		return c.errorF("CheckRequest(): %w", err)
	}

	// round-trip barrier so we know req has been fully processed
	bseq := c.nextSeq
	c.nextSeq++
	bpr := &pendingRequest{
		replyCh: make(chan base.ReplyReader, 1),
		errCh:   make(chan error, 1),
		done:    make(chan struct{}),
	}
	c.pendingMu.Lock()
	c.pending[bseq] = bpr
	c.pendingMu.Unlock()

	if err := c.writeRequest(&request.GetInputFocusRequest{}, bseq); err != nil {
		c.removePending(seq)
		c.removePending(bseq)
		c.writeMu.Unlock()
		return c.errorF("CheckRequest(): %w", err)
	}
	c.writeMu.Unlock()

	select {
	case <-bpr.replyCh:
	case <-bpr.errCh:
		// the barrier itself should not error; fall through and report req's
	case <-bpr.done:
		c.removePending(seq)
		return c.errorF("CheckRequest(): request cancelled")
	case <-time.After(30 * time.Second):
		c.removePending(seq)
		c.removePending(bseq)
		return c.errorF("CheckRequest(): timeout")
	}

	// the barrier reply means req was processed before it (readLoop is in order),
	// so any error for req is already in pr.errCh.
	c.removePending(seq)
	select {
	case err := <-pr.errCh:
		return err
	default:
		return nil
	}
}
