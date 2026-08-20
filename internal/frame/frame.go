package frame

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/lacsar712/mudlog/internal/annulus"
	"github.com/lacsar712/mudlog/internal/lithcode"
	"github.com/lacsar712/mudlog/internal/wellbore"
)

const MaxBody = 256 * 1024

// Lithology is a cuttings description attached to a mud-log frame.
type Lithology struct {
	Code        string  `json:"code"`
	Description string  `json:"description,omitempty"`
	Percent     float64 `json:"percent"`
}

// LagDepth is the sample lag relative to bit depth.
type LagDepth struct {
	MeasuredM  float64 `json:"measured_m"`
	TrueVertM  float64 `json:"true_vert_m,omitempty"`
	LagMinutes float64 `json:"lag_minutes,omitempty"`
}

// Frame is the signed WITS-like document a rig posts.
type Frame struct {
	Type      string          `json:"type"`
	Wellbore  string          `json:"wellbore"`
	Lithology Lithology       `json:"lithology"`
	LagDepth  LagDepth        `json:"lag_depth"`
	Payload   json.RawMessage `json:"payload"`
}

func Parse(body []byte) (Frame, error) {
	if len(body) == 0 {
		return Frame{}, fmt.Errorf("empty body")
	}
	if len(body) > MaxBody {
		return Frame{}, fmt.Errorf("body %d exceeds %d bytes", len(body), MaxBody)
	}
	var fr Frame
	if err := json.Unmarshal(body, &fr); err != nil {
		return Frame{}, fmt.Errorf("json: %w", err)
	}
	if err := ValidateType(fr.Type); err != nil {
		return Frame{}, err
	}
	if err := wellbore.Validate(fr.Wellbore); err != nil {
		return Frame{}, err
	}
	if err := validateLithology(fr.Lithology); err != nil {
		return Frame{}, err
	}
	if err := validateLag(fr.LagDepth); err != nil {
		return Frame{}, err
	}
	if len(fr.Payload) == 0 || string(fr.Payload) == "null" {
		return Frame{}, fmt.Errorf("payload is required")
	}
	if !json.Valid(fr.Payload) {
		return Frame{}, fmt.Errorf("payload is not valid json")
	}
	return fr, nil
}

func validateLithology(lith Lithology) error {
	code := strings.TrimSpace(lith.Code)
	if code == "" {
		return nil
	}
	if !lithcode.Known(code) {
		return fmt.Errorf("unknown lithology code %q", code)
	}
	if lith.Percent < 0 || lith.Percent > 100 {
		return fmt.Errorf("lithology percent %v not in [0,100]", lith.Percent)
	}
	return nil
}

func validateLag(lag LagDepth) error {
	if lag.MeasuredM < 0 {
		return fmt.Errorf("measured depth must be >= 0")
	}
	if lag.TrueVertM < 0 {
		return fmt.Errorf("tvd must be >= 0")
	}
	if lag.TrueVertM > 0 && lag.MeasuredM > 0 && lag.TrueVertM > lag.MeasuredM+1e-6 {
		return fmt.Errorf("tvd cannot exceed measured depth")
	}
	if lag.LagMinutes < 0 {
		return fmt.Errorf("lag minutes must be >= 0")
	}
	if lag.LagMinutes > annulus.MaxLagMinutes {
		return fmt.Errorf("lag minutes exceed %v", annulus.MaxLagMinutes)
	}
	return nil
}

func ValidateType(t string) error {
	t = strings.TrimSpace(t)
	if t == "" {
		return fmt.Errorf("frame type is required")
	}
	if len(t) > 128 {
		return fmt.Errorf("frame type too long")
	}
	parts := strings.Split(t, ".")
	if len(parts) < 1 {
		return fmt.Errorf("frame type is empty")
	}
	for _, p := range parts {
		if p == "" {
			return fmt.Errorf("frame type has empty segment")
		}
		for _, r := range p {
			if unicode.IsUpper(r) {
				return fmt.Errorf("frame type must be lowercase")
			}
			ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
			if !ok {
				return fmt.Errorf("frame type has illegal character %q", r)
			}
		}
	}
	return nil
}

func MatchPrefix(frameType, prefix string) bool {
	if prefix == "" {
		return true
	}
	if prefix == frameType {
		return true
	}
	if strings.HasSuffix(prefix, ".") {
		return strings.HasPrefix(frameType, prefix)
	}
	return frameType == prefix || strings.HasPrefix(frameType, prefix+".")
}
