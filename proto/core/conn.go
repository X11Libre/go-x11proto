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
}

type pendingRequest struct {
	replyCh chan base.ReplyReader
	errCh   chan error
	done    chan struct{}
}

func NewConn(display_name string, be bool) (*X11Conn, error) {
	if display_name == "" {
		display_name = os.Getenv("DISPLAY")
	}
	display, err := base.ParseDisplay(display_name)
	if err != nil {
		return nil, MakeX11ConnErrorF("malformed display string: %s - %s", display_name, err)
	}
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
		AtomCache: map[string]base.ATOM{
			"STRING":  atoms.STRING,
			"WM_NAME": atoms.WM_NAME,
		},
		nextID: 0,
	}

	if err := c.handshake(); err != nil {
		conn.Close()
		return nil, c.errorF("X11 handshake failed: %w", err)
	}

	return c, nil
}

const (
	x11_init_BE = "B\x00\x00\x0B\x00\x00\x00\x00\x00\x00\x00\x00"
	x11_init_LE = "l\x00\x0B\x00\x00\x00\x00\x00\x00\x00\x00\x00"
)

func (c *X11Conn) handshake() error {
	setupReq := x11_init_LE
	if c.BE {
		setupReq = x11_init_BE
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

func (c *X11Conn) readLoop() {
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

	log.Printf("X11 Error: %s (code=%d), seq=%d, opcode=%d.%d, id=%d\n",
		errorcode.Name(code), code, seq, majorOpcode, minorOpcode, badID)

	err := c.errorF("x11 error code %d", code)

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
	close(c.eventCh)
	close(c.errorCh)
}

func (c *X11Conn) Send(req base.Request) (base.CARD16, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	// first sequence number is 1
	if c.nextSeq == 0 {
		c.nextSeq++
	}

	seq := c.nextSeq
	c.nextSeq++

	if c.DebugRequests {
		log.Printf("=> [seq %d] %T %+v", seq, req, req)
	}

	if err := c.writeRequest(req, seq); err != nil {
		return 0, c.errorF("Send(): %w", err)
	}
	return seq, nil
}

func (c *X11Conn) SendAndWait(req base.Request) (*base.ReplyReader, error) {
	c.writeMu.Lock()

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
	c.pending[seq] = pr
	c.pendingMu.Unlock()

	if c.DebugRequests {
		log.Printf("=> [seq %d] %T %+v", seq, req, req)
	}

	if err := c.writeRequest(req, seq); err != nil {
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
	if !ok {
		log.Printf(" --> no pending request for sequence %d\n", reply.Sequence)
		c.pendingMu.Unlock()
		return nil
	}
	delete(c.pending, reply.Sequence)
	c.pendingMu.Unlock()

	select {
	case pr.replyCh <- reply:
	default:
	}

	return nil
}

func (c *X11Conn) handleEvent(header []byte) {
	ev, err := events.ParseEvent(header, c.BE)
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

func (c *X11Conn) writeRequest(req base.Request, seq base.CARD16) error {
	writer := base.MakeRequestWriter(c.BE)
	if err := req.WriteInto(&writer); err != nil {
		return err
	}

	//	log.Printf("REQUEST %T bytes: %+v\n", req, writer.ToBytes())
	_, err := c.conn.Write(writer.ToBytes())
	return err
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

	log.Printf("could not deliver event - no listener on window %d (%0x)\n", window, window)
	log.Printf(" event: %T %+v\n", ev, ev)
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
