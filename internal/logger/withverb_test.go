package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// loggedFields binds verb/entity/actor/run_id via WithVerb, emits one
// record, and decodes the resulting JSON line so tests can assert on
// the actual bound field values a real log line would carry. runID is
// fixed ("run-test") rather than logger.NewRunID() throughout this
// file: these tests pin WithVerb's own binding behavior, not id
// generation (that's runid_test.go's job).
const testRunID = "run-test"

func loggedFields(t *testing.T, verb, entity, actor string) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	base := New(Config{Enabled: true, Level: slog.LevelInfo, Format: "json"}, &buf)
	bound := WithVerb(base, verb, entity, actor, testRunID)
	bound.Info("event")

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output %q did not parse as JSON: %v", buf.String(), err)
	}
	return decoded
}

func TestWithVerb_NoLeak_PassesThroughUnchanged(t *testing.T) {
	t.Parallel()
	fields := loggedFields(t, "promote", "M-0090", "human/peter")
	if fields["verb"] != "promote" || fields["entity"] != "M-0090" || fields["actor"] != "human/peter" {
		t.Fatalf("fields = %+v, want verb/entity/actor to pass through unchanged", fields)
	}
}

func TestWithVerb_BindsRunID(t *testing.T) {
	t.Parallel()
	fields := loggedFields(t, "promote", "M-0090", "human/peter")
	if fields["run_id"] != testRunID {
		t.Fatalf("fields[run_id] = %v, want %q", fields["run_id"], testRunID)
	}
}

// Fixture home paths below use "x" as the placeholder username — the
// same synthetic fixture user .gitleaks.toml's test-placeholder
// allowlist already codifies, so these path-leak-shaped test inputs
// (deliberately exercising the scrubber) don't themselves trip the
// path-leak lint.

func TestWithVerb_ScrubsMacOSHomePath(t *testing.T) {
	t.Parallel()
	fields := loggedFields(t, "promote", "/Users/x/repos/aiwf/work/M-0090.md", "human/peter")
	got, ok := fields["entity"].(string)
	if !ok {
		t.Fatalf("fields[entity] = %v, want a string", fields["entity"])
	}
	if strings.Contains(got, "/Users/x/") {
		t.Fatalf("entity = %q, still leaks the operator's home path", got)
	}
	if !strings.HasSuffix(got, "repos/aiwf/work/M-0090.md") {
		t.Fatalf("entity = %q, want the path remainder preserved after scrubbing", got)
	}
}

func TestWithVerb_ScrubsLinuxHomePath(t *testing.T) {
	t.Parallel()
	fields := loggedFields(t, "promote", "M-0090", "/home/x/aiwf/.git/hooks/pre-push")
	got, ok := fields["actor"].(string)
	if !ok {
		t.Fatalf("fields[actor] = %v, want a string", fields["actor"])
	}
	if strings.Contains(got, "/home/x/") {
		t.Fatalf("actor = %q, still leaks the home path", got)
	}
}

func TestWithVerb_ScrubsAllThreeFields(t *testing.T) {
	t.Parallel()
	fields := loggedFields(t,
		"/Users/x/bin/aiwf",
		"/home/x/work/G-0001.md",
		"/Users/x/.gitconfig",
	)
	for _, key := range []string{"verb", "entity", "actor"} {
		got, ok := fields[key].(string)
		if !ok {
			t.Fatalf("fields[%s] = %v, want a string", key, fields[key])
		}
		if strings.Contains(got, "/Users/x/") || strings.Contains(got, "/home/x/") {
			t.Fatalf("%s = %q, still leaks the operator's home path", key, got)
		}
	}
}

func TestWithVerb_ScrubsMultipleFragmentsInOneValue(t *testing.T) {
	t.Parallel()
	fields := loggedFields(t, "promote", "M-0090", "/Users/x/a to /home/x/b")
	got, ok := fields["actor"].(string)
	if !ok {
		t.Fatalf("fields[actor] = %v, want a string", fields["actor"])
	}
	if strings.Contains(got, "/Users/x/") || strings.Contains(got, "/home/x/") {
		t.Fatalf("actor = %q, still leaks the home path across multiple fragments", got)
	}
}

func TestScrubHomePaths(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty string", "", ""},
		{"no leak", "human/peter", "human/peter"},
		{"macOS home", "/Users/x/repo", "<redacted-home>/repo"},
		{"linux home", "/home/x/repo", "<redacted-home>/repo"},
		{"both in one value", "/Users/x/p and /home/x/q", "<redacted-home>/p and <redacted-home>/q"},
		{"username-shaped substring without slashes is not a path", "Users/x", "Users/x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := scrubHomePaths(tc.in); got != tc.want {
				t.Fatalf("scrubHomePaths(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
