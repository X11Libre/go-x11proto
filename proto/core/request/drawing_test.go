package request

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
)

func TestCopyPlaneEncode(t *testing.T) {
	checkEncode(t, &CopyPlaneRequest{
		SrcDrawable: 0x10, DstDrawable: 0x20, Gc: 0x30,
		SrcX: 1, SrcY: 2, DstX: 3, DstY: 4, Width: 5, Height: 6, BitPlane: 1,
	}, req(63, 0, cat(u32(0x10), u32(0x20), u32(0x30), u16(1), u16(2), u16(3), u16(4), u16(5), u16(6), u32(1))))
}

func TestPolyPointEncode(t *testing.T) {
	checkEncode(t, &PolyPointRequest{CoordMode: CoordModePrevious, Drawable: 0x10, Gc: 0x20,
		Points: []base.Point{{X: 1, Y: 2}, {X: 3, Y: 4}}},
		req(64, 1, cat(u32(0x10), u32(0x20), u16(1), u16(2), u16(3), u16(4))))
}

func TestPolyLineEncode(t *testing.T) {
	checkEncode(t, &PolyLineRequest{CoordMode: CoordModePrevious, Drawable: 0x10, Gc: 0x20,
		Points: []base.Point{{X: 1, Y: 2}, {X: 3, Y: 4}}},
		req(65, 1, cat(u32(0x10), u32(0x20), u16(1), u16(2), u16(3), u16(4))))
}

func TestPolySegmentEncode(t *testing.T) {
	checkEncode(t, &PolySegmentRequest{Drawable: 0x10, Gc: 0x20,
		Segments: []base.Segment{{X1: 1, Y1: 2, X2: 3, Y2: 4}}},
		req(66, 0, cat(u32(0x10), u32(0x20), u16(1), u16(2), u16(3), u16(4))))
}

func TestPolyRectangleEncode(t *testing.T) {
	checkEncode(t, &PolyRectangleRequest{Drawable: 0x10, Gc: 0x20,
		Rectangles: []base.Rectangle{{X: 1, Y: 2, Width: 3, Height: 4}}},
		req(67, 0, cat(u32(0x10), u32(0x20), u16(1), u16(2), u16(3), u16(4))))
}

func TestPolyArcEncode(t *testing.T) {
	checkEncode(t, &PolyArcRequest{Drawable: 0x10, Gc: 0x20,
		Arcs: []base.Arc{{X: 1, Y: 2, Width: 3, Height: 4, Angle1: 5, Angle2: 6}}},
		req(68, 0, cat(u32(0x10), u32(0x20), u16(1), u16(2), u16(3), u16(4), u16(5), u16(6))))
}

func TestFillPolyEncode(t *testing.T) {
	checkEncode(t, &FillPolyRequest{Drawable: 0x10, Gc: 0x20, Shape: PolyShapeConvex, CoordMode: CoordModePrevious,
		Points: []base.Point{{X: 1, Y: 2}}},
		req(69, 0, cat(u32(0x10), u32(0x20), u8(2), u8(1), u16(0), u16(1), u16(2))))
}

func TestPolyFillArcEncode(t *testing.T) {
	checkEncode(t, &PolyFillArcRequest{Drawable: 0x10, Gc: 0x20,
		Arcs: []base.Arc{{X: 1, Y: 2, Width: 3, Height: 4, Angle1: 5, Angle2: 6}}},
		req(71, 0, cat(u32(0x10), u32(0x20), u16(1), u16(2), u16(3), u16(4), u16(5), u16(6))))
}

func TestGetImageEncode(t *testing.T) {
	checkEncode(t, &GetImageRequest{Format: ImageFormatZPixmap, Drawable: 0x10, X: 1, Y: 2, Width: 3, Height: 4, PlaneMask: 0xFF},
		req(73, 2, cat(u32(0x10), u16(1), u16(2), u16(3), u16(4), u32(0xFF))))
}

func TestGetImageReply(t *testing.T) {
	tail := cat(u32(0x20), make([]byte, 20), []byte{1, 2, 3, 4})
	r := makeReply(24, 1, tail)
	got := &GetImageReply{}
	if err := got.Parse(r); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetImageReply{Depth: 24, Visual: 0x20, Data: []byte{1, 2, 3, 4}})
}

func TestImageText8Encode(t *testing.T) {
	checkEncode(t, &ImageText8Request{Drawable: 0x10, Gc: 0x20, X: 1, Y: 2, Text: "Hi"},
		req(76, 2, cat(u32(0x10), u32(0x20), u16(1), u16(2), []byte("Hi"))))
}

func TestImageText16Encode(t *testing.T) {
	checkEncode(t, &ImageText16Request{Drawable: 0x10, Gc: 0x20, X: 1, Y: 2, Text: []base.CARD16{0x4869}},
		req(77, 1, cat(u32(0x10), u32(0x20), u16(1), u16(2), []byte{0x48, 0x69})))
}

func TestPolyText16Encode(t *testing.T) {
	checkEncode(t, &PolyText16Request{Drawable: 0x10, Gc: 0x20, X: 1, Y: 2, Text: []base.CARD16{0x4869}},
		req(75, 0, cat(u32(0x10), u32(0x20), u16(1), u16(2), u8(1), u8(0), []byte{0x48, 0x69})))
}
