package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// envelopeShape exercises the canonical fields every envelope must include.
// This locks in the shape from day one; later verbs add to `result`/`findings`
// but cannot drop these keys.
func TestEnvelopeShape_Help(t *testing.T) {
	got := helpEnvelope()

	want := envelope{
		Tool:    "aiwf",
		Version: "dev",
		Status:  "ok",
	}

	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(envelope{}, "Result")); diff != "" {
		t.Errorf("help envelope shape mismatch (-want +got):\n%s", diff)
	}

	if got.Result == nil {
		t.Error("help envelope must carry a Result describing the binary")
	}
}

func TestEnvelopeShape_Version(t *testing.T) {
	got := versionEnvelope()

	if got.Tool != "aiwf" {
		t.Errorf("version envelope: Tool = %q, want aiwf", got.Tool)
	}
	if got.Status != "ok" {
		t.Errorf("version envelope: Status = %q, want ok", got.Status)
	}
	if got.Version == "" {
		t.Error("version envelope: Version is empty")
	}
}

func TestNotImplemented_CarriesVerbName(t *testing.T) {
	tests := []struct {
		name string
		verb string
	}{
		{"add", "add"},
		{"promote", "promote"},
		{"verify", "verify"},
		{"unknown verb", "made-up-verb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := notImplementedEnvelope(tt.verb)

			if env.Status != "findings" {
				t.Errorf("Status = %q, want findings", env.Status)
			}
			if len(env.Findings) != 1 {
				t.Fatalf("len(Findings) = %d, want 1", len(env.Findings))
			}
			f := env.Findings[0]
			if f.Code != "NOT_YET_IMPLEMENTED" {
				t.Errorf("Findings[0].Code = %q, want NOT_YET_IMPLEMENTED", f.Code)
			}
			if !strings.Contains(f.Message, tt.verb) {
				t.Errorf("Findings[0].Message = %q, want it to contain verb %q", f.Message, tt.verb)
			}
			if got, ok := f.Context["verb"].(string); !ok || got != tt.verb {
				t.Errorf("Findings[0].Context[verb] = %v, want %q", f.Context["verb"], tt.verb)
			}
		})
	}
}

// TestEnvelopeRoundTrip confirms the envelope serializes deterministically
// and decodes back equivalent. This is the foundation of the JSON envelope
// contract; any future verb's result must remain encodable as `any`.
func TestEnvelopeRoundTrip(t *testing.T) {
	original := envelope{
		Tool:    "aiwf",
		Version: "0.1.0",
		Status:  "findings",
		Findings: []finding{{
			Code:     "TEST",
			Severity: "low",
			Message:  "round-trip test",
			Context:  map[string]any{"key": "value"},
		}},
		Result:   map[string]any{"count": float64(3)}, // float64 because JSON numbers decode that way
		Metadata: map[string]any{"correlation_id": "abc-123"},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded envelope
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if diff := cmp.Diff(original, decoded); diff != "" {
		t.Errorf("round-trip mismatch (-original +decoded):\n%s", diff)
	}
}
