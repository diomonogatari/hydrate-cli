// Package focus optionally reports whether the user's terminal is the focused
// window. It is a best-effort secondary gate: it must degrade to "unknown"
// (nil) on Wayland, when tools are missing, or on any error, and callers must
// never block a notification on it.
package focus

import (
	"os"
	"os/exec"
	"strings"
)

// terminalClasses are the window classes treated as "the terminal".
var terminalClasses = []string{"kitty", "xterm-kitty"}

// Probe returns a pointer to whether the terminal is focused, or nil when it
// cannot be determined.
func Probe() *bool {
	// There is no reliable cross-application active-window query under Wayland —
	// xdotool only sees Xwayland clients — so report unknown.
	if strings.EqualFold(os.Getenv("XDG_SESSION_TYPE"), "wayland") {
		return nil
	}
	path, err := exec.LookPath("xdotool")
	if err != nil {
		return nil
	}
	out, err := exec.Command(path, "getactivewindow", "getwindowclassname").Output()
	if err != nil {
		return nil
	}
	class := strings.TrimSpace(string(out))
	if class == "" {
		return nil
	}
	for _, tc := range terminalClasses {
		if strings.EqualFold(class, tc) {
			focused := true
			return &focused
		}
	}
	focused := false
	return &focused
}
