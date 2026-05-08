package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/entity"
)

var errBroken = errors.New("broken writer")

func TestRunSchema_AllKindsText(t *testing.T) {
	out := string(captureStdout(t, func() {
		if rc := run([]string{"schema"}); rc != exitOK {
			t.Fatalf("rc = %d", rc)
		}
	}))
	for _, k := range entity.AllKinds() {
		if !strings.Contains(out, "KIND: "+string(k)) {
			t.Errorf("output missing KIND: %s\nfull output:\n%s", k, out)
		}
	}
}

func TestRunSchema_OneKindText(t *testing.T) {
	out := string(captureStdout(t, func() {
		if rc := run([]string{"schema", "milestone"}); rc != exitOK {
			t.Fatalf("rc = %d", rc)
		}
	}))
	if !strings.Contains(out, "KIND: milestone") {
		t.Errorf("missing milestone header in output:\n%s", out)
	}
	if strings.Contains(out, "KIND: epic") {
		t.Errorf("epic should not appear when one kind is requested:\n%s", out)
	}
	for _, want := range []string{"id format:", "allowed statuses:", "required fields:", "reference fields:", "parent", "depends_on"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRunSchema_UnknownKind(t *testing.T) {
	if rc := run([]string{"schema", "nonsense"}); rc != exitUsage {
		t.Errorf("rc = %d, want %d", rc, exitUsage)
	}
}

func TestRunSchema_TooManyArgs(t *testing.T) {
	if rc := run([]string{"schema", "epic", "milestone"}); rc != exitUsage {
		t.Errorf("rc = %d, want %d", rc, exitUsage)
	}
}

func TestRunSchema_BadFormat(t *testing.T) {
	if rc := run([]string{"schema", "--format", "yaml"}); rc != exitUsage {
		t.Errorf("rc = %d, want %d", rc, exitUsage)
	}
}

func TestRunSchema_JSONEnvelope(t *testing.T) {
	out := captureStdout(t, func() {
		if rc := run([]string{"schema", "--format", "json"}); rc != exitOK {
			t.Fatalf("rc = %d", rc)
		}
	})
	var env struct {
		Tool    string `json:"tool"`
		Version string `json:"version"`
		Status  string `json:"status"`
		Result  struct {
			Schemas []entity.Schema `json:"schemas"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, string(out))
	}
	if env.Tool != "aiwf" {
		t.Errorf("Tool = %q, want %q", env.Tool, "aiwf")
	}
	if env.Status != "ok" {
		t.Errorf("Status = %q, want ok", env.Status)
	}
	if len(env.Result.Schemas) != len(entity.AllKinds()) {
		t.Errorf("Schemas length = %d, want %d", len(env.Result.Schemas), len(entity.AllKinds()))
	}
	for i, k := range entity.AllKinds() {
		if env.Result.Schemas[i].Kind != k {
			t.Errorf("Schemas[%d].Kind = %q, want %q", i, env.Result.Schemas[i].Kind, k)
		}
	}
}

func TestRunSchema_PrettyWithoutJSONIsHarmless(t *testing.T) {
	// --pretty without --format=json prints a stderr nudge but still
	// exits 0 with text output.
	if rc := run([]string{"schema", "--pretty", "epic"}); rc != exitOK {
		t.Errorf("rc = %d, want %d", rc, exitOK)
	}
}

func TestWriteSchemaText_WriterError(t *testing.T) {
	// Confirms the error-return path on an io.Writer that fails on the
	// first byte — covers the defensive `if _, err := ...; err != nil`
	// branches that stdout in normal tests can never reach.
	got := writeSchemaText(brokenWriter{}, []entity.Schema{{Kind: entity.KindEpic, IDFormat: "E-NN"}})
	if got == nil {
		t.Error("expected error from broken writer")
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errBroken }

func TestRunSchema_JSONOneKind(t *testing.T) {
	out := captureStdout(t, func() {
		if rc := run([]string{"schema", "--format", "json", "epic"}); rc != exitOK {
			t.Fatalf("rc = %d", rc)
		}
	})
	var env struct {
		Result struct {
			Schemas []entity.Schema `json:"schemas"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, string(out))
	}
	if len(env.Result.Schemas) != 1 || env.Result.Schemas[0].Kind != entity.KindEpic {
		t.Errorf("expected single epic schema; got %+v", env.Result.Schemas)
	}
}
