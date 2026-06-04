package cli

import (
	"testing"

	"github.com/diomonogatari/hydrate-cli/internal/config"
)

func TestParseAmount(t *testing.T) {
	cfg := config.Default() // glass_ml = 250

	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"", 250, false},      // default glass
		{"500", 500, false},   // bare ml
		{"500ml", 500, false}, // explicit ml
		{"16oz", 473, false},  // round(16 * 29.5735)
		{"1l", 1000, false},   // litres
		{"0.5l", 500, false},  // fractional litres
		{" 300 ml ", 300, false},
		{"0", 0, true},
		{"-100", 0, true},
		{"abc", 0, true},
	}

	for _, c := range cases {
		got, err := parseAmount(c.in, cfg)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseAmount(%q) = %d, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseAmount(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseAmount(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
