package base

type ReplyReader struct {
	ReadBuffer
	BE       bool
	Type     CARD8
	Data0    CARD8
	Sequence CARD16
	Length   CARD32
}

func (r *ReplyReader) SetPayload(data []byte, be bool) {
	r.ReadBuffer = MakeReadBuffer(data, be)
}
