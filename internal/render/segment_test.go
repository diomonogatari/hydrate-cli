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
		got := Segment(c.level, 3, 8)
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
