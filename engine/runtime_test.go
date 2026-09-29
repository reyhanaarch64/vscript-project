package engine

import (
	"image/color"
	"testing"
)

func TestParseHexColor(t *testing.T) {
	c, err := parseHexColor("#0af")
	if err != nil || c != (colorRGBA(0x00, 0xaa, 0xff, 0xff)) {
		t.Fatalf("warna #0af salah: %#v %v", c, err)
	}
	c, err = parseHexColor("#11223380")
	if err != nil || c != (colorRGBA(0x11, 0x22, 0x33, 0x80)) {
		t.Fatalf("warna alpha salah: %#v %v", c, err)
	}
	if _, err = parseHexColor("#gg0000"); err == nil {
		t.Fatal("warna invalid seharusnya menghasilkan error")
	}
}

func colorRGBA(r, g, b, a uint8) color.RGBA { return color.RGBA{r, g, b, a} }

func TestModuloByZero(t *testing.T) {
	env := NewEnv(nil)
	got, err := evalBinary(BinaryExpr{Left: LiteralExpr{Value: 10.0}, Op: TokenPercent, Right: LiteralExpr{Value: 0.0}}, env)
	if err == nil || got != nil {
		t.Fatalf("got %#v, err=%v", got, err)
	}
}

func TestParseDurationTextStrict(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"2s", 2},
		{"1500ms", 1.5},
		{"0.5", 0.5},
	}
	for _, tc := range tests {
		got, err := parseDurationText(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("duration %q: got %v, err=%v", tc.in, got, err)
		}
	}
	for _, bad := range []string{"2seconds", "1.5junk", "", "12msx"} {
		if _, err := parseDurationText(bad); err == nil {
			t.Fatalf("duration invalid %q seharusnya error", bad)
		}
	}
}
