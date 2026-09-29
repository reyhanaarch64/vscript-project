package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type Keyframe struct {
	Time  float64
	Value interface{}
}

type TimelineValue struct {
	Duration float64
	Tracks   map[string][]Keyframe
}

func NewTimeline(duration float64) *TimelineValue {
	return &TimelineValue{Duration: duration, Tracks: map[string][]Keyframe{}}
}

func (t *TimelineValue) Add(track string, at float64, value interface{}) error {
	if t == nil {
		return fmt.Errorf("timeline nil")
	}
	track = strings.TrimSpace(track)
	if track == "" {
		return fmt.Errorf("nama track kosong")
	}
	if math.IsNaN(at) || math.IsInf(at, 0) || at < 0 {
		return fmt.Errorf("waktu keyframe tidak valid: %v", at)
	}
	frames := t.Tracks[track]
	for i := range frames {
		if frames[i].Time == at {
			frames[i].Value = value
			t.Tracks[track] = frames
			return nil
		}
	}
	frames = append(frames, Keyframe{Time: at, Value: value})
	sort.Slice(frames, func(i, j int) bool { return frames[i].Time < frames[j].Time })
	t.Tracks[track] = frames
	return nil
}

func (t *TimelineValue) Value(track string, at float64) interface{} {
	if t == nil {
		return nil
	}
	frames := t.Tracks[track]
	if len(frames) == 0 {
		return nil
	}
	if at <= frames[0].Time {
		return frames[0].Value
	}
	if at >= frames[len(frames)-1].Time {
		return frames[len(frames)-1].Value
	}
	for i := 0; i < len(frames)-1; i++ {
		a := frames[i]
		b := frames[i+1]
		if at >= a.Time && at <= b.Time {
			span := b.Time - a.Time
			if span <= 0 {
				return b.Value
			}
			u := (at - a.Time) / span
			return interpolate(a.Value, b.Value, u)
		}
	}
	return frames[len(frames)-1].Value
}

func interpolate(a, b interface{}, u float64) interface{} {
	if u < 0 {
		u = 0
	}
	if u > 1 {
		u = 1
	}
	switch av := a.(type) {
	case float64:
		if !isNumeric(b) {
			return pickDiscrete(a, b, u)
		}
		return av + (number(b)-av)*u
	case int:
		if !isNumeric(b) {
			return pickDiscrete(a, b, u)
		}
		return float64(av) + (number(b)-float64(av))*u
	case int64:
		if !isNumeric(b) {
			return pickDiscrete(a, b, u)
		}
		return float64(av) + (number(b)-float64(av))*u
	case []interface{}:
		bv, ok := b.([]interface{})
		if !ok || len(av) != len(bv) {
			return pickDiscrete(a, b, u)
		}
		out := make([]interface{}, len(av))
		for i := range av {
			out[i] = interpolate(av[i], bv[i], u)
		}
		return out
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		if !ok {
			return pickDiscrete(a, b, u)
		}
		out := make(map[string]interface{}, len(av))
		for k, value := range av {
			if next, exists := bv[k]; exists {
				out[k] = interpolate(value, next, u)
			} else {
				out[k] = value
			}
		}
		for k, value := range bv {
			if _, exists := out[k]; !exists {
				out[k] = value
			}
		}
		return out
	default:
		return pickDiscrete(a, b, u)
	}
}

func pickDiscrete(a, b interface{}, u float64) interface{} {
	if u < 0.5 {
		return a
	}
	return b
}

func timelineMethod(t *TimelineValue, name string) interface{} {
	switch name {
	case "keyframe":
		return func(a []interface{}) (interface{}, error) {
			if len(a) < 3 {
				return nil, fmt.Errorf("timeline.keyframe(track, time, value) membutuhkan 3 argumen")
			}
			if err := t.Add(formatValue(a[0]), number(a[1]), a[2]); err != nil {
				return nil, err
			}
			return t, nil
		}
	case "value", "at":
		return func(a []interface{}) (interface{}, error) {
			if len(a) < 2 {
				return nil, fmt.Errorf("timeline.value(track, time) membutuhkan 2 argumen")
			}
			return t.Value(formatValue(a[0]), number(a[1])), nil
		}
	case "has":
		return func(a []interface{}) (interface{}, error) {
			if len(a) < 1 {
				return false, fmt.Errorf("timeline.has(track) membutuhkan 1 argumen")
			}
			_, ok := t.Tracks[formatValue(a[0])]
			return ok, nil
		}
	case "duration":
		return t.Duration
	}
	return nil
}
