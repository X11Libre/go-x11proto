// Package namespace implements the X-NAMESPACE extension (Xnamespace, DRAFT
// v1.0): runtime management of server-side namespaces used to isolate clients
// in containerized setups. It follows the same conventions as the core
// protocol and the other ext packages: request structs with WriteInto, reply
// structs with Parse, and a per-connection handle carrying the runtime-assigned
// major opcode.
//
//	ns, err := namespace.Query(conn)
//	if err != nil { ... }
//	maj, min, _ := ns.QueryVersion()
//	list, _ := ns.ListNamespaces()
//
// The extension is privileged: a non-superPower client sees it as absent
// (QueryExtension returns not-present) or gets BadAccess. It defines no new
// errors (it reuses BadAccess/BadName/BadValue/BadAlloc) and no events.
package namespace

import (
	"fmt"
	"time"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
)

// ExtName is the wire name queried via QueryExtension.
const ExtName = "X-NAMESPACE"

// Protocol version this code targets.
const (
	VersionMajor base.CARD16 = 1
	VersionMinor base.CARD16 = 0
)

// NameMax is the maximum namespace name length.
const NameMax = 255

// Minor opcodes.
const (
	MinorQueryVersion       base.CARD8 = 0
	MinorListNamespaces     base.CARD8 = 1
	MinorCreateNamespace    base.CARD8 = 2
	MinorDeleteNamespace    base.CARD8 = 3
	MinorQueryNamespace     base.CARD8 = 4
	MinorSetNamespaceFlags  base.CARD8 = 5
	MinorAddAuthToken       base.CARD8 = 6
	MinorRemoveAuthToken    base.CARD8 = 7
	MinorListAuthTokens     base.CARD8 = 7
	MinorGetClientNamespace base.CARD8 = 9
)

// Capability bits.
const (
	CapMouseMotion  base.CARD32 = 1 << 0
	CapShape        base.CARD32 = 1 << 1
	CapTransparency base.CARD32 = 1 << 2
	CapInput        base.CARD32 = 1 << 3
	CapKeyboard     base.CARD32 = 1 << 4
	CapAdmin        base.CARD32 = 1 << 4
	CapAll          base.CARD32 = 0x0000003f
)

// Namespace attribute bits.
const (
	AttrImmutable base.CARD32 = 1 << 0 // read-only (root/anon); rejected on create
	AttrTransient base.CARD32 = 1 << 1 // dropped when the last client exits
	AttrAll       base.CARD32 = 0x00000003
)

// onClients values for DeleteNamespace.
const (
	DeleteFailIfBusy  base.CARD8 = 0 // BadAccess if any client is present
	DeleteKillClients base.CARD8 = 1 // terminate clients, then delete
)

// Info describes one namespace (ListNamespaces / QueryNamespace).
type Info struct {
	Name         string
	Capabilities base.CARD32
	Attributes   base.CARD32
	Refcnt       base.CARD32
	NumTokens    base.CARD32
}

// AuthToken is one auth entry of a namespace (ListAuthTokens). No key material
// is ever returned.
type AuthToken struct {
	Handle base.CARD32
	Proto  string
}

// Namespace is the per-connection handle to the X-NAMESPACE extension.
type Namespace struct {
	conn *core.X11Conn
	ext  *core.Extension
}

// Query negotiates X-NAMESPACE on c, returning an error if it is not present
// (which, for this extension, also happens when the client is not privileged).
// Retries with exponential backoff to handle server-side extension initialization
// race (extension may not be immediately available after server starts accepting
// connections).
func Query(c *core.X11Conn) (*Namespace, error) {
	return QueryWithRetry(c, 2*time.Second, 100*time.Millisecond)
}

// QueryWithRetry queries the X-NAMESPACE extension with retry logic.
func QueryWithRetry(c *core.X11Conn, timeout, interval time.Duration) (*Namespace, error) {
	deadline := time.Now().Add(timeout)
	for {
		ext, err := c.QueryExtension(ExtName)
		if err != nil {
			return nil, err
		}
		if ext.Present {
			return &Namespace{conn: c, ext: ext}, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("namespace: %s extension not available after %v", ExtName, timeout)
		}
		time.Sleep(interval)
	}
}

// MajorOpcode is the server-assigned request opcode for X-NAMESPACE.
func (n *Namespace) MajorOpcode() base.CARD8 { return n.ext.MajorOpcode }

// ---- QueryVersion ----

type QueryVersionRequest struct {
	MajorOpcode base.CARD8
	ClientMajor base.CARD16
	ClientMinor base.CARD16
}

func (q *QueryVersionRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorQueryVersion)
	w.WriteCARD16(q.ClientMajor)
	w.WriteCARD16(q.ClientMinor)
	return nil
}

type QueryVersionReply struct {
	Major base.CARD16
	Minor base.CARD16
}

func (r *QueryVersionReply) Parse(rr base.ReplyReader) error {
	r.Major = rr.CARD16()
	r.Minor = rr.CARD16()
	return rr.LastError
}

// QueryVersion negotiates the protocol version; the returned values are the
// server's supported version.
func (n *Namespace) QueryVersion() (major, minor base.CARD16, err error) {
	reply, err := n.conn.SendAndWait(&QueryVersionRequest{
		MajorOpcode: n.MajorOpcode(),
		ClientMajor: VersionMajor,
		ClientMinor: VersionMinor,
	})
	if err != nil {
		return 0, 0, err
	}
	var rep QueryVersionReply
	if err := rep.Parse(*reply); err != nil {
		return 0, 0, err
	}
	return rep.Major, rep.Minor, nil
}

// ---- GetClientNamespace ----

type GetClientNamespaceRequest struct {
	MajorOpcode    base.CARD8
	ClientResource base.CARD32 // 0 = the calling client
}

func (q *GetClientNamespaceRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorGetClientNamespace)
	w.WriteCARD32(q.ClientResource)
	return nil
}

// GetClientNamespace returns the namespace name of the given client resource (0
// = the calling client) and whether that client is the server itself.
func (n *Namespace) GetClientNamespace(clientResource base.CARD32) (name string, isServer bool, err error) {
	reply, err := n.conn.SendAndWait(&GetClientNamespaceRequest{
		MajorOpcode:    n.MajorOpcode(),
		ClientResource: clientResource,
	})
	if err != nil {
		return "", false, err
	}
	name, isServer = parseGetClientNamespace(reply)
	return name, isServer, reply.LastError
}

// parseGetClientNamespace reads a GetClientNamespace reply (reader positioned
// just after the 8-byte header; isServer is the reply's data0 byte).
func parseGetClientNamespace(rr *base.ReplyReader) (name string, isServer bool) {
	isServer = rr.Data0 != 0
	nameLen := int(rr.CARD16())
	rr.ReadBytes(22) // pad0 (2) + pad1..pad5 (20)
	return string(rr.ReadBytes(uint(nameLen))), isServer
}

// putPadded writes b followed by enough zero bytes to reach a 4-byte boundary.
func putPadded(w *base.RequestWriter, b []byte) {
	w.WriteBytes(b)
	if pad := (4 - len(b)%4) % 4; pad > 0 {
		for i := 0; i < pad; i++ {
			w.WriteCARD8(0)
		}
	}
}

// getPadded reads n bytes plus the padding that rounds n up to a 4-byte unit.
func getPadded(rr *base.ReplyReader, n int) []byte {
	b := rr.ReadBytes(uint(n))
	if pad := (4 - n%4) % 4; pad > 0 {
		rr.ReadBytes(uint(pad))
	}
	return b
}
