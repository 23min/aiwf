package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loggingLine returns the `logging:` line from doctor output, or fails.
// Scoping the assertion to that one line keeps it structural rather than
// a flat substring grep over the whole report.
func loggingLine(t *testing.T, lines []string) string {
	t.Helper()
	for _, l := range lines {
		if strings.HasPrefix(l, "logging:") {
			return l
		}
	}
	t.Fatalf("no `logging:` line in doctor output:\n%s", strings.Join(lines, "\n"))
	return ""
}

// TestDoctorReport_LoggingDisabledByDefault pins M-0238/AC-4: with
// neither AIWF_LOG set nor a logging: block in aiwf.yaml, doctor
// reports logging as disabled — never a problem (it's the documented
// default-off state, not a misconfiguration).
// t.Setenv blocks t.Parallel; this test is intentionally serial.
func TestDoctorReport_LoggingDisabledByDefault(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AIWF_LOG", "")
	lines, problems := DoctorReport(root, DoctorOptions{})
	got := loggingLine(t, lines)
	if !strings.Contains(got, "disabled") {
		t.Errorf("logging line = %q, want it to contain \"disabled\"", got)
	}
	for _, p := range problems {
		if strings.Contains(p.Message, "logging") {
			t.Errorf("disabled logging must not be a problem; got %+v", p)
		}
	}
}

// TestDoctorReport_LoggingEnabledViaEnv pins the env-sourced case: the
// resolved level/format/destination are surfaced, each annotated with
// the tier that supplied it.
// t.Setenv blocks t.Parallel; this test is intentionally serial.
func TestDoctorReport_LoggingEnabledViaEnv(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", "stderr")
	lines, _ := DoctorReport(root, DoctorOptions{})
	got := loggingLine(t, lines)
	for _, want := range []string{"level=info", "format=json", "destination=stderr", "level: env", "format: env", "destination: env"} {
		if !strings.Contains(got, want) {
			t.Errorf("logging line = %q, want it to contain %q", got, want)
		}
	}
}

// TestDoctorReport_LoggingEnabledViaYAML pins the yaml-sourced case,
// with env left unset so every field's source is "yaml".
// t.Setenv blocks t.Parallel; this test is intentionally serial.
func TestDoctorReport_LoggingEnabledViaYAML(t *testing.T) {
	root := t.TempDir()
	yaml := "logging:\n  level: debug\n  format: text\n  destination: stderr\n"
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AIWF_LOG", "")
	t.Setenv("AIWF_LOG_FORMAT", "")
	t.Setenv("AIWF_LOG_FILE", "")
	lines, _ := DoctorReport(root, DoctorOptions{})
	got := loggingLine(t, lines)
	for _, want := range []string{"level=debug", "format=text", "destination=stderr", "level: yaml", "format: yaml", "destination: yaml"} {
		if !strings.Contains(got, want) {
			t.Errorf("logging line = %q, want it to contain %q", got, want)
		}
	}
}

// TestDoctorReport_LoggingMissingAiwfYAML pins the nil-config
// tolerance: no aiwf.yaml at all (config.Load fails) behaves exactly
// like an aiwf.yaml with no logging: block for this line — env alone
// still resolves normally.
// t.Setenv blocks t.Parallel; this test is intentionally serial.
func TestDoctorReport_LoggingMissingAiwfYAML(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FILE", "stderr")
	lines, _ := DoctorReport(root, DoctorOptions{})
	got := loggingLine(t, lines)
	if !strings.Contains(got, "level=info") || !strings.Contains(got, "level: env") {
		t.Errorf("logging line = %q, want it to resolve from env despite no aiwf.yaml", got)
	}
}

// TestDoctorReport_LoggingDefaultDestination pins the destination
// display for the unset case: the empty string (use the default
// XDG-state-home file) reads as a labeled placeholder, not a blank.
// t.Setenv blocks t.Parallel; this test is intentionally serial.
func TestDoctorReport_LoggingDefaultDestination(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FILE", "")
	lines, _ := DoctorReport(root, DoctorOptions{})
	got := loggingLine(t, lines)
	if !strings.Contains(got, "destination=(default XDG-state-home file)") {
		t.Errorf("logging line = %q, want the default-destination placeholder", got)
	}
}

// TestDoctorReport_LoggingInvalidLevel_ReportsProblem pins the error
// path: an invalid AIWF_LOG value is a real misconfiguration, surfaced
// as a problem rather than silently falling back to disabled.
// t.Setenv blocks t.Parallel; this test is intentionally serial.
func TestDoctorReport_LoggingInvalidLevel_ReportsProblem(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "aiwf.yaml"), []byte("hosts: [claude-code]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AIWF_LOG", "not-a-level")
	lines, problems := DoctorReport(root, DoctorOptions{})
	got := loggingLine(t, lines)
	if !strings.Contains(got, "not-a-level") {
		t.Errorf("logging line = %q, want it to name the invalid value", got)
	}
	found := false
	for _, p := range problems {
		if p.Severity == SeverityError && strings.Contains(p.Message, "not-a-level") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an error-severity problem naming the invalid level; got %+v", problems)
	}
}
