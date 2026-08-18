package pack

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/lacsar712/mudlog/internal/channel"
	"github.com/lacsar712/mudlog/internal/frame"
	"github.com/lacsar712/mudlog/internal/gas"
)

// Record is a compact WITS-style row sent to a geology store.
type Record struct {
	At       time.Time         `json:"at"`
	Wellbore string            `json:"wellbore"`
	Type     string            `json:"type"`
	Channels map[string]string `json:"channels"`
	LithCode string            `json:"lith_code,omitempty"`
	LagMD    float64           `json:"lag_md_m,omitempty"`
	GasShow  string            `json:"gas_show,omitempty"`
}

func FromFrame(fr frame.Frame, now time.Time) (Record, error) {
	if fr.Type == "" {
		return Record{}, fmt.Errorf("frame type required")
	}
	rec := Record{
		At:       now,
		Wellbore: fr.Wellbore,
		Type:     fr.Type,
		Channels: map[string]string{},
		LithCode: fr.Lithology.Code,
		LagMD:    fr.LagDepth.MeasuredM,
	}
	if len(fr.Payload) > 0 && string(fr.Payload) != "null" {
		var payload map[string]any
		if err := json.Unmarshal(fr.Payload, &payload); err != nil {
			return Record{}, fmt.Errorf("payload: %w", err)
		}
		if code, val, ok := channel.PickNumeric(payload); ok {
			rec.Channels[code] = strconv.FormatFloat(val, 'f', -1, 64)
		}
		if g, ok := payloadGas(payload); ok {
			rec.GasShow = g.Character()
		}
	}
	if rec.LithCode != "" {
		rec.Channels[channel.Lith1] = rec.LithCode
	}
	if rec.LagMD > 0 {
		rec.Channels[channel.LagMD] = strconv.FormatFloat(rec.LagMD, 'f', 2, 64)
	}
	return rec, nil
}

func payloadGas(payload map[string]any) (gas.Show, bool) {
	raw, ok := payload["gas"]
	if !ok {
		return gas.Show{}, false
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return gas.Show{}, false
	}
	var s gas.Show
	if err := json.Unmarshal(b, &s); err != nil {
		return gas.Show{}, false
	}
	if s.Total() <= 0 {
		return gas.Show{}, false
	}
	return s, true
}
