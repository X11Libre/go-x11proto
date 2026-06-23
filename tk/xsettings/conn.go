package xsettings

import (
	"fmt"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

const propModeReplace = 0

func selectionName(screen int) string { return fmt.Sprintf("_XSETTINGS_S%d", screen) }

// Client reads the XSETTINGS published for a screen.
type Client struct {
	conn     *core.X11Conn
	selAtom  base.ATOM
	propAtom base.ATOM
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

	if err := rpc.SetSelectionOwner(conn, win, sel, 0); err != nil {
		_ = rpc.DestroyWindow(conn, win)
		return nil, err
	}
	if owner, err := rpc.GetSelectionOwner(conn, sel); err != nil || owner != win {
		_ = rpc.DestroyWindow(conn, win)
		return nil, fmt.Errorf("xsettings: failed to acquire %s", selectionName(screen))
	}

	m := &Manager{conn: conn, win: win, selAtom: sel, propAtom: prop, mgrAtom: mgr}
	if err := m.announce(); err != nil {
		return nil, err
	}
	return m, nil
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
		Data:        [5]base.CARD32{0, base.CARD32(m.selAtom), base.CARD32(m.win), 0, 0},
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
