package base

import (
	"bytes"
	"encoding/binary"
)

type RequestWriter struct {
	WriteBuffer
	opcode CARD8
	param0 CARD8
}

func (rw *RequestWriter) SetOpcode(opcode CARD8) {
	rw.opcode = opcode
}

func (rw RequestWriter) GetOpcode() CARD8 {
	return rw.opcode
}

func (rw *RequestWriter) SetParam0(v CARD8) {
	rw.param0 = v
}

func (rw *RequestWriter) SetParam0bool(b bool) {
	if b {
		rw.param0 = 1
	} else {
		rw.param0 = 0
	}
}

// SetMinorOpcode sets the data byte (param0), which for extension requests is
// the minor opcode. Alias of SetParam0 that documents extension intent.
func (rw *RequestWriter) SetMinorOpcode(minor CARD8) {
	rw.param0 = minor
}

// SetExtOpcode sets the major opcode (the extension's assigned request opcode,
// from QueryExtension) and the minor opcode (the data byte) of an extension
// request in one call.
func (rw *RequestWriter) SetExtOpcode(major, minor CARD8) {
	rw.opcode = major
	rw.param0 = minor
}

func (rw RequestWriter) ToBytes() []byte {
	rounded := RoundFullUnits(uint(rw.Payload.Len()))
	missing := rounded - uint(rw.Payload.Len())
	for i := uint(0); i < missing; i++ {
		rw.WriteCARD8(0)
	}
	units := rw.Payload.Len() / 4

	packet := bytes.Buffer{}
	packet.WriteByte(byte(rw.opcode))
	packet.WriteByte(byte(rw.param0))
	lenbuf := make([]byte, 2)
	if rw.BE {
		binary.BigEndian.PutUint16(lenbuf, uint16(units+1))
	} else {
		binary.LittleEndian.PutUint16(lenbuf, uint16(units+1))
	}
	packet.Write(lenbuf)
	packet.Write(rw.Payload.Bytes())

	return packet.Bytes()
}

func MakeRequestWriter(BE bool) RequestWriter {
	return RequestWriter{WriteBuffer: MakeWriteBuffer(BE)}
}
