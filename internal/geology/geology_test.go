package geology_test

import (
	"testing"
	"time"

	"github.com/lacsar712/mudlog/internal/clock"
	"github.com/lacsar712/mudlog/internal/geology"
)

func TestMatchesSegmentBoundary(t *testing.T) {
	cases := []struct {
		prefix    string
		frameType string
		want      bool
	}{
		{"lithology", "lithology.cuttings", true},
		{"lithology.", "lithology.cuttings", true},
		{"lithology.cuttings", "lithology.cuttings", true},
		{"", "anything.else", true},
		{"li", "lithology.cuttings", false},
		{"lith", "lithology.cuttings", false},
		{"lithology.c", "lithology.cuttings", false},
		{"gas", "lithology.cuttings", false},
		{"lithology", "lithologies.cuttings", false},
	}
	for _, tc := range cases {
		d := geology.Store{
			Enabled:      true,
			TypePrefixes: []string{tc.prefix},
		}
		got := d.Matches(tc.frameType)
		if got != tc.want {
			t.Fatalf("prefix %q vs type %q: Matches=%v want %v", tc.prefix, tc.frameType, got, tc.want)
		}
	}
}

func TestMatchesDisabledNeverFires(t *testing.T) {
	d := geology.Store{
		Enabled:      false,
		TypePrefixes: []string{""},
	}
	if d.Matches("lithology.cuttings") {
		t.Fatal("disabled store must not match")
	}
}

func TestRegistryMatchingSkipsDisabled(t *testing.T) {
	clk := clock.NewFrozen(time.Unix(0, 0))
	reg := geology.NewRegistry(clk)
	off := false
	d, err := reg.Create(geology.CreateInput{
		Name:         "off",
		URL:          "http://127.0.0.1:8080/api/v1/cuttings",
		Secret:       "abcdefgh",
		TypePrefixes: []string{"lithology"},
		Enabled:      &off,
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Enabled {
		t.Fatal("expected disabled")
	}
	got := reg.Matching("lithology.cuttings")
	if len(got) != 0 {
		t.Fatalf("disabled store leaked into matching: %+v", got)
	}
}
