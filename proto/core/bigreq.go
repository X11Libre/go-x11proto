package core

import "github.com/X11Libre/go-x11proto/proto/base"

// BIG-REQUESTS lets a client send requests longer than the 65535 4-byte units a
// 16-bit length field can express. After QueryExtension, a single Enable
// request returns an enlarged maximum-request-length; oversized requests are
// then sent with the 16-bit length set to 0 followed by a 32-bit length (see
// encodeBigRequest).

const bigReqExtName = "BIG-REQUESTS"

// bigReqEnableRequest is the extension's only request (minor opcode 0).
type bigReqEnableRequest struct{ major base.CARD8 }

func (r *bigReqEnableRequest) WriteInto(w *base.RequestWriter) error {
	w.SetExtOpcode(r.major, 0)
	return nil
}

// enableBigRequests negotiates BIG-REQUESTS and, on success, raises maxReqUnits
// to the server's enlarged maximum. Best-effort: if the extension is absent or
// the exchange fails it leaves the normal (CARD16) limit in place.
func (c *X11Conn) enableBigRequests() {
	ext, err := c.QueryExtension(bigReqExtName)
	if err != nil || !ext.Present {
		return
	}
	reply, err := c.SendAndWait(&bigReqEnableRequest{major: ext.MajorOpcode})
	if err != nil {
		return
	}
	max := int(reply.CARD32()) // maximum-request-length, in 4-byte units
	if max > c.maxReqUnits {
		c.maxReqUnits = max
		c.bigReqEnabled = true
	}
}
