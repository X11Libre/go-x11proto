package base

import (
	"bytes"
	"encoding/binary"
)

type WriteBuffer struct {
	Payload bytes.Buffer
	BE      bool
}

func (rw *WriteBuffer) Pad() {
	for rw.Payload.Len()%4 != 0 {
		rw.WriteCARD8(0)
	}
}

func (rw *WriteBuffer) WriteCARD8(data CARD8) {
	rw.Payload.WriteByte(byte(data))
}

func (rw *WriteBuffer) WriteCARD8s(data []CARD8) {
	for _, v := range data {
		rw.WriteCARD8(v)
	}
}

func (rw *WriteBuffer) WriteCARD16(data CARD16) {
	buf := make([]byte, 2)
	if rw.BE {
		binary.BigEndian.PutUint16(buf, uint16(data))
	} else {
		binary.LittleEndian.PutUint16(buf, uint16(data))
	}
	rw.Payload.Write(buf)
}

func (rw *WriteBuffer) WriteCARD16s(data []CARD16) {
	for _, v := range data {
		rw.WriteCARD16(v)
	}
}

func (rw *WriteBuffer) WriteINT16(data INT16) {
	rw.WriteCARD16(CARD16(data))
}

func (rw *WriteBuffer) WriteCARD32(data CARD32) {
	buf := make([]byte, 4)
	if rw.BE {
		binary.BigEndian.PutUint32(buf, uint32(data))
	} else {
		binary.LittleEndian.PutUint32(buf, uint32(data))
	}
	rw.Payload.Write(buf)
}

func (rw *WriteBuffer) WriteCARD32s(data []CARD32) {
	for _, v := range data {
		rw.WriteCARD32(v)
	}
}

func (rw *WriteBuffer) WriteATOM(data ATOM) {
	rw.WriteCARD32(CARD32(data))
}

func (rw *WriteBuffer) WriteXID(data XID) {
	rw.WriteCARD32(CARD32(data))
}

func (rw *WriteBuffer) WriteDRAWABLE(data DRAWABLE) {
	rw.WriteCARD32(CARD32(data))
}

func (rw *WriteBuffer) WriteWINDOW(data WINDOW) {
	rw.WriteCARD32(CARD32(data))
}

func (rw *WriteBuffer) WriteGC(data GC) {
	rw.WriteCARD32(CARD32(data))
}

func (rw *WriteBuffer) WriteVISUAL(data VISUAL) {
	rw.WriteCARD32(CARD32(data))
}

func (rw *WriteBuffer) WriteBool(b bool) {
	if b {
		rw.WriteCARD8(1)
	} else {
		rw.WriteCARD8(0)
	}
}

func (rw *WriteBuffer) WriteBytes(b []byte) {
	rw.Payload.Write(b)
}

func (rw WriteBuffer) PayloadBytes() []byte {
	return rw.Payload.Bytes()
}

func MakeWriteBuffer(BE bool) WriteBuffer {
	return WriteBuffer{BE: BE}
}
