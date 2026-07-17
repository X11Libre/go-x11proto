package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/atoms"
	"github.com/X11Libre/go-x11proto/proto/core/errorcode"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/setup"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type X11Conn struct {
	// set this to true if we wanna talk in big-endian (unusual)
	BE bool

	// DebugRequests logs every outgoing request (type, seq, fields) to stderr.
	DebugRequests bool

	conn      net.Conn
	writeMu   sync.Mutex
	nextSeq   base.CARD16
	nextID    base.CARD32
	pending   map[base.CARD16]*pendingRequest
	pendingMu sync.RWMutex

	eventCh chan events.Event
	errorCh chan error

	closed atomic.Bool

	Setup          setup.XSetupOK
	AtomCache      map[string]base.ATOM
	windowHandlers map[base.WINDOW]X11WindowEventHandler

	// request-length limit in 4-byte units; raised past 0xffff once the
	// BIG-REQUESTS extension is enabled (bigReqEnabled).
	maxReqUnits   int
	bigReqEnabled bool

	// extension registry: cached QueryExtension results and the registered
	// event/error code ranges (see extension.go).
	extMu      sync.Mutex
	extensions map[string]*Extension
	extEvents  []extEventRange
	extErrors  []extErrorRange
}

type pendingRequest struct {
	replyCh chan base.ReplyReader
	errCh   chan error
	done    chan struct{}
	multi   bool // request produces a series of replies (e.g. ListFontsWithInfo)
}

func NewConn(display_name string, be bool) (*X11Conn, error) {
	return NewConnWithAuth(display_name, be, "", nil, nil)
}

// NewConnWithAuth connects to an X11 display with optional authentication.
//
// Parameters:
//   - display_name: X11 display string (e.g. ":0"); empty reads $DISPLAY
//   - be: true for big-endian wire protocol
//   - authorityPath: explicit path to an Xauthority file; empty falls back
//     to the XAUTHORITY environment variable, then ~/.Xauthority
//   - authProto: explicit auth protocol name (e.g. "MIT-MAGIC-COOKIE-1");
//     if non-nil this overrides any lookup in the authority file
//   - authData: the raw auth token bytes; must be non-nil when authProto is set
//
// When all three auth parameters are empty/nil, the function reads the
// Xauthority file (XAUTHORITY or ~/.Xauthority) and picks the entry matching
// the display.  If no matching entry is found the connection proceeds without
// authentication — the same behaviour as the legacy NewConn().
func NewConnWithAuth(display_name string, be bool, authorityPath string, authProto []byte, authData []byte) (*X11Conn, error) {
	if display_name == "" {
		display_name = os.Getenv("DISPLAY")
	}
	display, err := base.ParseDisplay(display_name)
	if err != nil {
		return nil, MakeX11ConnErrorF("malformed display string: %s - %s", display_name, err)
	}

	// Resolve auth credentials.
	protoName, cookie := resolveAuth(display, authorityPath, authProto, authData)

	conn, err := net.Dial(display.DialInfo())
	if err != nil {
		return nil, MakeX11ConnErrorF("failed to connect to X11 display: %s: %w", display_name, err)
	}

	c := &X11Conn{
		BE:             be,
		conn:           conn,
		pending:        make(map[base.CARD16]*pendingRequest),
		eventCh:        make(chan events.Event, 256),
		errorCh:        make(chan error, 16),
		windowHandlers: make(map[base.WINDOW]X11WindowEventHandler),
		extensions:     make(map[string]*Extension),
		AtomCache: map[string]base.ATOM{
			"STRING":  atoms.STRING,
			"WM_NAME": atoms.WM_NAME,
		},
		nextID: 0,
	}

	if err := c.handshakeWithAuth(protoName, cookie); err != nil {
		conn.Close()
		return nil, c.errorF("X11 handshake failed: %w", err)
	}

	c.maxReqUnits = int(c.Setup.MaxRequestSize)
	if c.maxReqUnits <= 0 {
		c.maxReqUnits = 0xffff
	}
	c.enableBigRequests() // raises maxReqUnits when the extension is present

	return c, nil
}

// resolveAuth determines the auth protocol name and data to use for the
// connection.  Explicit arguments override authority-file lookup.
func resolveAuth(display base.DisplaySpec, authorityPath string, explicitProto []byte, explicitData []byte) (protoName string, cookie []byte) {
	// Explicit auth always wins.
	if explicitProto != nil {
		return string(explicitProto), explicitData
	}

	// Read the authority file and look up matching entry.
	entries, err := base.XauthFileOrEntries(authorityPath)
	if err != nil || len(entries) == 0 {
		return "", nil
	}
	entry := base.LookupXauth(display, entries)
	if entry == nil {
		return "", nil
	}
	return entry.Proto, entry.Data
}

const (
	x11_init_BE = "B\x00\x00\x0B\x00\x00\x00\x00\x00\x00\x00\x00"
	x11_init_LE = "l\x00\x0B\x00\x00\x00\x00\x00\x00\x00\x00\x00"
)

func (c *X11Conn) handshake() error {
	return c.handshakeWithAuth("", nil)
}

// handshakeWithAuth sends the X11 connection setup request with optional
// authentication credentials.  When protoName is empty and cookie is nil,
// the request carries zero-length auth (same as the legacy path).
func (c *X11Conn) handshakeWithAuth(protoName string, cookie []byte) error {
	var setupReq []byte

	if protoName == "" && len(cookie) == 0 {
		// Fast path: original zero-auth request.
		if c.BE {
			setupReq = []byte(x11_init_BE)
		} else {
			setupReq = []byte(x11_init_LE)
		}
	} else {
		setupReq = c.buildAuthSetupRequest(protoName, cookie)
	}

	if _, err := c.conn.Write([]byte(setupReq)); err != nil {
		c.errorF("handshake: write() %w", err)
		return err
	}

	header := make([]byte, 8)
	if _, err := io.ReadFull(c.conn, header); err != nil {
		c.errorF("handshake: io.ReadFull() %w", err)
		return err
	}

	if header[0] == 0 {
		errmsg := make([]byte, header[1])
		if _, err := io.ReadFull(c.conn, errmsg); err != nil {
			return c.errorF("handshake: failed reading error packet: %w", err)
		}
		errmsgstr := string(errmsg)
		return fmt.Errorf("X11 rejected: %s\n", string(errmsgstr))
	}

	if header[0] != 1 {
		return c.errorF("bad handshake status: %d", header[0])
	}

	setup1 := setup.XSetupPrefix{}
	if err := setup1.ParseBytes(header, c.BE); err != nil {
		return c.errorF("failed parsing XSetupPrefix: %w", err)
	}

	additionalLen := setup1.Length * 4
	data := make([]byte, additionalLen)
	if _, err := io.ReadFull(c.conn, data); err != nil {
		return c.errorF("failed reading additional setup packet data: %w", err)
	}

	xsetupok := setup.XSetupOK{}
	if err := xsetupok.ParseBytes(data, c.BE); err != nil {
		return c.errorF("failed parsing OK block of XSetup message: %v", err)
	}

	c.Setup = xsetupok

	go c.readLoop()
	time.Sleep(30 * time.Millisecond)

	return nil
}

// buildAuthSetupRequest constructs the X11 Connection Setup request with
// authentication credentials.  The wire format is (all big-endian unless
// the connection is big-endian, in which case byte-order flag is 'B'):
//
//	Offset  Size  Field
//	0       1     Byte order ('l' or 'B')
//	1       1     Unused (0)
//	2       2     Protocol major version (11)
//	4       2     Protocol minor version (0)
//	6       2     Auth name length
//	8       2     Auth data length
//	10      2     Unused (0)
//	12      p4    Auth name (padded to 4-byte boundary)
//	12+p4   d4    Auth data (padded to 4-byte boundary)
func (c *X11Conn) buildAuthSetupRequest(protoName string, cookie []byte) []byte {
	authName := base.FormatAuthName(protoName)
	authData := base.FormatAuthData(cookie)

	// 12-byte header + padded auth name + padded auth data.
	totalLen := 12 + len(authName) + len(authData)
	req := make([]byte, totalLen)

	if c.BE {
		req[0] = 'B'
	} else {
		req[0] = 'l'
	}
	// req[1] = 0 (unused)

	// Protocol version: 11.0 (big-endian in wire format)
	// Multi-byte fields follow the byte order set by req[0] ('l' or 'B').
	if c.BE {
		req[2], req[3] = 0, 11 // major version 11, big-endian
		req[4], req[5] = 0, 0  // minor version 0
	} else {
		req[2], req[3] = 11, 0 // major version 11, little-endian
		req[4], req[5] = 0, 0  // minor version 0
	}

	// Auth name length — same byte order as the rest of the request.
	nameLen := len(protoName)
	if c.BE {
		req[6] = byte(nameLen >> 8)
		req[7] = byte(nameLen)
	} else {
		req[6] = byte(nameLen)
		req[7] = byte(nameLen >> 8)
	}

	// Auth data length — same byte order.
	dataLen := len(cookie)
	if c.BE {
		req[8] = byte(dataLen >> 8)
		req[9] = byte(dataLen)
	} else {
		req[8] = byte(dataLen)
		req[9] = byte(dataLen >> 8)
	}

	// req[10], req[11] = 0 (unused)

	copy(req[12:], authName)
	copy(req[12+len(authName):], authData)

	return req
}

func (c *X11Conn) readLoop() {
	// readLoop is the sole sender on eventCh/errorCh, so it is also their sole
	// closer: closing them here (after the loop returns) guarantees no send can
	// overlap the close. An external Close() unblocks the io.ReadFull below via
	// c.conn.Close(), so the loop returns and this runs exactly once. readLoop
	// is started at the end of a successful handshake, so any usable connection
	// always has it running to eventually close the channels.
	defer func() {
		close(c.eventCh)
		close(c.errorCh)
	}()

	header := make([]byte, 32)

	for {
		if _, err := io.ReadFull(c.conn, header); err != nil {
			if !errors.Is(err, net.ErrClosed) {
				fmt.Printf("Read error in readLoop: %v\n", err)
			}
			c.closeWithError(c.errorF("readLoop() %w", err))
			return
		}

		switch {
		case header[0] == 0:
			c.handleError(header)
		case header[0] == 1:
			c.handleReply(header)
		default:
			c.handleEvent(header)
		}
	}
}

func (c *X11Conn) convBE16(data []byte) uint16 {
	if c.BE {
		return binary.BigEndian.Uint16(data)
	} else {
		return binary.LittleEndian.Uint16(data)
	}
}

func (c *X11Conn) convBE32(data []byte) uint32 {
	if c.BE {
		return binary.BigEndian.Uint32(data)
	} else {
		return binary.LittleEndian.Uint32(data)
	}
}

func (c *X11Conn) handleError(header []byte) {
	code := header[1]
	seq := c.convBE16(header[2:4])
	badID := c.convBE32(header[4:8])
	minorOpcode := c.convBE16(header[8:10])
	majorOpcode := header[10]

	name := errorcode.Name(code)
	if ext := c.extErrorName(base.CARD8(code)); ext != "" {
		name = ext + " extension error"
	}
	log.Printf("X11 Error: %s (code=%d), seq=%d, opcode=%d.%d, id=%d\n",
		name, code, seq, majorOpcode, minorOpcode, badID)

	var err error = &RequestError{
		Code:        base.CARD8(code),
		Sequence:    base.CARD16(seq),
		MajorOpcode: base.CARD8(majorOpcode),
		MinorOpcode: base.CARD16(minorOpcode),
		BadID:       base.CARD32(badID),
	}

	c.pendingMu.Lock()
	if pr, ok := c.pending[base.CARD16(seq)]; ok {
		select {
		case pr.errCh <- err:
		default:
		}
		delete(c.pending, base.CARD16(seq))
	}
	c.pendingMu.Unlock()
}

func (c *X11Conn) closeWithError(err error) {
	if c.closed.Swap(true) {
		return
	}
	if err != nil {
		log.Printf("X11 connection closed: %v\n", err)
	}
	c.conn.Close()

	c.pendingMu.Lock()
	for _, pr := range c.pending {
		select {
		case pr.errCh <- err:
		default:
		}
		close(pr.done)
	}
	c.pending = nil
	c.pendingMu.Unlock()
	// eventCh/errorCh are NOT closed here: closeWithError can run on a caller's
	// goroutine (via Close()) while readLoop is mid-send on eventCh, which would
	// panic with "send on closed channel". Closing c.conn above unblocks
	// readLoop's io.ReadFull; readLoop is the sole sender and closes the
	// channels itself once it has returned (see readLoop's defer).
}

func (c *X11Conn) Send(req base.Request) (base.CARD16, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	// encode (and validate) before consuming a sequence number, so a rejected
	// request does not desync our request count from the server's.
	b, err := c.encodeRequest(req)
	if err != nil {
		return 0, c.errorF("Send(): %w", err)
	}

	// first sequence number is 1
	if c.nextSeq == 0 {
		c.nextSeq++
	}

	seq := c.nextSeq
	c.nextSeq++

	if c.DebugRequests {
		log.Printf("=> [seq %d] %T %+v", seq, req, req)
	}

	if _, err := c.conn.Write(b); err != nil {
		return 0, c.errorF("Send(): %w", err)
	}
	return seq, nil
}

func (c *X11Conn) SendAndWait(req base.Request) (*base.ReplyReader, error) {
	c.writeMu.Lock()

	// The connection may already be closed (e.g. a ConfigureEvent delivered to
	// the event loop raced with Close(), which nils out c.pending). Return a
	// clean error instead of panicking on "assignment to entry in nil map"
	// below — callers (notably termctl's resize path) must handle a dead
	// connection gracefully rather than take the whole event loop down.
	if c.closed.Load() {
		c.writeMu.Unlock()
		return nil, c.errorF("SendAndWait(): connection closed")
	}

	b, err := c.encodeRequest(req)
	if err != nil {
		c.writeMu.Unlock()
		return nil, c.errorF("SendAndWait(): %w", err)
	}

	// first sequence number is 1
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
	if c.pending == nil {
		// Closed concurrently with the check above: treat as closed.
		c.pendingMu.Unlock()
		c.writeMu.Unlock()
		return nil, c.errorF("SendAndWait(): connection closed")
	}
	c.pending[seq] = pr
	c.pendingMu.Unlock()

	if c.DebugRequests {
		log.Printf("=> [seq %d] %T %+v", seq, req, req)
	}

	if _, err := c.conn.Write(b); err != nil {
		c.removePending(seq)
		c.writeMu.Unlock()
		return nil, c.errorF("SendAndWait(): %w", err)
	}
	c.writeMu.Unlock()

	select {
	case reply := <-pr.replyCh:
		return &reply, nil
	case err := <-pr.errCh:
		return nil, err
	case <-pr.done:
		return nil, c.errorF("SendAndWait(): request cancelled")
	case <-time.After(30 * time.Second):
		c.removePending(seq)
		return nil, c.errorF("SendAndWait(): timeout")
	}
}

// ReplyIterator delivers the successive replies of a multi-reply request (the
// only core example is ListFontsWithInfo). Call Next repeatedly until the
// terminating reply, then Close; the connection cannot know where the series
// ends, so the caller is responsible for closing it.
type ReplyIterator struct {
	c      *X11Conn
	seq    base.CARD16
	pr     *pendingRequest
	closed bool
}

// SendAndIterate sends a request that produces a series of replies and returns
// an iterator over them. The caller must Close the iterator when done.
func (c *X11Conn) SendAndIterate(req base.Request) (*ReplyIterator, error) {
	c.writeMu.Lock()

	b, err := c.encodeRequest(req)
	if err != nil {
		c.writeMu.Unlock()
		return nil, c.errorF("SendAndIterate(): %w", err)
	}

	// first sequence number is 1
	if c.nextSeq == 0 {
		c.nextSeq++
	}
	seq := c.nextSeq
	c.nextSeq++

	pr := &pendingRequest{
		replyCh: make(chan base.ReplyReader, 8),
		errCh:   make(chan error, 1),
		done:    make(chan struct{}),
		multi:   true,
	}

	c.pendingMu.Lock()
	c.pending[seq] = pr
	c.pendingMu.Unlock()

	if c.DebugRequests {
		log.Printf("=> [seq %d] %T %+v (multi)", seq, req, req)
	}

	if _, err := c.conn.Write(b); err != nil {
		c.removePending(seq)
		c.writeMu.Unlock()
		return nil, c.errorF("SendAndIterate(): %w", err)
	}
	c.writeMu.Unlock()

	return &ReplyIterator{c: c, seq: seq, pr: pr}, nil
}

// Next blocks for the next reply in the series (or an error / timeout).
func (it *ReplyIterator) Next() (*base.ReplyReader, error) {
	select {
	case reply := <-it.pr.replyCh:
		return &reply, nil
	case err := <-it.pr.errCh:
		return nil, err
	case <-time.After(30 * time.Second):
		return nil, it.c.errorF("ReplyIterator.Next(): timeout")
	}
}

// Close unregisters the request, releasing the reader loop. Safe to call more
// than once.
func (it *ReplyIterator) Close() {
	if it.closed {
		return
	}
	it.closed = true
	it.c.removePending(it.seq)
}

func (c *X11Conn) handleReply(header []byte) error {
	rb := base.MakeReadBuffer(header, c.BE)

	reply := base.ReplyReader{}
	reply.Type = rb.CARD8()
	reply.Data0 = rb.CARD8()
	reply.Sequence = rb.CARD16()
	reply.Length = rb.CARD32()

	extradatalen := base.UnitsToBytes(reply.Length) // we already had read 32 bytes

	bbuf := bytes.Buffer{}
	bbuf.Write(rb.Binary)

	if extradatalen > 0 {
		extradata := make([]byte, extradatalen)
		if _, err1 := io.ReadFull(c.conn, extradata); err1 != nil {
			return c.errorF("handleReply: failed reading extra data: %s\n", err1)
		}
		bbuf.Write(extradata)
	}

	reply.SetPayload(bbuf.Bytes(), c.BE)

	c.pendingMu.Lock()
	pr, ok := c.pending[reply.Sequence]
	if ok && !pr.multi {
		// one-shot request: consume the pending entry now
		delete(c.pending, reply.Sequence)
	}
	c.pendingMu.Unlock()

	if !ok {
		log.Printf(" --> no pending request for sequence %d\n", reply.Sequence)
		return nil
	}

	if pr.multi {
		// multi-reply request: deliver every reply with backpressure until the
		// iterator is closed (which closes done and releases this send).
		select {
		case pr.replyCh <- reply:
		case <-pr.done:
		}
		return nil
	}

	select {
	case pr.replyCh <- reply:
	default:
	}

	return nil
}

func (c *X11Conn) handleEvent(header []byte) {
	// route to a registered extension event parser if the code is in range,
	// otherwise fall back to the core event parser
	var (
		ev  events.Event
		err error
	)
	if parser := c.extEventParser(base.CARD8(header[0] & 0x7f)); parser != nil {
		ev, err = parser(header, c.BE)
	} else {
		ev, err = events.ParseEvent(header, c.BE)
	}
	if err != nil {
		return
	}
	select {
	case c.eventCh <- ev:
	default:
	}
}

func (c *X11Conn) Events() <-chan events.Event {
	return c.eventCh
}

func (c *X11Conn) removePending(seq base.CARD16) {
	c.pendingMu.Lock()
	if pr, ok := c.pending[seq]; ok {
		close(pr.done)
		delete(c.pending, seq)
	}
	c.pendingMu.Unlock()
}

func (c *X11Conn) NextResourceID() base.XID {
	id := c.nextID
	c.nextID += 1
	id = (c.Setup.RidBase | (id & c.Setup.RidMask))
	return base.XID(id)
}

func (c *X11Conn) Close() {
	c.closeWithError(nil)
}

// encodeRequest serialises req to its wire bytes and validates its length. It
// allocates no sequence number and writes nothing, so a rejection here (e.g. an
// over-long request) leaves the connection untouched — callers must encode
// before allocating a sequence number, or the server's request count would
// drift from ours and desync replies.
func (c *X11Conn) encodeRequest(req base.Request) ([]byte, error) {
	writer := base.MakeRequestWriter(c.BE)
	if err := req.WriteInto(&writer); err != nil {
		return nil, err
	}
	b := writer.ToBytes()
	// The normal 16-bit length field counts 4-byte units; a request over 0xffff
	// units needs the BIG-REQUESTS encoding (and the extension enabled). Anything
	// beyond the effective maximum is refused cleanly (a wrapped length field
	// would desync the connection).
	units := len(b) / 4
	final := units
	big := false
	if units > 0xffff {
		if !c.bigReqEnabled {
			return nil, c.errorF("request %T is %d units; exceeds 65535 and BIG-REQUESTS is unavailable", req, units)
		}
		big = true
		final = units + 1 // the inserted 32-bit length word adds one unit
	}
	if final > c.maxReqUnits {
		return nil, c.errorF("request %T is %d units, exceeds the server maximum of %d", req, final, c.maxReqUnits)
	}
	if big {
		b = encodeBigRequest(b, c.BE)
	}
	return b, nil
}

// encodeBigRequest rewrites a normally-encoded request into BIG-REQUESTS form:
// the 16-bit length is zeroed (the marker) and a 32-bit length in 4-byte units
// (including the inserted word) is spliced in right after it.
func encodeBigRequest(b []byte, be bool) []byte {
	units := uint32(len(b)/4) + 1
	out := make([]byte, len(b)+4)
	out[0], out[1] = b[0], b[1] // opcode + data byte; out[2:4] stay 0 (the marker)
	if be {
		binary.BigEndian.PutUint32(out[4:8], units)
	} else {
		binary.LittleEndian.PutUint32(out[4:8], units)
	}
	copy(out[8:], b[4:])
	return out
}

func (c *X11Conn) DefaultRoot() base.WINDOW {
	return c.Setup.Screens[0].RootWindow
}

func (c *X11Conn) DefaultBlackPixel() base.CARD32 {
	return c.Setup.Screens[0].BlackPixel
}

func (c *X11Conn) DefaultWhitePixel() base.CARD32 {
	return c.Setup.Screens[0].WhitePixel
}

func (c *X11Conn) DeliverWindowEvent(ev events.Event) bool {
	window := ev.ReceiverWindow()

	if handler, ok := c.windowHandlers[window]; ok && handler != nil {
		return handler.HandleX11WindowEvent(window, ev)
	}

	return false
}

// FIXME: should support multiple handlers per window XID and also removing them
func (c *X11Conn) RegisterWindowHandler(window base.WINDOW, handler X11WindowEventHandler) {
	c.windowHandlers[window] = handler
}

func (c *X11Conn) SimpleEventLoop() {
	for ev := range c.Events() {
		c.DeliverWindowEvent(ev)
	}
}

func (c *X11Conn) errorF(format string, args ...any) X11ConnError {
	return X11ConnError{
		X11Error: base.MakeX11ErrorF("X11ConnError: "+format, args...),
		Conn:     c,
	}
}
