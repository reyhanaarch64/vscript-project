package engine

import "testing"

func TestTimelineInterpolation(t *testing.T) {
	tl := NewTimeline(10)
	tl.Add("x", 0, 0.0)
	tl.Add("x", 2, 100.0)
	if got := number(tl.Value("x", 1)); got != 50 {
		t.Fatalf("got %v, want 50", got)
	}
}

func TestTimelineObjectInterpolation(t *testing.T) {
	tl := NewTimeline(4)
	tl.Add("box", 0, map[string]interface{}{"x": 0.0, "y": 10.0, "label": "a"})
	tl.Add("box", 2, map[string]interface{}{"x": 100.0, "y": 30.0, "label": "b"})
	got, ok := tl.Value("box", 1).(map[string]interface{})
	if !ok {
		t.Fatal("hasil bukan object")
	}
	if number(got["x"]) != 50 || number(got["y"]) != 20 {
		t.Fatalf("hasil object salah: %#v", got)
	}
	if formatValue(got["label"]) != "b" && formatValue(got["label"]) != "a" {
		t.Fatal("label hilang")
	}
}
