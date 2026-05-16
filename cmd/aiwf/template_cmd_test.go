package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/entity"
)

func TestRunTemplate_OneKindRaw(t *testing.T) {
	out := string(captureStdout(t, func() {
		if rc := run([]string{"template", "epic"}); rc != cliutil.ExitOK {
			t.Fatalf("rc = %d", rc)
		}
	}))
	want := string(entity.BodyTemplate(entity.KindEpic))
	if out != want {
		t.Errorf("single-kind output should be raw template body\ngot:  %q\nwant: %q", out, want)
	}
	if strings.Contains(out, "KIND:") {
		t.Errorf("single-kind output should not have KIND: header; got:\n%s", out)
	}
}

func TestRunTemplate_AllKindsHasHeaders(t *testing.T) {
	out := string(captureStdout(t, func() {
		if rc := run([]string{"template"}); rc != cliutil.ExitOK {
			t.Fatalf("rc = %d", rc)
		}
	}))
	for _, k := range entity.AllKinds() {
		if !strings.Contains(out, "KIND: "+string(k)) {
			t.Errorf("missing KIND: %s\nfull output:\n%s", k, out)
		}
		// Spot-check that a body line from each kind is present.
		body := string(entity.BodyTemplate(k))
		bodyLines := strings.Split(strings.TrimSpace(body), "\n")
		// Find a non-empty section header line and assert it appears.
		for _, line := range bodyLines {
			if strings.HasPrefix(line, "## ") {
				if !strings.Contains(out, line) {
					t.Errorf("missing body line %q for kind %s", line, k)
				}
				break
			}
		}
	}
}

func TestRunTemplate_UnknownKind(t *testing.T) {
	t.Parallel()
	if rc := run([]string{"template", "nonsense"}); rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want %d", rc, cliutil.ExitUsage)
	}
}

func TestRunTemplate_TooManyArgs(t *testing.T) {
	t.Parallel()
	if rc := run([]string{"template", "epic", "milestone"}); rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want %d", rc, cliutil.ExitUsage)
	}
}

func TestRunTemplate_BadFormat(t *testing.T) {
	t.Parallel()
	if rc := run([]string{"template", "--format", "yaml"}); rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want %d", rc, cliutil.ExitUsage)
	}
}

func TestRunTemplate_PrettyWithoutJSONIsHarmless(t *testing.T) {
	t.Parallel()
	if rc := run([]string{"template", "--pretty", "epic"}); rc != cliutil.ExitOK {
		t.Errorf("rc = %d, want %d", rc, cliutil.ExitOK)
	}
}

func TestRunTemplate_JSONEnvelope(t *testing.T) {
	out := captureStdout(t, func() {
		if rc := run([]string{"template", "--format", "json"}); rc != cliutil.ExitOK {
			t.Fatalf("rc = %d", rc)
		}
	})
	var env struct {
		Tool    string `json:"tool"`
		Version string `json:"version"`
		Status  string `json:"status"`
		Result  struct {
			Templates []templateOut `json:"templates"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, string(out))
	}
	if env.Tool != "aiwf" {
		t.Errorf("Tool = %q", env.Tool)
	}
	if env.Status != "ok" {
		t.Errorf("Status = %q", env.Status)
	}
	if len(env.Result.Templates) != len(entity.AllKinds()) {
		t.Errorf("Templates length = %d, want %d", len(env.Result.Templates), len(entity.AllKinds()))
	}
	for i, k := range entity.AllKinds() {
		if env.Result.Templates[i].Kind != k {
			t.Errorf("Templates[%d].Kind = %q, want %q", i, env.Result.Templates[i].Kind, k)
		}
		if env.Result.Templates[i].Body != string(entity.BodyTemplate(k)) {
			t.Errorf("Templates[%d].Body mismatch for kind %s", i, k)
		}
	}
}

func TestRunTemplate_JSONOneKind(t *testing.T) {
	out := captureStdout(t, func() {
		if rc := run([]string{"template", "--format", "json", "epic"}); rc != cliutil.ExitOK {
			t.Fatalf("rc = %d", rc)
		}
	})
	var env struct {
		Result struct {
			Templates []templateOut `json:"templates"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("unmarshal: %v\noutput: %s", err, string(out))
	}
	if len(env.Result.Templates) != 1 || env.Result.Templates[0].Kind != entity.KindEpic {
		t.Errorf("expected single epic template; got %+v", env.Result.Templates)
	}
}

func TestWriteTemplateText_WriterError(t *testing.T) {
	t.Parallel()
	got := writeTemplateText(brokenWriter{}, []templateOut{{Kind: entity.KindEpic, Body: "x"}}, true)
	if got == nil {
		t.Error("expected error from broken writer")
	}
}
