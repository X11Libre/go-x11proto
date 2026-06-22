package xpm

import (
	"image"
	"testing"
)

func TestFromImageNRGBAFastPath(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	copy(src.Pix, []byte{1, 2, 3, 4, 5, 6, 7, 8})

	img := FromImage(src)
	if img.Width != 2 || img.Height != 1 {
		t.Fatalf("size = %dx%d, want 2x1", img.Width, img.Height)
	}
	// tightly-packed origin NRGBA is adopted directly (same backing array)
	if &img.Data[0] != &src.Pix[0] {
		t.Errorf("expected Pix to be reused without copying")
	}
}

func TestFromImageSubImageNormalised(t *testing.T) {
	// a sub-image has a non-zero origin and a stride wider than its width, so it
	// must be normalised rather than adopted.
	full := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	for i := range full.Pix {
		full.Pix[i] = byte(i)
	}
	sub := full.SubImage(image.Rect(1, 0, 3, 1)).(*image.NRGBA) // 2x1 window

	img := FromImage(sub)
	if img.Width != 2 || img.Height != 1 {
		t.Fatalf("size = %dx%d, want 2x1", img.Width, img.Height)
	}
	if len(img.Data) != 2*1*4 {
		t.Fatalf("len(Data) = %d, want 8 (tightly packed)", len(img.Data))
	}
	// the window starts at pixel (1,0): byte offset 1*4 = 4 in the full buffer
	want := full.Pix[4:12]
	for i, b := range want {
		if img.Data[i] != b {
			t.Errorf("Data[%d] = %d, want %d", i, img.Data[i], b)
		}
	}
}

func TestFromImageRGBAConverted(t *testing.T) {
	// a non-NRGBA type must be converted, not adopted.
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.Pix = []byte{10, 20, 30, 255}
	img := FromImage(src)
	if img.Width != 1 || img.Height != 1 || len(img.Data) != 4 {
		t.Fatalf("unexpected result %dx%d len=%d", img.Width, img.Height, len(img.Data))
	}
	if img.Data[0] != 10 || img.Data[1] != 20 || img.Data[2] != 30 || img.Data[3] != 255 {
		t.Errorf("Data = %v, want [10 20 30 255]", img.Data[:4])
	}
}
