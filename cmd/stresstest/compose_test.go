package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCompose_RendersEvents(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.jsonl")
	content := `{"attempt":0,"seed":1,"passed":true}` + "\n" + `{"attempt":1,"seed":2,"passed":false}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seed report file: %v", err)
	}

	var out bytes.Buffer
	if err := runCompose(path, &out); err != nil {
		t.Fatalf("runCompose: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "2 event(s)") {
		t.Fatalf("expected the event count in output, got %q", got)
	}
	if strings.Contains(got, "truncated") {
		t.Fatalf("did not expect a truncation note for a well-formed file, got %q", got)
	}
}

func TestRunCompose_NotesTruncation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.jsonl")
	content := `{"attempt":0,"seed":1,"passed":true}` + "\n" + `{"attempt":1,"broken`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seed report file: %v", err)
	}

	var out bytes.Buffer
	if err := runCompose(path, &out); err != nil {
		t.Fatalf("runCompose: %v", err)
	}
	if !strings.Contains(out.String(), "truncated") {
		t.Fatalf("expected a truncation note, got %q", out.String())
	}
}

func TestRunCompose_ZeroEvents(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("seed empty report file: %v", err)
	}

	var out bytes.Buffer
	if err := runCompose(path, &out); err != nil {
		t.Fatalf("runCompose: %v", err)
	}
	if !strings.Contains(out.String(), "0 event(s)") {
		t.Fatalf("expected the zero-event count in output, got %q", out.String())
	}
}

func TestRunCompose_ErrorsWhenFileMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var out bytes.Buffer
	if err := runCompose(filepath.Join(dir, "missing.jsonl"), &out); err == nil {
		t.Fatal("expected runCompose to error when the raw-report file doesn't exist")
	}
}
