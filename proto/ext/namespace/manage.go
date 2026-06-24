package namespace

import "github.com/X11Libre/go-x11proto/proto/base"

// ---- ListNamespaces ----

type ListNamespacesRequest struct{ MajorOpcode base.CARD8 }

func (q *ListNamespacesRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorListNamespaces)
	return nil
}

// ListNamespaces returns every namespace the server knows.
func (n *Namespace) ListNamespaces() ([]Info, error) {
	reply, err := n.conn.SendAndWait(&ListNamespacesRequest{MajorOpcode: n.MajorOpcode()})
	if err != nil {
		return nil, err
	}
	return parseInfoList(reply), reply.LastError
}

// parseInfoList reads a ListNamespaces reply: count then count NAMESPACEINFO
// records (reader positioned just after the 8-byte header).
func parseInfoList(rr *base.ReplyReader) []Info {
	count := int(rr.CARD32())
	rr.ReadBytes(20) // pad1..pad5
	out := make([]Info, 0, count)
	for i := 0; i < count; i++ {
		var it Info
		it.Capabilities = rr.CARD32()
		it.Attributes = rr.CARD32()
		it.Refcnt = rr.CARD32()
		it.NumTokens = rr.CARD32()
		nameLen := int(rr.CARD16())
		rr.CARD16() // pad
		it.Name = string(getPadded(rr, nameLen))
		out = append(out, it)
	}
	return out
}

// ---- CreateNamespace ----

type CreateNamespaceRequest struct {
	MajorOpcode  base.CARD8
	Capabilities base.CARD32
	Attributes   base.CARD32
	Name         string
}

func (q *CreateNamespaceRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorCreateNamespace)
	w.WriteCARD32(q.Capabilities)
	w.WriteCARD32(q.Attributes)
	w.WriteCARD16(base.CARD16(len(q.Name)))
	w.WriteCARD16(0) // pad0
	putPadded(w, []byte(q.Name))
	return nil
}

// CreateNamespace creates a namespace with the given capabilities and
// attributes (AttrTransient honored; AttrImmutable is rejected by the server).
func (n *Namespace) CreateNamespace(name string, capabilities, attributes base.CARD32) error {
	_, err := n.conn.SendAndWait(&CreateNamespaceRequest{
		MajorOpcode:  n.MajorOpcode(),
		Capabilities: capabilities,
		Attributes:   attributes,
		Name:         name,
	})
	return err
}

// ---- DeleteNamespace ----

type DeleteNamespaceRequest struct {
	MajorOpcode base.CARD8
	OnClients   base.CARD8
	Name        string
}

func (q *DeleteNamespaceRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorDeleteNamespace)
	w.WriteCARD8(q.OnClients)
	w.WriteCARD8(0) // pad0
	w.WriteCARD16(base.CARD16(len(q.Name)))
	putPadded(w, []byte(q.Name))
	return nil
}

// DeleteNamespace deletes a namespace. onClients is DeleteFailIfBusy (BadAccess
// if clients remain) or DeleteKillClients (terminate them first).
func (n *Namespace) DeleteNamespace(name string, onClients base.CARD8) error {
	_, err := n.conn.SendAndWait(&DeleteNamespaceRequest{
		MajorOpcode: n.MajorOpcode(),
		OnClients:   onClients,
		Name:        name,
	})
	return err
}

// ---- QueryNamespace ----

type QueryNamespaceRequest struct {
	MajorOpcode base.CARD8
	Name        string
}

func (q *QueryNamespaceRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorQueryNamespace)
	w.WriteCARD16(base.CARD16(len(q.Name)))
	w.WriteCARD16(0) // pad0
	putPadded(w, []byte(q.Name))
	return nil
}

// QueryNamespace returns the capabilities, attributes, refcount and token count
// of one namespace (Name is echoed from the argument).
func (n *Namespace) QueryNamespace(name string) (Info, error) {
	reply, err := n.conn.SendAndWait(&QueryNamespaceRequest{
		MajorOpcode: n.MajorOpcode(),
		Name:        name,
	})
	if err != nil {
		return Info{}, err
	}
	info := Info{
		Name:         name,
		Capabilities: reply.CARD32(),
		Attributes:   reply.CARD32(),
		Refcnt:       reply.CARD32(),
		NumTokens:    reply.CARD32(),
	}
	return info, reply.LastError
}

// ---- SetNamespaceFlags ----

type SetNamespaceFlagsRequest struct {
	MajorOpcode base.CARD8
	ValueMask   base.CARD32 // which capability bits to apply
	Values      base.CARD32 // new values for the masked bits
	Name        string
}

func (q *SetNamespaceFlagsRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorSetNamespaceFlags)
	w.WriteCARD32(q.ValueMask)
	w.WriteCARD32(q.Values)
	w.WriteCARD16(base.CARD16(len(q.Name)))
	w.WriteCARD16(0) // pad0
	putPadded(w, []byte(q.Name))
	return nil
}

// SetNamespaceFlags changes the masked capability bits of a namespace and
// returns the resulting capabilities. (Attributes are fixed at creation.)
func (n *Namespace) SetNamespaceFlags(name string, valueMask, values base.CARD32) (base.CARD32, error) {
	reply, err := n.conn.SendAndWait(&SetNamespaceFlagsRequest{
		MajorOpcode: n.MajorOpcode(),
		ValueMask:   valueMask,
		Values:      values,
		Name:        name,
	})
	if err != nil {
		return 0, err
	}
	caps := reply.CARD32()
	return caps, reply.LastError
}
