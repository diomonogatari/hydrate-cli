package cli

import (
	"os"
	"runtime/debug"
	"testing"
)

// TestRunTopLevelDispatch guards the bug where `hydrate --help` / `-h` /
// `--version` were swallowed by the default `status` subcommand's flag parser.
func TestRunTopLevelDispatch(t *testing.T) {
	// Silence stdout for the help/version banners.
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = devnull
	t.Cleanup(func() { os.Stdout = old; devnull.Close() })

	cases := []struct {
		args []string
		want int
	}{
		{[]string{"hydrate", "--help"}, 0},
		{[]string{"hydrate", "-h"}, 0},
		{[]string{"hydrate", "help"}, 0},
		{[]string{"hydrate", "--version"}, 0},
		{[]string{"hydrate", "-v"}, 0},
		{[]string{"hydrate", "version"}, 0},
		{[]string{"hydrate", "bogus"}, 2},
	}
	for _, c := range cases {
		if got := Run(c.args); got != c.want {
			t.Errorf("Run(%v) = %d, want %d", c.args[1:], got, c.want)
		}
	}
}

func TestVersionFrom(t *testing.T) {
	mod := func(v string) *debug.BuildInfo { return &debug.BuildInfo{Main: debug.Module{Version: v}} }
	withVCS := func(rev, modified string) *debug.BuildInfo {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "(devel)"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: rev},
				{Key: "vcs.modified", Value: modified},
			},
		}
	}

	cases := []struct {
		name    string
		ldflags string
		bi      *debug.BuildInfo
		want    string
	}{
		{"ldflags wins", "1.0.2", mod("v9.9.9"), "1.0.2"},
		{"go install module version", "", mod("v1.0.2"), "v1.0.2"},
		{"devel falls to vcs (dirty)", "", withVCS("0123456789abcdef0", "true"), "dev-0123456789ab-dirty"},
		{"devel falls to vcs (clean)", "", withVCS("0123456789abcdef0", "false"), "dev-0123456789ab"},
		{"no build info", "", nil, "dev"},
	}
	for _, c := range cases {
		if got := versionFrom(c.ldflags, c.bi); got != c.want {
			t.Errorf("%s: versionFrom(%q, …) = %q, want %q", c.name, c.ldflags, got, c.want)
		}
	}
}
