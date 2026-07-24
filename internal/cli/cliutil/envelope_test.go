package cliutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/version"
)

// The envelope constructors are the single source of truth for the
// tool/version identity pair every --format=json envelope carries, so
// these tests pin that pair (sourced from version.Current(), untagged
// under `go test` but still the one value the constructor must use) plus
// each helper's status logic and verbatim payload/metadata passthrough.

func TestNewEnvelope_FillsToolAndVersion(t *testing.T) {
	t.Parallel()
	env := newEnvelope()
	if env.Tool != "aiwf" {
		t.Errorf("Tool = %q, want aiwf", env.Tool)
	}
	if env.Version != version.Current().Version {
		t.Errorf("Version = %q, want %q", env.Version, version.Current().Version)
	}
	// The base carries nothing else — callers set status and payload.
	if env.Status != "" || env.Result != nil || env.Findings != nil || env.Error != nil || env.Metadata != nil {
		t.Errorf("base envelope carries more than tool/version: %+v", env)
	}
}

func TestOKEnvelope(t *testing.T) {
	t.Parallel()
	result := map[string]any{"rows": 3}
	meta := map[string]any{"root": "/repo", "count": 3}
	env := OKEnvelope(result, meta)

	if env.Tool != "aiwf" || env.Version != version.Current().Version {
		t.Errorf("identity pair not filled: tool=%q version=%q", env.Tool, env.Version)
	}
	if env.Status != "ok" {
		t.Errorf("Status = %q, want ok", env.Status)
	}
	if diff := cmp.Diff(result, env.Result); diff != "" {
		t.Errorf("Result passthrough mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(meta, env.Metadata); diff != "" {
		t.Errorf("Metadata passthrough mismatch (-want +got):\n%s", diff)
	}
	if env.Findings != nil || env.Error != nil {
		t.Errorf("OK envelope should carry no findings/error: %+v", env)
	}
}

func TestOKEnvelope_NilMetadataAndResult(t *testing.T) {
	t.Parallel()
	// schema/template pass a nil metadata; the filtered-out show path
	// passes a nil result. Both must round-trip as nil (the Envelope's
	// omitempty drops them from the JSON), not an empty non-nil map.
	env := OKEnvelope(nil, nil)
	if env.Status != "ok" {
		t.Errorf("Status = %q, want ok", env.Status)
	}
	if env.Result != nil {
		t.Errorf("Result = %v, want nil", env.Result)
	}
	if env.Metadata != nil {
		t.Errorf("Metadata = %v, want nil", env.Metadata)
	}
}

func TestFindingsEnvelope_StatusTracksFindings(t *testing.T) {
	t.Parallel()
	meta := map[string]any{"root": "/repo"}

	empty := FindingsEnvelope(nil, meta)
	if empty.Status != "ok" {
		t.Errorf("empty findings: Status = %q, want ok", empty.Status)
	}

	fs := []check.Finding{{Code: "x", Severity: check.SeverityError}}
	nonEmpty := FindingsEnvelope(fs, meta)
	if nonEmpty.Status != "findings" {
		t.Errorf("non-empty findings: Status = %q, want findings", nonEmpty.Status)
	}
	if diff := cmp.Diff(fs, nonEmpty.Findings); diff != "" {
		t.Errorf("Findings passthrough mismatch (-want +got):\n%s", diff)
	}
	if nonEmpty.Tool != "aiwf" || nonEmpty.Version != version.Current().Version {
		t.Errorf("identity pair not filled")
	}
	if nonEmpty.Result != nil {
		t.Errorf("findings envelope should carry no result: %+v", nonEmpty.Result)
	}
}

func TestErrorEnvelope(t *testing.T) {
	t.Parallel()
	meta := map[string]any{"root": "/repo", "id": "M-0001"}
	env := ErrorEnvelope("some-code", "boom", meta)

	if env.Status != "error" {
		t.Errorf("Status = %q, want error", env.Status)
	}
	if env.Error == nil {
		t.Fatalf("Error is nil, want populated")
	}
	if env.Error.Code != "some-code" || env.Error.Message != "boom" {
		t.Errorf("Error = %+v, want {some-code boom}", env.Error)
	}
	if diff := cmp.Diff(meta, env.Metadata); diff != "" {
		t.Errorf("Metadata passthrough mismatch (-want +got):\n%s", diff)
	}
	if env.Tool != "aiwf" || env.Version != version.Current().Version {
		t.Errorf("identity pair not filled")
	}
}

func TestErrorEnvelope_EmptyCodeOmitted(t *testing.T) {
	t.Parallel()
	// show's not-found path passes an empty code; the Envelope's omitempty
	// drops it from JSON, but the constructor still sets it to "".
	env := ErrorEnvelope("", "not found", nil)
	if env.Error == nil || env.Error.Code != "" || env.Error.Message != "not found" {
		t.Errorf("Error = %+v, want {\"\" not found}", env.Error)
	}
}
