package gas

import "math"

// Chromatograph totals used when a cuttings frame carries gas shows.

type Show struct {
	C1  float64
	C2  float64
	C3  float64
	IC4 float64
	NC4 float64
	IC5 float64
	NC5 float64
}

func (s Show) Total() float64 {
	return s.C1 + s.C2 + s.C3 + s.IC4 + s.NC4 + s.IC5 + s.NC5
}

func (s Show) Wetness() float64 {
	t := s.Total()
	if t <= 0 {
		return 0
	}
	wet := s.C2 + s.C3 + s.IC4 + s.NC4 + s.IC5 + s.NC5
	return wet / t
}

func (s Show) Balance() float64 {
	den := s.C3 + s.IC4 + s.NC4 + s.IC5 + s.NC5
	if den <= 0 {
		return 0
	}
	return (s.C1 + s.C2) / den
}

func (s Show) Character() string {
	w := s.Wetness()
	b := s.Balance()
	switch {
	case w < 0.05:
		return "dry"
	case w < 0.17 && b > 100:
		return "oil_possible"
	case w >= 0.17 && b < 17:
		return "oil"
	case w >= 0.17 && b >= 17 && b <= 100:
		return "gas_condensate"
	default:
		return "gas"
	}
}

func Normalized(unitsUnits float64, calFactor float64) float64 {
	if calFactor <= 0 {
		calFactor = 1
	}
	v := unitsUnits * calFactor
	if v < 0 {
		return 0
	}
	return math.Round(v*10) / 10
}
