package pump

import "math"

// Duplex and triplex mud-pump displacement helpers used when converting
// strokes into annular lag.

type Kind int

const (
	Triplex Kind = iota
	Duplex
)

type Spec struct {
	Kind          Kind
	LinerInch     float64
	StrokeInch    float64
	RodInch       float64
	Efficiency    float64
	StrokesPerMin float64
}

func (s Spec) DisplacementBBL() float64 {
	if s.LinerInch <= 0 || s.StrokeInch <= 0 {
		return 0
	}
	eff := s.Efficiency
	if eff <= 0 {
		eff = 0.95
	}
	if eff > 1 {
		eff = 1
	}
	// bbl per stroke ≈ (liner^2 * stroke * factor) / 1029.4
	factor := 3.0
	if s.Kind == Duplex {
		if s.RodInch < 0 {
			s.RodInch = 0
		}
		// double acting: two liner volumes minus rod displacement
		factor = 2.0
		fwd := (s.LinerInch * s.LinerInch * s.StrokeInch) / 1029.4
		rev := ((s.LinerInch*s.LinerInch - s.RodInch*s.RodInch) * s.StrokeInch) / 1029.4
		return (fwd + rev) * eff
	}
	return (s.LinerInch * s.LinerInch * s.StrokeInch * factor / 1029.4) * eff
}

func (s Spec) OutputBBLMin() float64 {
	if s.StrokesPerMin <= 0 {
		return 0
	}
	return s.DisplacementBBL() * s.StrokesPerMin
}

func StrokesForVolume(spec Spec, bbl float64) int {
	d := spec.DisplacementBBL()
	if d <= 0 || bbl <= 0 {
		return 0
	}
	return int(math.Ceil(bbl / d))
}

func SPMFromOutput(spec Spec, bblMin float64) float64 {
	d := spec.DisplacementBBL()
	if d <= 0 || bblMin <= 0 {
		return 0
	}
	return bblMin / d
}
