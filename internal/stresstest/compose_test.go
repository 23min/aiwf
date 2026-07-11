package stresstest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCompose_ParsesWellFormedEvents(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.jsonl")
	content := `{"scenario":"a","result":"pass"}` + "\n" + `{"scenario":"b","result":"fail"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	result, err := Compose(path)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if result.Truncated {
		t.Fatal("expected Truncated to be false for a well-formed file")
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result.Events))
	}
	var first map[string]string
	if err := json.Unmarshal(result.Events[0], &first); err != nil {
		t.Fatalf("decode first event: %v", err)
	}
	if first["scenario"] != "a" || first["result"] != "pass" {
		t.Fatalf("unexpected first event: %+v", first)
	}
}

func TestCompose_DropsTruncatedTrailingLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.jsonl")
	// Simulates a kill -9 mid-write: one complete record, then a
	// second record cut off partway through — no closing brace, no
	// trailing newline.
	content := `{"scenario":"a","result":"pass"}` + "\n" + `{"scenario":"b","resu`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	result, err := Compose(path)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if !result.Truncated {
		t.Fatal("expected Truncated to be true for a file ending in a malformed line")
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 complete event before the truncation, got %d", len(result.Events))
	}
}

func TestCompose_EmptyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("seed empty file: %v", err)
	}

	result, err := Compose(path)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if result.Truncated {
		t.Fatal("expected Truncated to be false for an empty file")
	}
	if len(result.Events) != 0 {
		t.Fatalf("expected 0 events for an empty file, got %d", len(result.Events))
	}
}

func TestCompose_MalformedMiddleLineErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.jsonl")
	content := `not-json-at-all` + "\n" + `{"scenario":"b","result":"pass"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	if _, err := Compose(path); err == nil {
		t.Fatal("expected Compose to error on a malformed non-trailing line")
	}
}

func TestCompose_ErrorsWhenFileMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if _, err := Compose(filepath.Join(dir, "does-not-exist.jsonl")); err == nil {
		t.Fatal("expected Compose to error when the raw-report file doesn't exist")
	}
}
