package request

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
)

func TestCreateColormapEncode(t *testing.T) {
	checkEncode(t, &CreateColormapRequest{Alloc: ColormapAllocAll, Mid: 0x10, Window: 0x20, Visual: 0x30},
		req(78, 1, cat(u32(0x10), u32(0x20), u32(0x30))))
}

func TestFreeColormapEncode(t *testing.T) {
	checkEncode(t, &FreeColormapRequest{Colormap: 0x10}, req(79, 0, u32(0x10)))
}

func TestCopyColormapAndFreeEncode(t *testing.T) {
	checkEncode(t, &CopyColormapAndFreeRequest{Mid: 0x10, SrcMap: 0x20}, req(80, 0, cat(u32(0x10), u32(0x20))))
}

func TestInstallColormapEncode(t *testing.T) {
	checkEncode(t, &InstallColormapRequest{Colormap: 0x10}, req(81, 0, u32(0x10)))
}

func TestUninstallColormapEncode(t *testing.T) {
	checkEncode(t, &UninstallColormapRequest{Colormap: 0x10}, req(82, 0, u32(0x10)))
}

func TestListInstalledColormapsEncode(t *testing.T) {
	checkEncode(t, &ListInstalledColormapsRequest{Window: 0x10}, req(83, 0, u32(0x10)))
}

func TestListInstalledColormapsReply(t *testing.T) {
	tail := cat(u16(2), make([]byte, 22), u32(0xA), u32(0xB))
	got := &ListInstalledColormapsReply{}
	if err := got.Parse(makeReply(0, 2, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &ListInstalledColormapsReply{Colormaps: []base.COLORMAP{0xA, 0xB}})
}

func TestAllocColorEncode(t *testing.T) {
	checkEncode(t, &AllocColorRequest{Colormap: 0x10, Red: 1, Green: 2, Blue: 3},
		req(84, 0, cat(u32(0x10), u16(1), u16(2), u16(3), u16(0))))
}

func TestAllocColorReply(t *testing.T) {
	tail := cat(u16(1), u16(2), u16(3), u16(0), u32(0x12345), make([]byte, 12))
	got := &AllocColorReply{}
	if err := got.Parse(makeReply(0, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &AllocColorReply{Red: 1, Green: 2, Blue: 3, Pixel: 0x12345})
}

func TestAllocNamedColorEncode(t *testing.T) {
	checkEncode(t, &AllocNamedColorRequest{Colormap: 0x10, Name: "red"},
		req(85, 0, cat(u32(0x10), u16(3), u16(0), []byte("red"))))
}

func TestAllocNamedColorReply(t *testing.T) {
	tail := cat(u32(0x12345), u16(1), u16(2), u16(3), u16(4), u16(5), u16(6), make([]byte, 8))
	got := &AllocNamedColorReply{}
	if err := got.Parse(makeReply(0, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &AllocNamedColorReply{Pixel: 0x12345, ExactRed: 1, ExactGreen: 2, ExactBlue: 3, VisualRed: 4, VisualGreen: 5, VisualBlue: 6})
}

func TestAllocColorCellsEncode(t *testing.T) {
	checkEncode(t, &AllocColorCellsRequest{Contiguous: true, Colormap: 0x10, Colors: 2, Planes: 1},
		req(86, 1, cat(u32(0x10), u16(2), u16(1))))
}

func TestAllocColorCellsReply(t *testing.T) {
	tail := cat(u16(2), u16(1), make([]byte, 20), u32(0x100), u32(0x101), u32(0xF000))
	got := &AllocColorCellsReply{}
	if err := got.Parse(makeReply(0, 3, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &AllocColorCellsReply{Pixels: []base.CARD32{0x100, 0x101}, Masks: []base.CARD32{0xF000}})
}

func TestAllocColorPlanesEncode(t *testing.T) {
	checkEncode(t, &AllocColorPlanesRequest{Contiguous: false, Colormap: 0x10, Colors: 1, Reds: 1, Greens: 1, Blues: 1},
		req(87, 0, cat(u32(0x10), u16(1), u16(1), u16(1), u16(1))))
}

func TestAllocColorPlanesReply(t *testing.T) {
	tail := cat(u16(1), u16(0), u32(0xFF0000), u32(0x00FF00), u32(0x0000FF), make([]byte, 8), u32(0x100))
	got := &AllocColorPlanesReply{}
	if err := got.Parse(makeReply(0, 1, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &AllocColorPlanesReply{RedMask: 0xFF0000, GreenMask: 0x00FF00, BlueMask: 0x0000FF, Pixels: []base.CARD32{0x100}})
}

func TestFreeColorsEncode(t *testing.T) {
	checkEncode(t, &FreeColorsRequest{Colormap: 0x10, PlaneMask: 0xFF, Pixels: []base.CARD32{0x100, 0x101}},
		req(88, 0, cat(u32(0x10), u32(0xFF), u32(0x100), u32(0x101))))
}

func TestStoreColorsEncode(t *testing.T) {
	checkEncode(t, &StoreColorsRequest{Colormap: 0x10, Items: []ColorItem{{Pixel: 0x100, Red: 1, Green: 2, Blue: 3, Flags: 7}}},
		req(89, 0, cat(u32(0x10), u32(0x100), u16(1), u16(2), u16(3), u8(7), u8(0))))
}

func TestStoreNamedColorEncode(t *testing.T) {
	checkEncode(t, &StoreNamedColorRequest{Flags: 7, Colormap: 0x10, Pixel: 0x100, Name: "red"},
		req(90, 7, cat(u32(0x10), u32(0x100), u16(3), u16(0), []byte("red"))))
}

func TestQueryColorsEncode(t *testing.T) {
	checkEncode(t, &QueryColorsRequest{Colormap: 0x10, Pixels: []base.CARD32{0x100}},
		req(91, 0, cat(u32(0x10), u32(0x100))))
}

func TestQueryColorsReply(t *testing.T) {
	tail := cat(u16(1), make([]byte, 22), u16(10), u16(20), u16(30), u16(0))
	got := &QueryColorsReply{}
	if err := got.Parse(makeReply(0, 2, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &QueryColorsReply{Colors: []Rgb{{Red: 10, Green: 20, Blue: 30}}})
}

func TestLookupColorEncode(t *testing.T) {
	checkEncode(t, &LookupColorRequest{Colormap: 0x10, Name: "red"},
		req(92, 0, cat(u32(0x10), u16(3), u16(0), []byte("red"))))
}

func TestLookupColorReply(t *testing.T) {
	tail := cat(u16(1), u16(2), u16(3), u16(4), u16(5), u16(6), make([]byte, 12))
	got := &LookupColorReply{}
	if err := got.Parse(makeReply(0, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &LookupColorReply{ExactRed: 1, ExactGreen: 2, ExactBlue: 3, VisualRed: 4, VisualGreen: 5, VisualBlue: 6})
}
