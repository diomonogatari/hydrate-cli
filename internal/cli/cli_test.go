package cli

import (
	"os"
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
