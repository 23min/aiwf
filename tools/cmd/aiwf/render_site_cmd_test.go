package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun_RenderHTML_DispatchesToSite: `aiwf render --format=html`
// runs the htmlrender package and emits the JSON envelope on stdout
// with out_dir / files_written / elapsed_ms set. The seam test for
// step 4 — without it, a future renaming or refactor in either the
// dispatcher OR the htmlrender package can drift unnoticed.
func TestRun_RenderHTML_DispatchesToSite(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "Schema", "--actor", "human/test", "--root", root)

	out := filepath.Join(t.TempDir(), "site")
	captured := captureStdout(t, func() {
		mustRun(t, "render", "--root", root, "--format", "html", "--out", out)
	})

	var env struct {
		Tool   string `json:"tool"`
		Status string `json:"status"`
		Result struct {
			OutDir       string `json:"out_dir"`
			FilesWritten int    `json:"files_written"`
			ElapsedMs    int64  `json:"elapsed_ms"`
		} `json:"result"`
	}
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parse JSON envelope: %v\n%s", err, captured)
	}
	if env.Tool != "aiwf" || env.Status != "ok" {
		t.Errorf("envelope tool/status = %q/%q", env.Tool, env.Status)
	}
	if env.Result.OutDir != out {
		t.Errorf("result.out_dir = %q, want %q", env.Result.OutDir, out)
	}
	// 1 index + 1 status + 1 epic + 1 milestone.
	if env.Result.FilesWritten != 4 {
		t.Errorf("result.files_written = %d, want 4", env.Result.FilesWritten)
	}

	for _, name := range []string{"index.html", "status.html", "E-01.html", "M-001.html", "assets/style.css"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("expected %s in out dir; %v", name, err)
		}
	}
}

// TestRun_RenderHTML_HonorsAiwfYAMLOutDir: when --out is omitted and
// aiwf.yaml.html.out_dir is set, the renderer writes to the YAML-
// configured path. This is the consumer's expressed-intent path —
// most users will set the field once and run `aiwf render
// --format=html` without a flag override.
func TestRun_RenderHTML_HonorsAiwfYAMLOutDir(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)

	yamlPath := filepath.Join(root, "aiwf.yaml")
	raw, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read aiwf.yaml: %v", err)
	}
	patched := string(raw) + "html:\n  out_dir: docs/site\n"
	if err := os.WriteFile(yamlPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write aiwf.yaml: %v", err)
	}

	captured := captureStdout(t, func() {
		mustRun(t, "render", "--root", root, "--format", "html")
	})
	var env struct {
		Result struct {
			OutDir string `json:"out_dir"`
		} `json:"result"`
	}
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	wantSuffix := filepath.Join("docs", "site")
	if !strings.HasSuffix(env.Result.OutDir, wantSuffix) {
		t.Errorf("out_dir = %q, want suffix %q", env.Result.OutDir, wantSuffix)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "site", "index.html")); err != nil {
		t.Errorf("index.html missing at YAML-configured path; %v", err)
	}
}

// TestRun_RenderHTML_DeterministicAcrossInvocations: two back-to-back
// renders produce byte-identical output for every emitted file.
// Pins I3 plan §8 "Determinism" through the dispatcher seam (the
// htmlrender package has its own determinism test; this one
// asserts the property holds when invoked through `aiwf render`).
func TestRun_RenderHTML_DeterministicAcrossInvocations(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-01", "--title", "M", "--actor", "human/test", "--root", root)

	out1 := filepath.Join(t.TempDir(), "s1")
	out2 := filepath.Join(t.TempDir(), "s2")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out1)
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out2)

	for _, rel := range []string{"index.html", "E-01.html", "M-001.html", "assets/style.css"} {
		a := readFileT(t, filepath.Join(out1, rel))
		b := readFileT(t, filepath.Join(out2, rel))
		if a != b {
			t.Errorf("non-deterministic dispatcher output for %s", rel)
		}
	}
}

// TestRun_Render_DispatcherDistinguishesSubcommandFromFormat: the
// `roadmap` subcommand still works; --format=html without a
// subcommand routes to runRenderSite; an unknown subcommand without
// --format is a usage error.
func TestRun_Render_DispatcherDistinguishesSubcommandFromFormat(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test")
	mustRun(t, "add", "epic", "--title", "F", "--actor", "human/test", "--root", root)

	// roadmap subcommand still emits markdown on stdout.
	captured := captureStdout(t, func() {
		mustRun(t, "render", "roadmap", "--root", root)
	})
	if !strings.Contains(string(captured), "# Roadmap") {
		t.Errorf("render roadmap stdout missing # Roadmap header:\n%s", captured)
	}

	// --format=html without subcommand → site render.
	out := filepath.Join(t.TempDir(), "site")
	mustRun(t, "render", "--root", root, "--format", "html", "--out", out)
	if _, err := os.Stat(filepath.Join(out, "index.html")); err != nil {
		t.Errorf("expected index.html after --format=html dispatch; %v", err)
	}

	// Unknown subcommand without --format → usage error.
	if rc := run([]string{"render", "nope", "--root", root}); rc == exitOK {
		t.Errorf("render nope should be a usage error; got exitOK")
	}
}

// TestRun_Render_HelpFlag: `aiwf render --help` prints the verb's
// usage and exits cleanly. Pre-fix the dispatcher fell into the
// subcommand switch and reported "unknown subcommand --help" with
// exitUsage. Both surfaces (roadmap + --format=html) must appear
// so the help text is a true catalog.
func TestRun_Render_HelpFlag(t *testing.T) {
	cases := []struct {
		name string
		arg  string
	}{
		{"--help", "--help"},
		{"-h", "-h"},
		{"help", "help"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rc int
			captured := captureStdout(t, func() {
				rc = run([]string{"render", tc.arg})
			})
			if rc != exitOK {
				t.Errorf("rc = %d, want exitOK", rc)
			}
			out := string(captured)
			for _, want := range []string{"roadmap", "--format=html"} {
				if !strings.Contains(out, want) {
					t.Errorf("help output missing %q:\n%s", want, out)
				}
			}
		})
	}
}

// readFileT mirrors readFile from htmlrender_test but is package-
// local for cmd/aiwf tests.
func readFileT(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
