package logger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestConcurrentAppend_NoInterleavingOrTearing proves the property
// ADR-0017 Decision #5 rests on: many independently-opened writers
// (simulating separate aiwf processes, e.g. concurrent worktrees)
// appending to the same daily log file at once never interleave or
// tear each other's lines. Each writer gets its own *os.File — its
// own file descriptor, its own slog.Logger, its own internal buffer —
// so it's the OS-level O_APPEND guarantee under test, not any
// in-process serialization a shared handler's mutex would provide.
func TestConcurrentAppend_NoInterleavingOrTearing(t *testing.T) {
	t.Parallel()
	const writers = 40
	const recordsPerWriter = 25

	path := filepath.Join(t.TempDir(), "concurrent.log")

	var wg sync.WaitGroup
	for gid := 0; gid < writers; gid++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			f, err := appendFile(path)
			if err != nil {
				t.Errorf("appendFile() error = %v", err)
				return
			}
			defer f.Close()
			l := New(Config{Enabled: true, Level: slog.LevelInfo, Format: "json"}, f)
			for seq := 0; seq < recordsPerWriter; seq++ {
				l.Info("concurrent.write", "gid", gid, "seq", seq)
			}
		}(gid)
	}
	wg.Wait()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	type key struct{ gid, seq int }
	want := make(map[key]bool, writers*recordsPerWriter)
	for gid := 0; gid < writers; gid++ {
		for seq := 0; seq < recordsPerWriter; seq++ {
			want[key{gid, seq}] = true
		}
	}

	scanner := bufio.NewScanner(bytes.NewReader(raw))
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		lineCount++
		var decoded struct {
			GID int `json:"gid"`
			Seq int `json:"seq"`
		}
		if jsonErr := json.Unmarshal(line, &decoded); jsonErr != nil {
			t.Fatalf("line %d did not parse as valid JSON (interleaved or torn): %q: %v", lineCount, line, jsonErr)
		}
		k := key{decoded.GID, decoded.Seq}
		if !want[k] {
			t.Fatalf("line %d decoded to (gid=%d, seq=%d), which is not an outstanding expected record (duplicate or corrupted)", lineCount, decoded.GID, decoded.Seq)
		}
		delete(want, k)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		t.Fatalf("scanning file: %v", scanErr)
	}
	if lineCount != writers*recordsPerWriter {
		t.Fatalf("got %d lines, want %d", lineCount, writers*recordsPerWriter)
	}
	if len(want) != 0 {
		t.Fatalf("%d expected records never appeared: %v", len(want), want)
	}
}
