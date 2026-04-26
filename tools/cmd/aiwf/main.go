// Command aiwf is the ai-workflow framework's single binary.
//
// Stage 2 lands the kernel: events.jsonl, projection, verify. Verb
// implementations land in subsequent PRs; this stub binary recognizes
// the verb names so test scaffolding can target them, but returns a
// "not yet implemented" finding for any verb that hasn't shipped.
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Version is the binary's reported version. Set via -ldflags at build time
// once releases start shipping; defaults to "dev" otherwise.
var Version = "dev"

// envelope is the JSON shape every aiwf invocation writes to stdout.
// The full envelope schema is described in docs/architecture.md §5.5.
type envelope struct {
	Tool     string         `json:"tool"`
	Version  string         `json:"version"`
	Status   string         `json:"status"`
	Findings []finding      `json:"findings"`
	Result   any            `json:"result,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// finding is a single structured report. Real verbs will emit richer
// findings; the stub uses this shape so the envelope is correct from day one.
type finding struct {
	Code     string         `json:"code"`
	Severity string         `json:"severity"`
	Message  string         `json:"message"`
	Context  map[string]any `json:"context,omitempty"`
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		emit(usageEnvelope())
		os.Exit(2)
	}

	switch args[0] {
	case "--help", "-h", "help":
		emit(helpEnvelope())
		os.Exit(0)
	case "--version", "-v", "version":
		emit(versionEnvelope())
		os.Exit(0)
	default:
		emit(notImplementedEnvelope(args[0]))
		os.Exit(1)
	}
}

func emit(env envelope) {
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(env); err != nil {
		// If JSON encoding fails, fall back to a minimal stderr line.
		// Library code never panics; main can write diagnostics directly.
		fmt.Fprintf(os.Stderr, "aiwf: failed to encode envelope: %v\n", err)
		os.Exit(3)
	}
}

func usageEnvelope() envelope {
	return envelope{
		Tool:    "aiwf",
		Version: Version,
		Status:  "error",
		Findings: []finding{{
			Code:     "USAGE",
			Severity: "high",
			Message:  "missing subcommand. Try 'aiwf --help'.",
		}},
	}
}

func helpEnvelope() envelope {
	return envelope{
		Tool:    "aiwf",
		Version: Version,
		Status:  "ok",
		Result: map[string]any{
			"description": "ai-workflow CLI. Stage 2 (kernel) is in flight; most verbs land in subsequent PRs.",
			"verbs": map[string]string{
				"--help":    "show this message",
				"--version": "show binary version",
			},
			"docs": "https://github.com/23min/ai-workflow-v2/blob/main/docs/architecture.md",
		},
	}
}

func versionEnvelope() envelope {
	return envelope{
		Tool:    "aiwf",
		Version: Version,
		Status:  "ok",
		Result:  map[string]any{"version": Version},
	}
}

func notImplementedEnvelope(verb string) envelope {
	return envelope{
		Tool:    "aiwf",
		Version: Version,
		Status:  "findings",
		Findings: []finding{{
			Code:     "NOT_YET_IMPLEMENTED",
			Severity: "high",
			Message:  fmt.Sprintf("verb %q not yet implemented in this build", verb),
			Context: map[string]any{
				"verb":          verb,
				"current_stage": "Stage 2 (kernel)",
				"reference":     "https://github.com/23min/ai-workflow-v2/blob/main/docs/build-plan.md",
			},
		}},
	}
}
