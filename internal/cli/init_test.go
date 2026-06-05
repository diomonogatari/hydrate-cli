package cli

import (
	"strings"
	"testing"
)

func TestRenderInitSummary(t *testing.T) {
	steps := []initStep{
		{markOK, "config saved"},
		{markOK, "heartbeat running (systemd timer)"},
		{markSkip, "tmux already wired"},
		{markWarn, "zsh: some error"},
	}
	out := renderInitSummary(steps, summaryOpts{wired: true, hookPath: "/x/hook.zsh"})

	for _, want := range []string{
		"hydrate is ready",
		"config saved",
		"tmux already wired",
		"zsh: some error",
		"Try it",
		"hydrate log",
		"hydrate --help",
		"*.hydrate.bak", // shown because wired
	} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q\n---\n%s", want, out)
		}
	}
	// Manual block only appears when requested.
	if strings.Contains(out, "Finish wiring") {
		t.Error("manual block should be absent when manual=false")
	}

	manual := renderInitSummary(steps, summaryOpts{manual: true, hookPath: "/x/hook.zsh"})
	if !strings.Contains(manual, "Finish wiring") || !strings.Contains(manual, "/x/hook.zsh") {
		t.Errorf("manual block missing:\n%s", manual)
	}
}
