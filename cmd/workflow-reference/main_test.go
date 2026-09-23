package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec"
)

func TestGeneratorWritesFreshReference(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "reference.md")
	if err := os.WriteFile(path, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if code := run(context.Background(), []string{"--out", path}); code != 0 {
			t.Fatalf("exit = %d", code)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, spec.RenderReference()) {
			t.Fatal("generator did not write fresh reference")
		}
	}
}

func TestGeneratorRefusesInvalidInvocation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{"extra argument", []string{"unexpected"}, 2},
		{"unknown flag", []string{"--unknown"}, 2},
		{"unwritable destination", []string{"--out", filepath.Join(t.TempDir(), "missing", "reference.md")}, 3},
		{"help", []string{"--help"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := run(context.Background(), tc.args); got != tc.code {
				t.Errorf("exit = %d, want %d", got, tc.code)
			}
		})
	}
}

func TestGeneratorHonorsCancellationWithoutWriting(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "reference.md")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code := run(ctx, []string{"--out", path}); code != 3 {
		t.Fatalf("exit = %d", code)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original" {
		t.Fatal("cancelled command changed destination")
	}
}
