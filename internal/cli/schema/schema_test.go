package schema_test

import (
	"errors"
	"testing"

	"github.com/23min/aiwf/internal/cli/schema"
	"github.com/23min/aiwf/internal/entity"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := schema.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "schema [kind]" {
		t.Errorf("Use = %q", cmd.Use)
	}
	for _, flag := range []string{"format", "pretty"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}

// errAllowedStatusesWrite is the sentinel failAfterNWriter returns
// once its allotted successful writes are exhausted.
var errAllowedStatusesWrite = errors.New("boom on allowed-statuses write")

// failAfterNWriter succeeds for its first n Write calls, then fails on
// every call after. Unlike a writer that fails on the very first byte
// (which only ever reaches WriteSchemaText's first error-return, at
// the "KIND:" line), this lets a test target a *specific* later
// Fprintf's own `if err != nil { return err }` branch — each is a
// distinct defensive branch the diff-scoped coverage audit tracks
// per line.
type failAfterNWriter struct{ n int }

func (w *failAfterNWriter) Write(p []byte) (int, error) {
	if w.n <= 0 {
		return 0, errAllowedStatusesWrite
	}
	w.n--
	return len(p), nil
}

// TestWriteSchemaText_AllowedStatusesWriteError targets the
// "allowed statuses: %s" Fprintf's error-return branch specifically
// (the third write WriteSchemaText makes per schema, after "KIND:"
// and "id format:" both succeed) — this is also the line that joins
// the now-typed []entity.Status AllowedStatuses via
// entity.StatusStrings for the boundary cast to plain strings.
func TestWriteSchemaText_AllowedStatusesWriteError(t *testing.T) {
	t.Parallel()
	s := entity.Schema{
		Kind:            entity.KindEpic,
		IDFormat:        "E-NN",
		AllowedStatuses: []entity.Status{entity.StatusActive, entity.StatusDone},
	}
	err := schema.WriteSchemaText(&failAfterNWriter{n: 2}, []entity.Schema{s})
	if !errors.Is(err, errAllowedStatusesWrite) {
		t.Fatalf("WriteSchemaText error = %v, want %v", err, errAllowedStatusesWrite)
	}
}
