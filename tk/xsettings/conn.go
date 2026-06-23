package xsettings

import (
	"fmt"
	"time"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/atoms"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

const (
	propModeReplace = 0
	propModeAppend  = 2
)

func selectionName(screen int) string { return fmt.Sprintf("_XSETTINGS_S%d", screen) }

// Client reads the XSETTINGS published for a screen.
type Client struct {
	conn       *core.X11Conn
	selAtom    base.ATOM
	propAtom   base.ATOM
	managerWin base.WINDOW
	onChange   func(*Settings)
}

// NewClient interns the atoms for the given screen's XSETTINGS.
func NewClient(conn *core.X11Conn, screen int) (*Client, error) {
	sel, err := rpc.InternAtom(conn, selectionName(screen))
	if err != nil {
		return nil, err
	}
	prop, err := rpc.InternAtom(conn, "_XSETTINGS_SETTINGS")
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, selAtom: sel, propAtom: prop}, nil
}

// ManagerWindow returns the current settings-manager window, or 0 if none runs.
func (c *Client) ManagerWindow() (base.WINDOW, error) {
	return rpc.GetSelectionOwner(c.conn, c.selAtom)
}

// Get reads and decodes the current settings. It returns (nil, nil) when no
// settings manager is running.
func (c *Client) Get() (*Settings, error) {
	owner, err := c.ManagerWindow()
	if err != nil {
		return nil, err
	}
	if owner == 0 {
		return nil, nil
	}
	data, err := c.readProperty(owner)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	return decode(data)
}

// Watch calls onChange whenever the settings change (and once now if a manager
// is present). It selects PropertyChange on the manager window and registers a
// handler, so the application must run the connection's event loop
// (SimpleEventLoop / DeliverWindowEvent). It watches the manager that is running
// at the time of the call; appearance of a new manager later is not tracked.
func (c *Client) Watch(onChange func(*Settings)) error {
	c.onChange = onChange
	owner, err := c.ManagerWindow()
	if err != nil {
		return err
	}
	if owner == 0 {
		return nil // no manager yet, nothing to watch
	}
	c.managerWin = owner
	if err := rpc.ChangeWindowAttributes(c.conn, &request.ChangeWindowAttributesRequest{
		Window:    owner,
		ValueMask: request.CW_EVENT_MASK,
		EventMask: base.CARD32(event_mask.PropertyChange),
	}); err != nil {
		return err
	}
	c.conn.RegisterWindowHandler(owner, c)
	if s, err := c.Get(); err == nil && s != nil && onChange != nil {
		onChange(s)
	}
	return nil
}

// HandleX11WindowEvent re-reads the settings and fires the Watch callback when
// the manager's settings property changes.
func (c *Client) HandleX11WindowEvent(window base.WINDOW, ev events.Event) bool {
	pe, ok := ev.(*events.PropertyEvent)
	if !ok || pe.Window != c.managerWin || base.ATOM(pe.Atom) != c.propAtom || pe.Deleted {
		return true
	}
	if s, err := c.Get(); err == nil && s != nil && c.onChange != nil {
		c.onChange(s)
	}
	return true
}

// readProperty fetches the whole _XSETTINGS_SETTINGS property (chunked).
func (c *Client) readProperty(owner base.WINDOW) ([]byte, error) {
	var out []byte
	off := base.CARD32(0)
	for {
		// type = AnyPropertyType (0); length in 4-byte units.
		rep, err := rpc.GetProperty(c.conn, false, owner, c.propAtom, 0, off, 0x4000)
		if err != nil {
			return nil, err
		}
		out = append(out, rep.Value...)
		if rep.BytesAfter == 0 {
			break
		}
		off += base.CARD32(len(rep.Value) / 4)
	}
	return out, nil
}

// Manager owns the XSETTINGS selection for a screen and publishes settings.
type Manager struct {
	conn     *core.X11Conn
	win      base.WINDOW
	selAtom  base.ATOM
	propAtom base.ATOM
	mgrAtom  base.ATOM
	time     base.CARD32 // timestamp used to acquire the selection
	serial   uint32
}

// NewManager acquires the _XSETTINGS_S<screen> selection (creating a window to
// own it) and announces itself via the MANAGER client message. It fails if
// another manager already owns the selection.
//
// The selection is taken with CurrentTime; that is good enough for a single
// owner but is technically discouraged by the ICCCM (a real timestamp from an
// event is preferable if managers may contend).
func NewManager(conn *core.X11Conn, screen int) (*Manager, error) {
	sel, err := rpc.InternAtom(conn, selectionName(screen))
	if err != nil {
		return nil, err
	}
	prop, err := rpc.InternAtom(conn, "_XSETTINGS_SETTINGS")
	if err != nil {
		return nil, err
	}
	mgr, err := rpc.InternAtom(conn, "MANAGER")
	if err != nil {
		return nil, err
	}

	if owner, err := rpc.GetSelectionOwner(conn, sel); err != nil {
		return nil, err
	} else if owner != 0 {
		return nil, fmt.Errorf("xsettings: a manager already owns %s", selectionName(screen))
	}

	win, err := rpc.CreateWindow1(conn, conn.DefaultRoot(), -10, -10, 1, 1,
		base.CARD32(event_mask.PropertyChange))
	if err != nil {
		return nil, err
	}

	m := &Manager{conn: conn, win: win, selAtom: sel, propAtom: prop, mgrAtom: mgr}
	m.time = m.serverTimestamp() // a real timestamp; 0 (CurrentTime) on fallback

	if err := rpc.SetSelectionOwner(conn, win, sel, m.time); err != nil {
		_ = rpc.DestroyWindow(conn, win)
		return nil, err
	}
	if owner, err := rpc.GetSelectionOwner(conn, sel); err != nil || owner != win {
		_ = rpc.DestroyWindow(conn, win)
		return nil, fmt.Errorf("xsettings: failed to acquire %s", selectionName(screen))
	}

	if err := m.announce(); err != nil {
		return nil, err
	}
	return m, nil
}

// serverTimestamp obtains a current server time the proper way: a zero-length
// property append on our window generates a PropertyNotify carrying the time.
// It briefly reads the connection's event channel, so NewManager must be called
// before the application starts its own event loop. Returns 0 (CurrentTime) if
// no notify arrives in time.
func (m *Manager) serverTimestamp() base.CARD32 {
	if err := rpc.ChangeProperty8(m.conn, propModeAppend, m.win, m.propAtom, atoms.STRING, nil); err != nil {
		return 0
	}
	timeout := time.After(2 * time.Second)
	for {
		select {
		case ev := <-m.conn.Events():
			if pe, ok := ev.(*events.PropertyEvent); ok && pe.Window == m.win {
				return pe.Timestamp
			}
		case <-timeout:
			return 0
		}
	}
}

// Window is the manager's selection-owner window.
func (m *Manager) Window() base.WINDOW { return m.win }

// announce broadcasts the MANAGER client message so existing clients notice the
// new settings manager (ICCCM manager-selection convention).
func (m *Manager) announce() error {
	root := m.conn.DefaultRoot()
	ev := events.ClientMessageEvent{
		Window:      root,
		MessageType: m.mgrAtom,
		Format:      32,
		Data:        [5]base.CARD32{m.time, base.CARD32(m.selAtom), base.CARD32(m.win), 0, 0},
	}
	_, err := m.conn.Send(&request.SendEventRequest{
		Propagate:   false,
		Destination: root,
		EventMask:   base.CARD32(event_mask.StructureNotify),
		Event:       ev.Encode(m.conn.BE),
	})
	return err
}

// Set publishes items as the complete settings, bumping the serial so clients
// re-read. Items without a LastChange get the current serial.
func (m *Manager) Set(items []Setting) error {
	for i := range items {
		if items[i].LastChange == 0 {
			items[i].LastChange = m.serial
		}
	}
	data := encode(m.serial, items, m.conn.BE)
	m.serial++
	return rpc.ChangeProperty8(m.conn, propModeReplace, m.win, m.propAtom, m.propAtom, toCARD8(data))
}

// Close releases the selection and destroys the manager window.
func (m *Manager) Close() error {
	_ = rpc.SetSelectionOwner(m.conn, 0, m.selAtom, 0)
	return rpc.DestroyWindow(m.conn, m.win)
}

func toCARD8(b []byte) []base.CARD8 {
	out := make([]base.CARD8, len(b))
	for i, v := range b {
		out[i] = base.CARD8(v)
	}
	return out
}
