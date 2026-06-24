package namespace

import "github.com/X11Libre/go-x11proto/proto/base"

// ---- AddAuthToken ----

type AddAuthTokenRequest struct {
	MajorOpcode base.CARD8
	Name        string // target namespace
	Proto       string // auth protocol name, e.g. "MIT-MAGIC-COOKIE-1"
	Data        []byte // auth data (key material)
}

func (q *AddAuthTokenRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorAddAuthToken)
	w.WriteCARD16(base.CARD16(len(q.Name)))
	w.WriteCARD16(base.CARD16(len(q.Proto)))
	w.WriteCARD16(base.CARD16(len(q.Data)))
	w.WriteCARD16(0) // pad0
	putPadded(w, []byte(q.Name))
	putPadded(w, []byte(q.Proto))
	putPadded(w, q.Data)
	return nil
}

// AddAuthToken registers an auth token for a namespace and returns its handle.
func (n *Namespace) AddAuthToken(name, proto string, data []byte) (base.CARD32, error) {
	reply, err := n.conn.SendAndWait(&AddAuthTokenRequest{
		MajorOpcode: n.MajorOpcode(),
		Name:        name,
		Proto:       proto,
		Data:        data,
	})
	if err != nil {
		return 0, err
	}
	handle := reply.CARD32()
	return handle, reply.LastError
}

// ---- RemoveAuthToken ----

type RemoveAuthTokenRequest struct {
	MajorOpcode base.CARD8
	TokenHandle base.CARD32
	Name        string
}

func (q *RemoveAuthTokenRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorRemoveAuthToken)
	w.WriteCARD32(q.TokenHandle)
	w.WriteCARD16(base.CARD16(len(q.Name)))
	w.WriteCARD16(0) // pad0
	putPadded(w, []byte(q.Name))
	return nil
}

// RemoveAuthToken removes the auth token with the given handle from a namespace.
func (n *Namespace) RemoveAuthToken(name string, tokenHandle base.CARD32) error {
	_, err := n.conn.SendAndWait(&RemoveAuthTokenRequest{
		MajorOpcode: n.MajorOpcode(),
		TokenHandle: tokenHandle,
		Name:        name,
	})
	return err
}

// ---- ListAuthTokens ----

type ListAuthTokensRequest struct {
	MajorOpcode base.CARD8
	Name        string
}

func (q *ListAuthTokensRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(q.MajorOpcode, MinorListAuthTokens)
	w.WriteCARD16(base.CARD16(len(q.Name)))
	w.WriteCARD16(0) // pad0
	putPadded(w, []byte(q.Name))
	return nil
}

// ListAuthTokens lists a namespace's auth tokens (handle + protocol name; never
// the key material).
func (n *Namespace) ListAuthTokens(name string) ([]AuthToken, error) {
	reply, err := n.conn.SendAndWait(&ListAuthTokensRequest{
		MajorOpcode: n.MajorOpcode(),
		Name:        name,
	})
	if err != nil {
		return nil, err
	}
	return parseTokenList(reply), reply.LastError
}

// parseTokenList reads a ListAuthTokens reply: count then count token records
// (reader positioned just after the 8-byte header).
func parseTokenList(rr *base.ReplyReader) []AuthToken {
	count := int(rr.CARD32())
	rr.ReadBytes(20) // pad1..pad5
	out := make([]AuthToken, 0, count)
	for i := 0; i < count; i++ {
		var tok AuthToken
		tok.Handle = rr.CARD32()
		protoLen := int(rr.CARD16())
		rr.CARD16() // pad
		tok.Proto = string(getPadded(rr, protoLen))
		out = append(out, tok)
	}
	return out
}
