package frame_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/lacsar712/mudlog/internal/frame"
)

func TestParseAndPrefix(t *testing.T) {
	env, err := frame.Parse([]byte(`{"type":"lithology.cuttings","wellbore":"W-12H","lithology":{"code":"SS","percent":80},"lag_depth":{"measured_m":1842.5},"payload":{"rop_m_hr":12}}`))
	if err != nil {
		t.Fatal(err)
	}
	if env.Type != "lithology.cuttings" {
		t.Fatalf("type %s", env.Type)
	}
	if env.Lithology.Code != "SS" {
		t.Fatalf("lithology %s", env.Lithology.Code)
	}
	if env.LagDepth.MeasuredM != 1842.5 {
		t.Fatalf("lag %v", env.LagDepth.MeasuredM)
	}
	if !frame.MatchPrefix("lithology.cuttings", "lithology") {
		t.Fatal("lithology should match lithology.cuttings")
	}
	if frame.MatchPrefix("lithology.cuttings", "gas") {
		t.Fatal("gas should not match")
	}
	if _, err := frame.Parse([]byte(`{"type":"Lithology.Cuttings","payload":{}}`)); err == nil {
		t.Fatal("uppercase type should fail")
	}
}

func TestMatchPrefixSegmentBoundary(t *testing.T) {
	cases := []struct {
		frameType string
		prefix    string
		want      bool
	}{
		{"lithology.cuttings", "lithology", true},
		{"lithology.cuttings", "lithology.", true},
		{"lithology.cuttings", "lithology.cuttings", true},
		{"lithology.cuttings", "", true},
		{"lithology.cuttings", "li", false},
		{"lithology.cuttings", "lith", false},
		{"lithology.cuttings", "lithology.c", false},
		{"lithology.cuttings", "gas", false},
		{"lithologies.cuttings", "lithology", false},
	}
	for _, tc := range cases {
		got := frame.MatchPrefix(tc.frameType, tc.prefix)
		if got != tc.want {
			t.Fatalf("MatchPrefix(%q, %q)=%v want %v", tc.frameType, tc.prefix, got, tc.want)
		}
	}
}

func TestParseWrapsSyntaxError(t *testing.T) {
	_, err := frame.Parse([]byte(`{`))
	if err == nil {
		t.Fatal("expected syntax error")
	}
	var syn *json.SyntaxError
	if !errors.As(err, &syn) {
		t.Fatalf("want json.SyntaxError via errors.As, got %v", err)
	}
}
