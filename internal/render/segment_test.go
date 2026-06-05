package render

import (
	"strings"
	"testing"

	"github.com/diomonogatari/hydrate-cli/internal/hydration"
)

func TestSegmentProgressAndStyle(t *testing.T) {
	cases := []struct {
		level     hydration.Level
		wantParts []string // substrings that must all be present
	}{
		{hydration.LevelOK, []string{"3/8", "fg=blue"}},
		{hydration.LevelDue, []string{"3/8", "fg=yellow"}},
		{hydration.LevelOverdue, []string{"3/8", "bold", "·due·"}},
		{hydration.LevelCritical, []string{"3/8", "blink", "bold", "DRINK WATER"}},
	}

	for _, c := range cases {
		got := Segment(c.level, 3, 8, false)
		for _, part := range c.wantParts {
			if !strings.Contains(got, part) {
				t.Errorf("Segment(%s) = %q, missing %q", c.level, got, part)
			}
		}
		if !strings.HasSuffix(got, "#[default]") {
			t.Errorf("Segment(%s) = %q, should reset styling with #[default]", c.level, got)
		}
	}
}

func TestCriticalPulseAlternates(t *testing.T) {
	off := Segment(hydration.LevelCritical, 3, 8, false)
	on := Segment(hydration.LevelCritical, 3, 8, true)

	if off == on {
		t.Fatal("critical pulse should differ between phases")
	}
	// Both phases stay loud and informative.
	for _, s := range []string{off, on} {
		for _, part := range []string{"DRINK WATER", "3/8", "blink", "bold"} {
			if !strings.Contains(s, part) {
				t.Errorf("critical phase %q missing %q", s, part)
			}
		}
	}
	// The two phases use distinct backgrounds (luminance motion).
	if strings.Contains(off, "bg=colour196") == strings.Contains(on, "bg=colour196") {
		t.Errorf("phases should use different background shades:\n off=%q\n on=%q", off, on)
	}
}

func TestPulseIgnoredBelowCritical(t *testing.T) {
	// Non-critical levels must be unaffected by the pulse flag.
	for _, lvl := range []hydration.Level{hydration.LevelOK, hydration.LevelDue, hydration.LevelOverdue} {
		if Segment(lvl, 3, 8, false) != Segment(lvl, 3, 8, true) {
			t.Errorf("pulse should not affect level %s", lvl)
		}
	}
}
