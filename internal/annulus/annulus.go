package annulus

import (
	"math"

	"github.com/lacsar712/mudlog/internal/pump"
)

const (
	MaxLagMinutes   = 240.0
	DefaultIDInch   = 8.5
	DefaultPipeInch = 5.0
)

// VolumeBBL returns annular volume in barrels for a measured interval.
func VolumeBBL(idInch, pipeODInch, lengthM float64) float64 {
	if idInch <= 0 || lengthM < 0 {
		return 0
	}
	if pipeODInch < 0 {
		pipeODInch = 0
	}
	if pipeODInch >= idInch {
		return 0
	}
	// capacity factor: (ID^2 - OD^2) / 1029.4 bbl per foot, length in metres.
	feet := lengthM * 3.28084
	cap := (idInch*idInch - pipeODInch*pipeODInch) / 1029.4
	return cap * feet
}

// LagMinutes estimates sample lag from pump output (bbl/min) and annulus volume.
func LagMinutes(annulusBBL, pumpBBLMin float64) float64 {
	if pumpBBLMin <= 0 || annulusBBL <= 0 {
		return 0
	}
	m := annulusBBL / pumpBBLMin
	if m > MaxLagMinutes {
		return MaxLagMinutes
	}
	return m
}

// StrokesToBBL converts mud-pump strokes to barrels.
func StrokesToBBL(strokes int, displacementBBL float64, efficiency float64) float64 {
	if strokes < 0 || displacementBBL <= 0 {
		return 0
	}
	if efficiency <= 0 {
		efficiency = 0.95
	}
	if efficiency > 1 {
		efficiency = 1
	}
	return float64(strokes) * displacementBBL * efficiency
}

// BitLagDepth estimates the cuttings depth given bit MD and lag metres.
func BitLagDepth(bitMD, lagMetres float64) float64 {
	d := bitMD - lagMetres
	if d < 0 {
		return 0
	}
	return math.Round(d*100) / 100
}

// FromPump estimates lag minutes from a mud-pump spec and hole geometry.
func FromPump(spec pump.Spec, idInch, pipeODInch, lengthM float64) float64 {
	vol := VolumeBBL(idInch, pipeODInch, lengthM)
	return LagMinutes(vol, spec.OutputBBLMin())
}
