package store

import (
	"os"
	"testing"

	"github.com/diomonogatari/hydrate-cli/internal/paths"
)

// sandbox points the XDG state root at a temp dir for the duration of a test.
func sandbox(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
}

func TestEventRoundTrip(t *testing.T) {
	sandbox(t)

	// A missing log is empty, not an error.
	got, err := LoadEvents()
	if err != nil {
		t.Fatalf("LoadEvents on missing log: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 events, got %d", len(got))
	}

	want := []Event{{TS: 100, ML: 250}, {TS: 200, ML: 500}, {TS: 300, ML: 250}}
	for _, e := range want {
		if err := AppendEvent(e); err != nil {
			t.Fatalf("AppendEvent: %v", err)
		}
	}

	got, err = LoadEvents()
	if err != nil {
		t.Fatalf("LoadEvents: %v", err)
	}
	if len(got) != 3 || got[0] != want[0] || got[2] != want[2] {
		t.Fatalf("round trip mismatch: got %+v", got)
	}
}

func TestRemoveLastSince(t *testing.T) {
	sandbox(t)
	for _, e := range []Event{{TS: 100, ML: 250}, {TS: 200, ML: 500}, {TS: 300, ML: 250}} {
		if err := AppendEvent(e); err != nil {
			t.Fatalf("AppendEvent: %v", err)
		}
	}

	// boundary 250 -> only the ts=300 event qualifies as "today".
	removed, ok, err := RemoveLastSince(250)
	if err != nil || !ok {
		t.Fatalf("RemoveLastSince: ok=%v err=%v", ok, err)
	}
	if removed.TS != 300 {
		t.Fatalf("removed ts = %d, want 300", removed.TS)
	}

	got, _ := LoadEvents()
	if len(got) != 2 {
		t.Fatalf("after remove, len = %d, want 2", len(got))
	}

	// Nothing left at/after the boundary -> no-op.
	_, ok, err = RemoveLastSince(250)
	if err != nil {
		t.Fatalf("second RemoveLastSince err: %v", err)
	}
	if ok {
		t.Fatal("expected no removal when nothing qualifies")
	}
}

func TestWriteSegment(t *testing.T) {
	sandbox(t)
	const seg = "#[fg=blue]water#[default]"
	if err := WriteSegment(seg); err != nil {
		t.Fatalf("WriteSegment: %v", err)
	}
	data, err := os.ReadFile(paths.SegmentFile())
	if err != nil {
		t.Fatalf("read segment: %v", err)
	}
	if string(data) != seg+"\n" {
		t.Fatalf("segment file = %q, want %q", string(data), seg+"\n")
	}
}

func TestReadActivity(t *testing.T) {
	sandbox(t)

	if _, ok := ReadActivity(); ok {
		t.Fatal("expected no activity before any is written")
	}

	if err := os.MkdirAll(paths.StateDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.ActivityFile(), []byte("1780600000"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts, ok := ReadActivity()
	if !ok || ts != 1780600000 {
		t.Fatalf("ReadActivity = (%d, %v), want (1780600000, true)", ts, ok)
	}
}
