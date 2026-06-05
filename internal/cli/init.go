package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"

	"github.com/diomonogatari/hydrate-cli/internal/config"
	"github.com/diomonogatari/hydrate-cli/internal/setup"
)

// cmdInit is the interactive, gh-style onboarding: a short form to tailor the
// hydration profile, then optional one-keystroke system wiring (heartbeat timer
// + zsh hook). It deliberately stops short of editing the user's shell rc or
// tmux.conf, printing those two lines instead.
func cmdInit(args []string) int {
	cfg, _ := config.Load() // seed the form from current values (or defaults)

	if !interactive() {
		// Piped / no TTY (e.g. CI): don't hang on a prompt, just ensure a config.
		if err := config.Save(cfg); err != nil {
			return fail(err)
		}
		fmt.Println("hydrate: non-interactive shell; wrote config to", config.Path())
		return 0
	}

	// Form-bound state (strings/bools; numeric inputs are validated then parsed).
	goal := strconv.Itoa(cfg.DailyGoalML)
	glass := strconv.Itoa(cfg.GlassML)
	start := strconv.Itoa(cfg.DayStartHour)
	end := strconv.Itoa(cfg.DayEndHour)
	units := cfg.Units
	floor := cfg.NotifyMinLevel
	nuclear := cfg.NuclearEscalation
	enableTimer := true
	installHook := true

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewNote().
				Title("💧 hydrate setup").
				Description("A few questions to tailor your nudges.\nType to edit, enter to accept."),
			huh.NewInput().Title("Daily goal (ml)").Value(&goal).Validate(positiveInt),
			huh.NewInput().Title("Glass size (ml)").
				Description("Default amount for `hydrate log`").
				Value(&glass).Validate(positiveInt),
			huh.NewSelect[string]().Title("Display units").
				Options(huh.NewOption("millilitres (ml)", "ml"), huh.NewOption("ounces (oz)", "oz")).
				Value(&units),
		),
		huh.NewGroup(
			huh.NewInput().Title("Waking window — start hour (0-23)").Value(&start).Validate(hourOfDay),
			huh.NewInput().Title("Waking window — end hour (0-23)").Value(&end).Validate(hourOfDay),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Title("Notify (when away) starting at…").
				Options(
					huh.NewOption("overdue — gentle, ~1.5× interval", "overdue"),
					huh.NewOption("critical only — ~2.5× interval", "critical"),
					huh.NewOption("due — earliest, 1× interval", "due"),
				).Value(&floor),
			huh.NewConfirm().Title("Recolor the whole tmux bar at critical?").
				Description("Maximally catchable, more intrusive.").Value(&nuclear),
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Enable the background heartbeat now?").
				Description("Installs + starts the systemd --user timer.").Value(&enableTimer),
			huh.NewConfirm().Title("Install the zsh activity hook?").
				Description("Lets notifications respect 'terminal in use'.").Value(&installHook),
		),
	)

	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Aborted — nothing changed.")
			return 1
		}
		return fail(err)
	}

	// Apply the (validated) answers and persist.
	cfg.DailyGoalML = atoiOr(goal, cfg.DailyGoalML)
	cfg.GlassML = atoiOr(glass, cfg.GlassML)
	cfg.DayStartHour = atoiOr(start, cfg.DayStartHour)
	cfg.DayEndHour = atoiOr(end, cfg.DayEndHour)
	cfg.Units = units
	cfg.NotifyMinLevel = floor
	cfg.NuclearEscalation = nuclear

	if err := config.Save(cfg); err != nil {
		return fail(err)
	}
	fmt.Println("\n✓ Saved config →", config.Path())

	hookPath := setup.HookPath()
	if Assets == nil {
		fmt.Fprintln(os.Stderr, "  ! embedded assets unavailable; skipping system wiring")
	} else {
		if enableTimer {
			if _, err := setup.InstallUnits(Assets); err != nil {
				fmt.Fprintln(os.Stderr, "  ! could not install units:", err)
			} else if err := setup.EnableTimer(); err != nil {
				fmt.Fprintln(os.Stderr, "  ! could not enable timer:", err)
			} else {
				fmt.Println("✓ Heartbeat enabled (systemd --user hydrate.timer)")
			}
		}
		if installHook {
			if p, err := setup.InstallHook(Assets); err != nil {
				fmt.Fprintln(os.Stderr, "  ! could not install hook:", err)
			} else {
				hookPath = p
				fmt.Println("✓ Installed zsh hook →", p)
			}
		}
	}

	printNextSteps(hookPath)
	return 0
}

func printNextSteps(hookPath string) {
	fmt.Print(`
Two lines to finish — hydrate won't edit your shell or tmux for you:

  • zsh  — add to your interactive shell config:
      source "` + hookPath + `"

  • tmux — add to tmux.conf, after any theme/tpm line:
      set -g status-interval 5
      set -g status-right-length 100   # default 40 truncates the segment
      set -ag status-right ' #(cat ${XDG_STATE_HOME:-$HOME/.local/state}/hydrate/segment 2>/dev/null)'

Then run:  hydrate log     (the bar updates within a few seconds)
`)
}

func interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func positiveInt(s string) error {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err != nil || n <= 0 {
		return errors.New("enter a positive whole number")
	}
	return nil
}

func hourOfDay(s string) error {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err != nil || n < 0 || n > 23 {
		return errors.New("enter an hour from 0 to 23")
	}
	return nil
}

func atoiOr(s string, fallback int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return n
	}
	return fallback
}
