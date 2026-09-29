package engine

import "testing"

func TestTimelineInterpolation(t *testing.T) {
	tl := NewTimeline(10)
	if err := tl.Add("x", 0, 0.0); err != nil {
		t.Fatal(err)
	}
	if err := tl.Add("x", 2, 100.0); err != nil {
		t.Fatal(err)
	}
	if got := number(tl.Value("x", 1)); got != 50 {
		t.Fatalf("got %v, want 50", got)
	}
}

func TestTimelineObjectInterpolation(t *testing.T) {
	tl := NewTimeline(4)
	if err := tl.Add("box", 0, map[string]interface{}{"x": 0.0, "y": 10.0, "label": "a"}); err != nil {
		t.Fatal(err)
	}
	if err := tl.Add("box", 2, map[string]interface{}{"x": 100.0, "y": 30.0, "label": "b"}); err != nil {
		t.Fatal(err)
	}
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

func TestTimelineDuplicateReplaces(t *testing.T) {
	tl := NewTimeline(5)
	if err := tl.Add("x", 1, 10.0); err != nil {
		t.Fatal(err)
	}
	if err := tl.Add("x", 1, 20.0); err != nil {
		t.Fatal(err)
	}
	if got := number(tl.Value("x", 1)); got != 20 {
		t.Fatalf("got %v, want 20", got)
	}
	if len(tl.Tracks["x"]) != 1 {
		t.Fatalf("duplicate keyframe tidak diganti: %#v", tl.Tracks["x"])
	}
}

func TestTimelineTypeMismatchIsDiscrete(t *testing.T) {
	tl := NewTimeline(2)
	if err := tl.Add("x", 0, 10.0); err != nil {
		t.Fatal(err)
	}
	if err := tl.Add("x", 2, "end"); err != nil {
		t.Fatal(err)
	}
	if got := tl.Value("x", 0.5); got != 10.0 {
		t.Fatalf("got %#v, want numeric first value", got)
	}
	if got := tl.Value("x", 1.5); got != "end" {
		t.Fatalf("got %#v, want string second value", got)
	}
}
