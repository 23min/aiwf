package stresstest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// ComposeResult is the outcome of composing a raw-report JSONL file:
// every successfully-parsed event, in order, plus whether the file's
// final line was dropped as truncated.
type ComposeResult struct {
	Events    []json.RawMessage
	Truncated bool
}

// Compose reads path (a raw-report JSONL file written by
// ReportWriter) and returns every well-formed event line. A malformed
// trailing line — the shape a kill -9 mid-write leaves behind, since
// ReportWriter's O_APPEND + one-Write()-per-record discipline (AC-2)
// guarantees only the last record can ever be partial — is dropped
// silently and reported via Truncated, never treated as a
// whole-report failure: the same "errors are findings, not parse
// failures" posture aiwf check holds toward the entity tree. A
// malformed line anywhere else in the file is a different kind of
// corruption than an abort and surfaces as an error.
func Compose(path string) (ComposeResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ComposeResult{}, fmt.Errorf("reading raw-report file %s: %w", path, err)
	}
	lines := splitNonEmptyLines(data)

	var result ComposeResult
	for i, line := range lines {
		var raw json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			if i == len(lines)-1 {
				result.Truncated = true
				break
			}
			return ComposeResult{}, fmt.Errorf("raw-report file %s: malformed record on line %d: %w", path, i+1, err)
		}
		result.Events = append(result.Events, raw)
	}
	return result, nil
}

// splitNonEmptyLines splits data on '\n' and drops empty entries — a
// well-formed JSONL file ends with a newline, which would otherwise
// produce one spurious empty trailing element.
func splitNonEmptyLines(data []byte) [][]byte {
	raw := bytes.Split(data, []byte("\n"))
	var out [][]byte
	for _, line := range raw {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		out = append(out, line)
	}
	return out
}
