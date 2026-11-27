package adaptivevideo

import (
	"errors"
	"math"
	"time"
)

const (
	MaxHeight    = 16384
	MaxVariants  = 16
	SafetyFactor = 0.8
)

var (
	ErrNoQuality = errors.New("adaptive video: no variant meets the receiver's minimum height")
	ErrNoFit     = errors.New("adaptive video: no variant fits the video deadline")
)

type Variant struct {
	Height int
	Bytes  int64
}

type Selection struct {
	Variant
	ETA  time.Duration
	Late bool
}

// Select uses the already discounted rate in bytes per second.
func Select(variants []Variant, minimum int, safeRate float64, budget time.Duration) (Selection, error) {
	if safeRate <= 0 || math.IsNaN(safeRate) || math.IsInf(safeRate, 0) || minimum < 0 || minimum > MaxHeight || budget < 0 {
		return Selection{}, errors.New("adaptive video: invalid rate, budget or minimum height")
	}
	if len(variants) < 2 || len(variants) > MaxVariants {
		return Selection{}, errors.New("adaptive video: expected between two and 16 variants")
	}

	seen := make(map[int]bool, len(variants))
	var best, fastest Selection
	for _, v := range variants {
		if v.Height <= 0 || v.Height > MaxHeight || v.Bytes <= 0 || seen[v.Height] {
			return Selection{}, errors.New("adaptive video: invalid variant or duplicate height")
		}
		seen[v.Height] = true
		if v.Height < minimum {
			continue
		}
		nanos := math.Ceil(float64(v.Bytes) / safeRate * float64(time.Second))
		// Saturate only the displayed duration. Compare the unrounded range
		// against the budget and use bytes for ordering even on saturation.
		eta := time.Duration(math.MaxInt64)
		if nanos < float64(math.MaxInt64) {
			eta = time.Duration(nanos)
		}
		candidate := Selection{Variant: v, ETA: eta, Late: eta > budget || nanos >= float64(math.MaxInt64)}
		if !candidate.Late && v.Height > best.Height {
			best = candidate
		}
		if fastest.Height == 0 || v.Bytes < fastest.Bytes || (v.Bytes == fastest.Bytes && v.Height > fastest.Height) {
			fastest = candidate
		}
	}
	if best.Height != 0 {
		return best, nil
	}
	if fastest.Height == 0 {
		return Selection{}, ErrNoQuality
	}
	if minimum == 0 {
		return Selection{}, ErrNoFit
	}
	return fastest, nil
}
