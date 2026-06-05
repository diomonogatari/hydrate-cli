package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"time"

	"github.com/diomonogatari/hydrate-cli/internal/config"
	"github.com/diomonogatari/hydrate-cli/internal/focus"
	"github.com/diomonogatari/hydrate-cli/internal/format"
	"github.com/diomonogatari/hydrate-cli/internal/hydration"
	"github.com/diomonogatari/hydrate-cli/internal/notify"
	"github.com/diomonogatari/hydrate-cli/internal/render"
	"github.com/diomonogatari/hydrate-cli/internal/store"
)

func cmdStatus(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "machine-readable output")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, st, err := loadState()
	if err != nil {
		return fail(err)
	}

	if *asJSON {
		return printJSON(statusPayload(cfg, st))
	}

	percent := percentOf(st.TodayML, st.GoalML)
	fmt.Printf("💧 %d/%d glasses · %s / %s (%d%%)\n",
		st.GlassesDone, st.GlassesGoal,
		displayVolume(st.TodayML, cfg.Units), displayVolume(st.GoalML, cfg.Units), percent)
	fmt.Printf("%s %d%%\n", progressBar(float64(st.TodayML)/float64(max1(st.GoalML)), 20), percent)

	next := "now"
	if st.NextDue > 0 {
		next = "in " + format.HumanizeDuration(st.NextDue)
	}
	fmt.Printf("last drink %s · next due %s · status: %s\n",
		format.HumanizeDuration(st.SinceLast), next, st.Level)
	if !st.InWakingWindow {
		fmt.Println("(outside your waking window — resting)")
	}
	return 0
}

func cmdLog(args []string) int {
	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "machine-readable output")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Load()
	if err != nil {
		return fail(fmt.Errorf("reading config: %w", err))
	}

	ml, err := parseAmount(fs.Arg(0), cfg)
	if err != nil {
		return fail(err)
	}

	if err := store.AppendEvent(store.Event{TS: time.Now().Unix(), ML: ml}); err != nil {
		return fail(fmt.Errorf("logging drink: %w", err))
	}

	// Recompute and calm the bar immediately.
	events, _ := store.LoadEvents()
	st := hydration.Compute(cfg, events, time.Now())
	refreshSegment(st)
	calmNotifyLevel(st)

	if *asJSON {
		return printJSON(statusPayload(cfg, st))
	}
	fmt.Printf("Logged %s — %d/%d today (%s / %s)\n",
		displayVolume(ml, cfg.Units), st.GlassesDone, st.GlassesGoal,
		displayVolume(st.TodayML, cfg.Units), displayVolume(st.GoalML, cfg.Units))
	return 0
}

func cmdUndo(args []string) int {
	fs := flag.NewFlagSet("undo", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "machine-readable output")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Load()
	if err != nil {
		return fail(fmt.Errorf("reading config: %w", err))
	}

	boundary := hydration.ResetBoundary(cfg, time.Now()).Unix()
	removed, ok, err := store.RemoveLastSince(boundary)
	if err != nil {
		return fail(fmt.Errorf("undoing: %w", err))
	}

	events, _ := store.LoadEvents()
	st := hydration.Compute(cfg, events, time.Now())
	refreshSegment(st)
	calmNotifyLevel(st)

	if *asJSON {
		return printJSON(map[string]any{
			"removed": ok,
			"amount":  removed.ML,
			"status":  statusPayload(cfg, st),
		})
	}
	if !ok {
		fmt.Println("Nothing to undo today.")
		return 0
	}
	at := time.Unix(removed.TS, 0).Local().Format("15:04")
	fmt.Printf("Removed %s logged at %s — now %d/%d today.\n",
		displayVolume(removed.ML, cfg.Units), at, st.GlassesDone, st.GlassesGoal)
	return 0
}

func cmdSegment(args []string) int {
	_, st, err := loadState()
	if err != nil {
		return fail(err)
	}
	fmt.Println(render.Segment(st.Level, st.GlassesDone, st.GlassesGoal))
	return 0
}

// cmdTick is the heartbeat run by the systemd --user timer (and safe to run by
// hand). It recomputes state, refreshes the cached tmux segment, and nudges any
// running tmux. It is silent on success so journal logs stay quiet; later phases
// hang the notification decision off this same path.
func cmdTick(args []string) int {
	fs := flag.NewFlagSet("tick", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "machine-readable output")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, st, err := loadState()
	if err != nil {
		return fail(err)
	}
	refreshSegment(st)
	decision, sent := maybeNotify(cfg, st)

	if *asJSON {
		payload := statusPayload(cfg, st)
		payload["notify_sent"] = sent
		payload["notify_reason"] = decision.Reason
		return printJSON(payload)
	}
	return 0
}

// maybeNotify gathers the live inputs, asks the policy whether to notify, and
// (if so) delivers and records it. Failures are reported but never abort the
// heartbeat. It returns the decision and whether a notification was sent.
func maybeNotify(cfg config.Config, st hydration.State) (notify.Decision, bool) {
	minLevel, ok := hydration.ParseLevel(cfg.NotifyMinLevel)
	if !ok {
		minLevel = hydration.LevelOverdue
	}
	prev := store.LoadNotifyState()
	prevLevel, _ := hydration.ParseLevel(prev.LastNotifiedLevel)
	lastAct, haveAct := store.ReadActivity()

	decision := notify.Decide(notify.Inputs{
		Level:            st.Level,
		MinLevel:         minLevel,
		Now:              st.Now.Unix(),
		LastActivity:     lastAct,
		HaveActivity:     haveAct,
		IdleThresholdSec: cfg.IdleThresholdSec,
		Focused:          focus.Probe(),
		PrevLevel:        prevLevel,
		PrevNotifyTS:     prev.LastNotifyTS,
		CooldownSec:      cfg.NotifyCooldownSec,
	})
	if !decision.Send {
		return decision, false
	}

	summary, body := notify.Compose(st)
	id, err := notify.Send(notify.Message{
		Summary:    summary,
		Body:       body,
		ReplacesID: prev.LastNotifyID,
		Critical:   st.Level == hydration.LevelCritical,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "hydrate: notify failed:", err)
		return decision, false
	}
	_ = store.SaveNotifyState(store.NotifyState{
		LastNotifyTS:      st.Now.Unix(),
		LastNotifiedLevel: st.Level.String(),
		LastNotifyID:      id,
	})
	return decision, true
}

// calmNotifyLevel re-arms escalation after the user acts: it records the (now
// lower) level as the last-notified one, so a fresh climb past the floor counts
// as an escalation rather than waiting out a full cooldown. The timestamp and id
// are preserved so cooldown still applies to repeat nudges at the same level.
func calmNotifyLevel(st hydration.State) {
	ns := store.LoadNotifyState()
	ns.LastNotifiedLevel = st.Level.String()
	_ = store.SaveNotifyState(ns)
}

func cmdConfig(args []string) int {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	edit := fs.Bool("edit", false, "open the config file in $EDITOR")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	path, err := config.EnsureExists()
	if err != nil {
		return fail(fmt.Errorf("preparing config: %w", err))
	}

	if *edit {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		cmd := exec.Command(editor, path)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fail(fmt.Errorf("running %s: %w", editor, err))
		}
		return 0
	}

	cfg, err := config.Load()
	if err != nil {
		return fail(err)
	}
	fmt.Println("config:", path)
	fmt.Printf("  daily_goal_ml       = %d\n", cfg.DailyGoalML)
	fmt.Printf("  glass_ml            = %d\n", cfg.GlassML)
	fmt.Printf("  day_start_hour      = %d\n", cfg.DayStartHour)
	fmt.Printf("  day_end_hour        = %d\n", cfg.DayEndHour)
	fmt.Printf("  day_reset_hour      = %d\n", cfg.DayResetHour)
	fmt.Printf("  idle_threshold_sec  = %d\n", cfg.IdleThresholdSec)
	fmt.Printf("  notify_min_level    = %q\n", cfg.NotifyMinLevel)
	fmt.Printf("  notify_cooldown_sec = %d\n", cfg.NotifyCooldownSec)
	fmt.Printf("  units               = %q\n", cfg.Units)
	return 0
}

// --- shared output helpers ---

func statusPayload(cfg config.Config, st hydration.State) map[string]any {
	return map[string]any{
		"today_ml":         st.TodayML,
		"goal_ml":          st.GoalML,
		"glasses_done":     st.GlassesDone,
		"glasses_goal":     st.GlassesGoal,
		"percent":          percentOf(st.TodayML, st.GoalML),
		"since_last_sec":   int64(st.SinceLast.Seconds()),
		"next_due_sec":     int64(st.NextDue.Seconds()),
		"interval_sec":     int64(st.Interval.Seconds()),
		"level":            st.Level.String(),
		"in_waking_window": st.InWakingWindow,
		"units":            cfg.Units,
	}
}

func printJSON(v any) int {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fail(err)
	}
	return 0
}

func percentOf(part, whole int) int {
	if whole <= 0 {
		return 0
	}
	return int(math.Round(float64(part) / float64(whole) * 100))
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "hydrate:", err)
	return 1
}
