package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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
	enableTimer := true
	autoWire := true

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
		),
		huh.NewGroup(
			huh.NewConfirm().Title("Enable the background heartbeat now?").
				Description("Installs + starts the systemd --user timer.").Value(&enableTimer),
			huh.NewConfirm().Title("Wire your shell + tmux for you?").
				Description("Adds the hook to your zshrc and the segment to tmux.conf (.bak backups).").Value(&autoWire),
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

	if err := config.Save(cfg); err != nil {
		return fail(err)
	}
	fmt.Println("\n✓ Saved config →", config.Path())

	// Gather outcomes, then render one cohesive, styled summary.
	steps := []initStep{{markOK, "config saved"}}
	hookPath := setup.HookPath()
	var wiredFiles, needManual bool

	if Assets == nil {
		steps = append(steps, initStep{markWarn, "system wiring skipped (no embedded assets)"})
		printInitSummary(steps, summaryOpts{manual: true, hookPath: hookPath})
		return 0
	}

	switch {
	case !enableTimer:
		steps = append(steps, initStep{markSkip, "heartbeat left off"})
	default:
		if _, err := setup.InstallUnits(Assets); err != nil {
			steps = append(steps, initStep{markWarn, "heartbeat: " + short(err)})
		} else if err := setup.EnableTimer(); err != nil {
			steps = append(steps, initStep{markWarn, "heartbeat: " + short(err)})
		} else {
			steps = append(steps, initStep{markOK, "heartbeat running (systemd timer)"})
		}
	}

	// Always lay down the hook file so a `source` line has a target.
	if p, err := setup.InstallHook(Assets); err != nil {
		steps = append(steps, initStep{markWarn, "zsh hook: " + short(err)})
	} else {
		hookPath = p
		steps = append(steps, initStep{markOK, "zsh hook installed"})
	}

	if autoWire {
		if r, err := setup.WireZsh(hookPath); err != nil {
			steps, needManual = append(steps, initStep{markWarn, "zsh: " + short(err)}), true
		} else if r.Changed {
			steps, wiredFiles = append(steps, initStep{markOK, "zsh wired"}), true
		} else {
			steps = append(steps, initStep{markSkip, "zsh already wired"})
		}

		if r, err := setup.WireTmux(); err != nil {
			steps, needManual = append(steps, initStep{markWarn, "tmux: " + short(err)}), true
		} else if r.Changed {
			steps, wiredFiles = append(steps, initStep{markOK, "tmux wired"}), true
			setup.ReloadTmux(r.Path)
		} else {
			steps = append(steps, initStep{markSkip, "tmux already wired"})
		}
	} else {
		needManual = true
	}

	printInitSummary(steps, summaryOpts{manual: needManual, wired: wiredFiles, hookPath: hookPath})
	return 0
}

type initStep struct{ mark, text string }

const (
	markOK   = "ok"
	markSkip = "skip"
	markWarn = "warn"
)

type summaryOpts struct {
	manual   bool
	wired    bool
	hookPath string
}

var (
	stTitle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	stHead   = lipgloss.NewStyle().Bold(true)
	stOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	stSkip   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	stWarn   = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	stDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	stCmd    = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
	stCmdCol = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true).Width(16)
)

func printInitSummary(steps []initStep, o summaryOpts) {
	fmt.Print(renderInitSummary(steps, o))
}

func renderInitSummary(steps []initStep, o summaryOpts) string {
	var b strings.Builder
	line := func(s string) { b.WriteString(s + "\n") }

	b.WriteByte('\n')
	line(stTitle.Render("💧 hydrate is ready"))
	b.WriteByte('\n')
	for _, s := range steps {
		mark := stSkip.Render("•")
		switch s.mark {
		case markOK:
			mark = stOK.Render("✓")
		case markWarn:
			mark = stWarn.Render("!")
		}
		line("  " + mark + " " + s.text)
	}

	b.WriteByte('\n')
	line(stHead.Render("Try it"))
	line("  " + stCmdCol.Render("hydrate log") + stDim.Render("log a glass"))
	line("  " + stCmdCol.Render("hydrate") + stDim.Render("today's progress"))
	line("  " + stCmdCol.Render("hydrate --help") + stDim.Render("all commands"))

	if o.manual {
		b.WriteByte('\n')
		line(stHead.Render("Finish wiring") + stDim.Render("  (add these yourself)"))
		line("  " + stDim.Render(`zsh   source "`+o.hookPath+`"`))
		line("  " + stDim.Render(`tmux  set -g status-interval 5`))
		line("  " + stDim.Render(`      set -g status-right-length 100`))
		line("  " + stDim.Render(`      set -ag status-right ' #(cat ${XDG_STATE_HOME:-$HOME/.local/state}/hydrate/segment 2>/dev/null)'`))
	}

	b.WriteByte('\n')
	foot := stDim.Render("New shell adds shortcuts ") + stCmd.Render("w") + stDim.Render(" (log a glass) and ") +
		stCmd.Render("ww") + stDim.Render(" (status).")
	if o.wired {
		foot += stDim.Render("  ·  backups: *.hydrate.bak")
	}
	line(foot)
	return b.String()
}

func short(err error) string {
	s := err.Error()
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 60 {
		s = s[:57] + "…"
	}
	return s
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
