package hydration

import (
	"testing"
	"time"

	"github.com/diomonogatari/hydrate-cli/internal/config"
	"github.com/diomonogatari/hydrate-cli/internal/store"
)

func ev(t time.Time, ml int) store.Event { return store.Event{TS: t.Unix(), ML: ml} }

func TestComputeBasics(t *testing.T) {
	cfg := config.Default() // goal 2000, glass 250 -> 8 glasses, window 16h -> 2h interval
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	events := []store.Event{
		ev(time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC), 250),
		ev(time.Date(2026, 6, 5, 11, 0, 0, 0, time.UTC), 500),
		// Yesterday's drink must not count toward today.
		ev(time.Date(2026, 6, 4, 22, 0, 0, 0, time.UTC), 999),
	}

	st := Compute(cfg, events, now)

	if st.TodayML != 750 {
		t.Errorf("TodayML = %d, want 750", st.TodayML)
	}
	if st.GlassesGoal != 8 {
		t.Errorf("GlassesGoal = %d, want 8", st.GlassesGoal)
	}
	if st.Interval != 2*time.Hour {
		t.Errorf("Interval = %s, want 2h", st.Interval)
	}
	if st.GlassesDone != 3 { // round(750/250)
		t.Errorf("GlassesDone = %d, want 3", st.GlassesDone)
	}
	if !st.InWakingWindow {
		t.Error("expected to be in waking window at noon")
	}
}

func TestLevels(t *testing.T) {
	cfg := config.Default() // 2h interval
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		lastDrink time.Time // zero => no drinks today
		want      Level
	}{
		{"ok", time.Date(2026, 6, 5, 11, 0, 0, 0, time.UTC), LevelOK},           // 1h < 2h
		{"due", time.Date(2026, 6, 5, 9, 30, 0, 0, time.UTC), LevelDue},         // 2.5h -> 1.25x
		{"overdue", time.Date(2026, 6, 5, 8, 30, 0, 0, time.UTC), LevelOverdue}, // 3.5h -> 1.75x
		{"critical-none", time.Time{}, LevelCritical},                           // since day start 07:00 = 5h -> 2.5x
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var events []store.Event
			if !c.lastDrink.IsZero() {
				events = append(events, ev(c.lastDrink, 250))
			}
			st := Compute(cfg, events, now)
			if st.Level != c.want {
				t.Errorf("Level = %s, want %s (since_last=%s, interval=%s)",
					st.Level, c.want, st.SinceLast, st.Interval)
			}
		})
	}
}

func TestAsleepIsCalm(t *testing.T) {
	cfg := config.Default()
	now := time.Date(2026, 6, 5, 2, 0, 0, 0, time.UTC) // before waking window
	st := Compute(cfg, nil, now)
	if st.Level != LevelOK {
		t.Errorf("Level at 2am = %s, want ok", st.Level)
	}
	if st.InWakingWindow {
		t.Error("2am should not be in the waking window")
	}
}

func TestDailyTotals(t *testing.T) {
	cfg := config.Default() // reset hour 4
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	events := []store.Event{
		ev(time.Date(2026, 6, 5, 9, 0, 0, 0, time.UTC), 250),  // today
		ev(time.Date(2026, 6, 5, 11, 0, 0, 0, time.UTC), 500), // today
		ev(time.Date(2026, 6, 4, 20, 0, 0, 0, time.UTC), 750), // yesterday
		ev(time.Date(2026, 6, 5, 2, 0, 0, 0, time.UTC), 100),  // 2am -> belongs to JUNE 4 logical day
		ev(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), 999),  // far outside the window
	}

	totals := DailyTotals(cfg, events, now, 3)
	if len(totals) != 3 {
		t.Fatalf("want 3 days, got %d", len(totals))
	}
	// Oldest first; last is today.
	if got := totals[2].ML; got != 750 { // today: 250 + 500
		t.Errorf("today total = %d, want 750", got)
	}
	if got := totals[1].ML; got != 850 { // yesterday: 750 + the 2am-of-the-5th glass
		t.Errorf("yesterday total = %d, want 850", got)
	}
	if got := totals[0].ML; got != 0 {
		t.Errorf("two days ago total = %d, want 0", got)
	}
	// Day ordering is contiguous and chronological.
	if !totals[2].Day.Equal(ResetBoundary(cfg, now)) {
		t.Errorf("last bucket should be today's reset boundary")
	}
}

func TestResetBoundary(t *testing.T) {
	cfg := config.Default() // reset hour 4
	// 01:00 belongs to the previous logical day -> boundary is yesterday 04:00.
	now := time.Date(2026, 6, 5, 1, 0, 0, 0, time.UTC)
	got := ResetBoundary(cfg, now)
	want := time.Date(2026, 6, 4, 4, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("ResetBoundary = %s, want %s", got, want)
	}
}
