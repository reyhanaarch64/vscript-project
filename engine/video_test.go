package engine

import (
	"image"
	"image/color"
	"testing"
)

func TestBuiltinTextUsesRequestedHeight(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 320, 160))
	if err := drawText(img, "REYHAN", 160, 80, 28, color.RGBA{255, 255, 255, 255}); err != nil {
		t.Fatal(err)
	}
	minX, minY, maxX, maxY := 320, 160, -1, -1
	for y := 0; y < 160; y++ {
		for x := 0; x < 320; x++ {
			i := img.Pix[y*img.Stride+x*4:]
			if i[3] != 0 && (i[0] != 0 || i[1] != 0 || i[2] != 0) {
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if maxX < minX || maxY < minY {
		t.Fatal("text built-in tidak menghasilkan pixel")
	}
	if maxY-minY+1 > 34 {
		t.Fatalf("tinggi glyph terlalu besar: %d", maxY-minY+1)
	}
}

func TestRoundedRectCorners(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 40, 40))
	c := color.RGBA{255, 0, 0, 255}
	if err := drawRect(img, 5, 5, 20, 20, c, 8); err != nil {
		t.Fatal(err)
	}
	if img.RGBAAt(5, 5).A != 0 {
		t.Fatal("sudut rounded rect seharusnya kosong")
	}
	if img.RGBAAt(15, 5).A == 0 {
		t.Fatal("bagian atas rounded rect hilang")
	}
	if img.RGBAAt(24, 15).A == 0 {
		t.Fatal("bagian kanan rounded rect hilang")
	}
}
