package base

import (
	"encoding/binary"
	"fmt"
)

type ReadBuffer struct {
	BE        bool
	Binary    []byte
	Original  []byte
	LastError error
}

func (r *ReadBuffer) convCARD16(data []byte) CARD16 {
	if r.BE {
		return CARD16(binary.BigEndian.Uint16(data))
	} else {
		return CARD16(binary.LittleEndian.Uint16(data))
	}
}

func (r *ReadBuffer) convCARD32(data []byte) CARD32 {
	if r.BE {
		return CARD32(binary.BigEndian.Uint32(data))
	} else {
		return CARD32(binary.LittleEndian.Uint32(data))
	}
}

func (r *ReadBuffer) ReadCARD8() (CARD8, error) {
	if len(r.Binary) < 1 {
		r.LastError = fmt.Errorf("ReadCARD8() not enough bytes in buffer")
		return 0, r.LastError
	}

	val := CARD8(r.Binary[0])
	r.Binary = r.Binary[1:]

	return val, nil
}

func (r *ReadBuffer) ReadCARD16() (CARD16, error) {
	if len(r.Binary) < 2 {
		r.LastError = fmt.Errorf("ReadCARD16() not enough bytes in buffer")
		return 0, r.LastError
	}

	val := r.convCARD16(r.Binary)
	r.Binary = r.Binary[2:]

	return val, nil
}

func (r *ReadBuffer) ReadCARD32() (CARD32, error) {
	if len(r.Binary) < 4 {
		r.LastError = fmt.Errorf("ReadCARD32() not enough bytes in buffer")
		return 0, r.LastError
	}

	val := r.convCARD32(r.Binary)
	r.Binary = r.Binary[4:]

	return val, nil
}

func (r *ReadBuffer) CARD32() CARD32 {
	v, _ := r.ReadCARD32()
	return v
}

func (r *ReadBuffer) CARD16() CARD16 {
	v, _ := r.ReadCARD16()
	return v
}

func (r *ReadBuffer) INT16() INT16 {
	return INT16(r.CARD16())
}

func (r *ReadBuffer) CARD8() CARD8 {
	v, _ := r.ReadCARD8()
	return v
}

func (r *ReadBuffer) XID() XID {
	return XID(r.CARD32())
}

func (r *ReadBuffer) Bool() bool {
	v := r.CARD8()
	return v == 1
}

func (r *ReadBuffer) ReadBytes(sz uint) []byte {
	v := r.Binary[:sz]
	r.Binary = r.Binary[sz:]
	return v
}

func (r *ReadBuffer) Reset() {
	r.Binary = r.Original
}

func (r *ReadBuffer) ReadString() string {
	buf := r.ReadBytes(uint(r.CARD8()))
	return string(buf)
}

func MakeReadBuffer(data []byte, be bool) ReadBuffer {
	return ReadBuffer{Original: data, Binary: data, BE: be}
}
