package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func testFontPath() string {
	candidates := []string{
		os.Getenv("VSCRIPT_TEST_FONT"),
		"/usr/share/fonts/fonts-go/Go-Regular.ttf",
		"/usr/share/fonts/fonts-go/Go-Bold.ttf",
		filepath.Join(os.Getenv("PREFIX"), "share/fonts/Go-Regular.ttf"),
		filepath.Join(os.Getenv("PREFIX"), "share/fonts/Go-Bold.ttf"),
	}
	for _, p := range candidates {
		if p != "" {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

func TestFontRenderStable(t *testing.T) {
	path := testFontPath()
	if path == "" {
		t.Skip("font pengujian tidak ditemukan; set VSCRIPT_TEST_FONT untuk menjalankan regresi font")
	}
	f, err := NewFont(path, 48)
	if err != nil {
		t.Skipf("FreeType/font tidak tersedia: %v", err)
	}
	defer f.Close()

	const w, h = 320, 120
	makeFrame := func() []byte {
		buf := make([]byte, w*h*4)
		for i := 0; i < len(buf); i += 4 {
			buf[i+3] = 255
		}
		return buf
	}

	first := makeFrame()
	second := makeFrame()
	if err := f.DrawRGBA(first, w, h, w*4, w/2, h/2, 48, [4]uint8{255, 255, 255, 255}, "VSCRIPT FONT 0123"); err != nil {
		t.Fatal(err)
	}
	if err := f.DrawRGBA(second, w, h, w*4, w/2, h/2, 48, [4]uint8{255, 255, 255, 255}, "VSCRIPT FONT 0123"); err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("hasil glyph berubah antara render pertama dan kedua")
	}

	covered := 0
	for i := 0; i < len(first); i += 4 {
		if first[i] != 0 || first[i+1] != 0 || first[i+2] != 0 {
			covered++
		}
	}
	if covered < 20 {
		t.Fatalf("bitmap text terlalu sedikit: %d pixel", covered)
	}
}

func TestFontTextSizeOverride(t *testing.T) {
	path := testFontPath()
	if path == "" {
		t.Skip("font pengujian tidak ditemukan; set VSCRIPT_TEST_FONT untuk menjalankan regresi font")
	}
	f, err := NewFont(path, 24)
	if err != nil {
		t.Skipf("FreeType/font tidak tersedia: %v", err)
	}
	defer f.Close()

	const w, h = 240, 120
	makeFrame := func() []byte {
		buf := make([]byte, w*h*4)
		for i := 0; i < len(buf); i += 4 {
			buf[i+3] = 255
		}
		return buf
	}

	small := makeFrame()
	large := makeFrame()
	if err := f.DrawRGBA(small, w, h, w*4, w/2, h/2, 20, [4]uint8{255, 255, 255, 255}, "size"); err != nil {
		t.Fatal(err)
	}
	if err := f.DrawRGBA(large, w, h, w*4, w/2, h/2, 48, [4]uint8{255, 255, 255, 255}, "size"); err != nil {
		t.Fatal(err)
	}
	if string(small) == string(large) {
		t.Fatal("ukuran text() tidak memengaruhi font renderer")
	}
}
