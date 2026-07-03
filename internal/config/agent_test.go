package config

import (
	"strings"
	"testing"
)

// TestValidateAgents pins the closed-set vocabulary for the per-agent
// model/effort knobs (G-0353): a set value must be in the vocabulary, an
// omitted field is always legal, and the offending map key is named.
func TestValidateAgents(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		agents  map[string]Agent
		wantErr string // substring; "" = expect no error
	}{
		{"nil map", nil, ""},
		{"model and effort valid", map[string]Agent{"reviewer": {Model: "sonnet", Effort: "high"}}, ""},
		{"model only", map[string]Agent{"builder": {Model: "haiku"}}, ""},
		{"effort only", map[string]Agent{"planner": {Effort: "xhigh"}}, ""},
		{"inherit is a valid model", map[string]Agent{"deployer": {Model: "inherit"}}, ""},
		{"fable is a valid model", map[string]Agent{"reviewer": {Model: "fable"}}, ""},
		{"max is a valid effort", map[string]Agent{"reviewer": {Effort: "max"}}, ""},
		{"both fields empty", map[string]Agent{"reviewer": {}}, ""},
		{"unknown model", map[string]Agent{"reviewer": {Model: "sonnet-5"}}, "agents.reviewer.model"},
		{"unknown effort", map[string]Agent{"builder": {Effort: "highest"}}, "agents.builder.effort"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := (&Config{Agents: tc.agents}).Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

// TestValidateAreasErrorShortCircuits confirms Validate returns the areas
// error before reaching agent validation (the new early-return branch).
func TestValidateAreasErrorShortCircuits(t *testing.T) {
	t.Parallel()
	// required:true with zero members is the canonical Areas.validate() error.
	err := (&Config{Areas: Areas{Required: true}}).Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want an areas error to propagate")
	}
}
