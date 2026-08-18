package lithcode

import "strings"

// Common cuttings lithology codes used on the wellsite.
var table = map[string]string{
	"SS":  "sandstone",
	"SH":  "shale",
	"LS":  "limestone",
	"DOL": "dolomite",
	"ANH": "anhydrite",
	"SLT": "siltstone",
	"CLY": "claystone",
	"CGT": "conglomerate",
	"CHT": "chert",
	"COA": "coal",
	"MAR": "marl",
	"HAL": "halite",
	"GYP": "gypsum",
	"VOL": "volcanic",
	"IGN": "igneous",
	"MET": "metamorphic",
	"SST": "siltstone",
	"MDT": "mudstone",
	"CAL": "calcite",
	"QTZ": "quartz",
}

func Known(code string) bool {
	_, ok := table[strings.ToUpper(strings.TrimSpace(code))]
	return ok
}

func Name(code string) string {
	return table[strings.ToUpper(strings.TrimSpace(code))]
}

func Codes() []string {
	out := make([]string, 0, len(table))
	for k := range table {
		out = append(out, k)
	}
	return out
}

// MixOK reports whether two cuttings percentages can sit in one description
// row (they must sum to 100 ± 0.5 when both are set).
func MixOK(a, b float64) bool {
	if a == 0 && b == 0 {
		return true
	}
	sum := a + b
	if sum < 99.5 || sum > 100.5 {
		return false
	}
	return true
}
