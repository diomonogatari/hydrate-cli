// Package hydration computes the user's current hydration state from config and
// the drink log. Nothing here is stored; every value is derived on demand so the
// append-only log stays the single source of truth.
package hydration

import (
	"math"
	"time"

	"github.com/diomonogatari/hydrate-cli/internal/config"
	"github.com/diomonogatari/hydrate-cli/internal/store"
)

// Level is the urgency of the next drink, escalating with time since the last.
type Level int

const (
	LevelOK Level = iota
	LevelDue
	LevelOverdue
	LevelCritical
)

func (l Level) String() string {
	switch l {
	case LevelOK:
		return "ok"
	case LevelDue:
		return "due"
	case LevelOverdue:
		return "overdue"
	case LevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// ParseLevel maps a config string to a Level.
func ParseLevel(s string) (Level, bool) {
	switch s {
	case "ok":
		return LevelOK, true
	case "due":
		return LevelDue, true
	case "overdue":
		return LevelOverdue, true
	case "critical":
		return LevelCritical, true
	default:
		return LevelOK, false
	}
}

// State is the fully derived hydration picture at a given instant.
type State struct {
	Now            time.Time
	TodayML        int           // total consumed in the current logical day
	GoalML         int           // daily target
	GlassML        int           // configured glass size
	GlassesDone    int           // round(TodayML / GlassML)
	GlassesGoal    int           // ceil(GoalML / GlassML)
	Interval       time.Duration // ideal gap between glasses across the waking window
	LastEvent      time.Time     // last drink, or the day's start (grace period)
	SinceLast      time.Duration // Now - LastEvent
	NextDue        time.Duration // Interval - SinceLast (negative once overdue)
	InWakingWindow bool          // whether Now is within [day_start, day_end)
	Level          Level
}

// Compute derives the full state from config and the raw event log at time now.
func Compute(cfg config.Config, events []store.Event, now time.Time) State {
	loc := now.Location()

	// The logical day rolls over at day_reset_hour. Everything logged since the
	// most recent reset boundary counts as "today".
	reset := ResetBoundary(cfg, now)
	// The waking window's start/end live on the logical day's calendar date,
	// which is the reset boundary's date (reset 4h < start 7h < end 23h).
	dayStart := atHour(reset, cfg.DayStartHour)
	dayEnd := atHour(reset, cfg.DayEndHour)

	// Sum today's intake and find the most recent drink.
	todayML := 0
	var lastDrink time.Time
	for _, e := range events {
		if e.TS < reset.Unix() {
			continue
		}
		todayML += e.ML
		t := time.Unix(e.TS, 0).In(loc)
		if t.After(lastDrink) {
			lastDrink = t
		}
	}

	glassesGoal := ceilDiv(cfg.DailyGoalML, cfg.GlassML)
	if glassesGoal < 1 {
		glassesGoal = 1
	}
	windowSec := (cfg.DayEndHour - cfg.DayStartHour) * 3600
	if windowSec <= 0 {
		windowSec = 16 * 3600 // defensive: a sane 16h window
	}
	interval := time.Duration(windowSec/glassesGoal) * time.Second

	// last_event = max(last drink today, the day's start). Starting the clock at
	// day_start gives a one-interval grace period each morning.
	lastEvent := dayStart
	if lastDrink.After(lastEvent) {
		lastEvent = lastDrink
	}
	sinceLast := now.Sub(lastEvent)

	inWaking := !now.Before(dayStart) && now.Before(dayEnd)

	state := State{
		Now:            now,
		TodayML:        todayML,
		GoalML:         cfg.DailyGoalML,
		GlassML:        cfg.GlassML,
		GlassesDone:    int(math.Round(float64(todayML) / float64(cfg.GlassML))),
		GlassesGoal:    glassesGoal,
		Interval:       interval,
		LastEvent:      lastEvent,
		SinceLast:      sinceLast,
		NextDue:        interval - sinceLast,
		InWakingWindow: inWaking,
		Level:          levelFor(sinceLast, interval, inWaking),
	}
	return state
}

// levelFor maps elapsed time against the interval. Outside the waking window the
// user is asleep/winding down, so urgency is pinned to OK — the tool never nags
// at 3am.
func levelFor(since, interval time.Duration, inWaking bool) Level {
	if !inWaking || interval <= 0 {
		return LevelOK
	}
	r := since.Seconds() / interval.Seconds()
	switch {
	case r < 1.0:
		return LevelOK
	case r < 1.5:
		return LevelDue
	case r < 2.5:
		return LevelOverdue
	default:
		return LevelCritical
	}
}

// ResetBoundary returns the most recent day_reset_hour boundary at or before
// now. Events at or after it belong to the current logical day.
func ResetBoundary(cfg config.Config, now time.Time) time.Time {
	reset := atHour(now, cfg.DayResetHour)
	if now.Before(reset) {
		reset = reset.AddDate(0, 0, -1)
	}
	return reset
}

// atHour returns t's calendar date at the given hour, in t's location.
func atHour(t time.Time, hour int) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, hour, 0, 0, 0, t.Location())
}

func ceilDiv(a, b int) int {
	if b <= 0 {
		return 0
	}
	return (a + b - 1) / b
}
