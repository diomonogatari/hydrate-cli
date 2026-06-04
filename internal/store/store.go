// Package store owns hydrate's on-disk state: the append-only JSONL drink log
// (the single source of truth) and the pre-rendered tmux segment cache. State
// is derived from the log; no mutable counter is ever kept.
package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/diomonogatari/hydrate-cli/internal/paths"
)

// Event is one logged drink. Kept intentionally tiny and greppable:
//
//	{"ts": 1780596736, "ml": 250}
type Event struct {
	TS int64 `json:"ts"` // unix seconds
	ML int   `json:"ml"` // amount in millilitres
}

// AppendEvent adds one drink to the log, creating the state dir if needed.
func AppendEvent(e Event) error {
	if err := os.MkdirAll(paths.StateDir(), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(paths.LogFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

// LoadEvents returns every logged drink in file order. A missing log is not an
// error: it simply means nothing has been logged yet.
func LoadEvents() ([]Event, error) {
	f, err := os.Open(paths.LogFile())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Event
		if err := json.Unmarshal(line, &e); err != nil {
			// Skip a corrupt line rather than losing the whole history.
			continue
		}
		events = append(events, e)
	}
	return events, sc.Err()
}

// RemoveLastSince deletes the most recent event whose timestamp is at or after
// boundary (i.e. the last drink in the current logical day) and rewrites the
// log atomically. It reports the removed event and whether anything was removed.
func RemoveLastSince(boundary int64) (Event, bool, error) {
	events, err := LoadEvents()
	if err != nil {
		return Event{}, false, err
	}

	idx := -1
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].TS >= boundary {
			idx = i
			break
		}
	}
	if idx == -1 {
		return Event{}, false, nil
	}

	removed := events[idx]
	events = append(events[:idx], events[idx+1:]...)
	if err := rewriteLog(events); err != nil {
		return removed, false, err
	}
	return removed, true, nil
}

func rewriteLog(events []Event) error {
	var buf []byte
	for _, e := range events {
		line, err := json.Marshal(e)
		if err != nil {
			return err
		}
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}
	return atomicWrite(paths.LogFile(), buf, 0o644)
}

// WriteSegment caches the styled tmux string for the status-bar hot path.
func WriteSegment(s string) error {
	if err := os.MkdirAll(paths.StateDir(), 0o755); err != nil {
		return err
	}
	return atomicWrite(paths.SegmentFile(), []byte(s+"\n"), 0o644)
}

// ReadActivity returns the last shell-activity timestamp written by the zsh
// hook, and whether it was available. The hook writes a bare unix-seconds
// integer with no trailing newline, but we trim defensively.
func ReadActivity() (int64, bool) {
	data, err := os.ReadFile(paths.ActivityFile())
	if err != nil {
		return 0, false
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, false
	}
	return ts, true
}

// atomicWrite writes via a temp file in the same directory followed by rename,
// so readers never observe a half-written file.
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
