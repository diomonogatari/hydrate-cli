package notify

import (
	"testing"
	"time"

	"github.com/diomonogatari/hydrate-cli/internal/hydration"
)

func boolPtr(b bool) *bool { return &b }

func TestDecide(t *testing.T) {
	// A baseline "should fire": overdue, floor overdue, terminal idle, no focus
	// info, no prior notification, cooldown long elapsed.
	base := Inputs{
		Level:            hydration.LevelOverdue,
		MinLevel:         hydration.LevelOverdue,
		Now:              10_000,
		LastActivity:     0, // 10000s ago
		HaveActivity:     true,
		IdleThresholdSec: 600,
		Focused:          nil,
		PrevLevel:        hydration.LevelOK,
		PrevNotifyTS:     0,
		CooldownSec:      1800,
	}

	cases := []struct {
		name     string
		mutate   func(in *Inputs)
		wantSend bool
	}{
		{"fires when away and escalated", nil, true},
		{"below floor stays silent", func(in *Inputs) { in.Level = hydration.LevelDue }, false},
		{"terminal in use suppresses", func(in *Inputs) { in.LastActivity = in.Now - 60 }, false},
		{"missing activity treated as away", func(in *Inputs) { in.HaveActivity = false }, true},
		{"focused terminal suppresses", func(in *Inputs) { in.Focused = boolPtr(true) }, false},
		{"unfocused does not suppress", func(in *Inputs) { in.Focused = boolPtr(false) }, true},
		{
			"within cooldown and not escalated stays silent",
			func(in *Inputs) {
				in.PrevLevel = hydration.LevelOverdue // same level => not escalated
				in.PrevNotifyTS = in.Now - 100        // 100s < 1800s cooldown
			},
			false,
		},
		{
			"cooldown elapsed re-fires",
			func(in *Inputs) {
				in.PrevLevel = hydration.LevelOverdue
				in.PrevNotifyTS = in.Now - 2000 // > cooldown
			},
			true,
		},
		{
			"escalation beats cooldown",
			func(in *Inputs) {
				in.Level = hydration.LevelCritical
				in.PrevLevel = hydration.LevelOverdue
				in.PrevNotifyTS = in.Now - 1 // basically just notified
			},
			true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := base
			if c.mutate != nil {
				c.mutate(&in)
			}
			got := Decide(in)
			if got.Send != c.wantSend {
				t.Errorf("Decide().Send = %v (%s), want %v", got.Send, got.Reason, c.wantSend)
			}
		})
	}
}

func TestCompose(t *testing.T) {
	st := hydration.State{
		SinceLast:   100 * time.Minute, // 1h 40m
		GlassesDone: 2,
		GlassesGoal: 8,
	}
	summary, body := Compose(st)
	if summary == "" {
		t.Error("summary should not be empty")
	}
	want := "It's been 1h 40m — 2/8 glasses today."
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}
