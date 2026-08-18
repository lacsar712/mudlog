package channel

import "strings"

// Standard WITS mud-logging channel identifiers used when packing outbound
// frames for a geology store.
const (
	ROP     = "0108" // rate of penetration
	WOB     = "0110" // weight on bit
	RPM     = "0112"
	SPP     = "0114" // standpipe pressure
	FlowIn  = "0116"
	FlowOut = "0118"
	Gas     = "0120"
	Gamma   = "0122"
	Hook    = "0124"
	Torque  = "0126"
	LagMD   = "0201"
	LagTVD  = "0202"
	Lith1   = "0301"
)

var names = map[string]string{
	ROP:     "rop_m_hr",
	WOB:     "wob_klbf",
	RPM:     "rpm",
	SPP:     "spp_psi",
	FlowIn:  "flow_in_gpm",
	FlowOut: "flow_out_gpm",
	Gas:     "gas_pct",
	Gamma:   "gamma_api",
	Hook:    "hookload_klbf",
	Torque:  "torque_kftlb",
	LagMD:   "lag_md_m",
	LagTVD:  "lag_tvd_m",
	Lith1:   "lithology_code",
}

func Name(code string) string {
	return names[strings.TrimSpace(code)]
}

func Known(code string) bool {
	_, ok := names[strings.TrimSpace(code)]
	return ok
}

func All() []string {
	out := make([]string, 0, len(names))
	for k := range names {
		out = append(out, k)
	}
	return out
}

// PickNumeric returns the first known numeric channel present in payload.
func PickNumeric(payload map[string]any) (code string, value float64, ok bool) {
	order := []string{ROP, WOB, RPM, SPP, FlowIn, FlowOut, Gas, Gamma, Hook, Torque, LagMD, LagTVD}
	for _, c := range order {
		key := names[c]
		raw, exists := payload[key]
		if !exists {
			raw, exists = payload[c]
		}
		if !exists {
			continue
		}
		switch n := raw.(type) {
		case float64:
			return c, n, true
		case int:
			return c, float64(n), true
		}
	}
	return "", 0, false
}
