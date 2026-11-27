package adaptivevideo

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestSelect(t *testing.T) {
	standard := []Variant{{1080, 75_000_000}, {360, 12_000_000}, {144, 4_000_000}}
	nonMonotonic := []Variant{{144, 17_000_000}, {1080, 15_000_000}, {360, 12_000_000}}
	for _, tc := range []struct {
		name      string
		variants  []Variant
		minimum   int
		rate      float64
		seconds   int
		height    int
		eta       time.Duration
		late      bool
		wantError error
	}{
		{"fast", standard, 0, 2_500_000, 60, 1080, 30 * time.Second, false, nil},
		{"medium", standard, 0, 312_500, 60, 360, 38400 * time.Millisecond, false, nil},
		{"slow", standard, 0, 100_000, 60, 144, 40 * time.Second, false, nil},
		{"exact, unordered", nonMonotonic, 0, 250_000, 60, 1080, 60 * time.Second, false, nil},
		{"lower height can be larger", nonMonotonic, 0, 225_000, 60, 360, 53333333334, false, nil},
		{"late fastest at minimum", []Variant{{144, 2_000_000}, {1080, 6_000_000}, {360, 8_000_000}}, 360, 100_000, 10, 1080, time.Minute, true, nil},
		{"late tie prefers quality", []Variant{{360, 6_000_000}, {1080, 6_000_000}}, 360, 100_000, 10, 1080, time.Minute, true, nil},
		{"cannot meet quality", standard, 2160, 100_000, 60, 0, 0, false, ErrNoQuality},
		{"cannot meet deadline", standard, 0, 50_000, 60, 0, 0, false, ErrNoFit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Select(tc.variants, tc.minimum, tc.rate, time.Duration(tc.seconds)*time.Second)
			if !errors.Is(err, tc.wantError) || got.Height != tc.height || got.ETA != tc.eta || got.Late != tc.late {
				t.Fatalf("got %+v, %v; want height=%d eta=%v late=%v error=%v", got, err, tc.height, tc.eta, tc.late, tc.wantError)
			}
		})
	}
}

func TestSelectInvalid(t *testing.T) {
	variants := []Variant{{360, 1000}, {144, 500}}
	for _, rate := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := Select(variants, 0, rate, time.Second); err == nil {
			t.Errorf("accepted rate %g", rate)
		}
	}
	if _, err := Select(variants, 0, 1000, -time.Second); err == nil {
		t.Error("accepted a negative budget")
	}
	if _, err := Select([]Variant{{360, 1000}, {360, 500}}, 0, 1000, time.Second); err == nil {
		t.Error("accepted a duplicate height")
	}
}
