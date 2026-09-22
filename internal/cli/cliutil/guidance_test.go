package cliutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/projectguidance"
	"github.com/23min/aiwf/internal/testsupport"
)

func guidancePromptFixture(t *testing.T) (string, *config.Config, projectguidance.Catalogue) {
	t.Helper()
	root := t.TempDir()
	if err := gitops.Init(t.Context(), root); err != nil {
		t.Fatal(err)
	}
	writeAiwfYAML(t, root, "# Keep my notes.\nhosts: []\ncustom: retained\nguidance:\n  source: local-source\n  wire_agentsmd: false\n  packs: [existing]\n  ignored: [ignored]\n")
	if writeErr := os.WriteFile(filepath.Join(root, "source.xyz"), []byte("source"), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var catalogue projectguidance.Catalogue
	for _, id := range []string{"existing", "ignored", "first", "second", "third"} {
		catalogue.Packs = append(catalogue.Packs, projectguidance.Pack{ID: id, Description: "Policy for " + id, Detect: []string{"*.xyz"}})
	}
	return root, cfg, catalogue
}

func TestGuidancePrompt_SavesCompletedChoicesAndSkipsDecidedPacks(t *testing.T) {
	t.Parallel()
	root, cfg, catalogue := guidancePromptFixture(t)
	var out bytes.Buffer
	prompt := GuidancePrompt{In: strings.NewReader("invalid\ns\ni\n\n"), Out: &out}
	if err := prompt.Select(t.Context(), root, catalogue, cfg); err != nil {
		t.Fatal(err)
	}
	reloaded, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]string{"existing", "first"}, *reloaded.Guidance.Packs); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff([]string{"ignored", "second"}, reloaded.Guidance.Ignored); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff(reloaded.Guidance, cfg.Guidance); diff != "" {
		t.Fatal(diff)
	}
	raw, err := os.ReadFile(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Keep my notes.", "custom: retained", "source: local-source", "wire_agentsmd: false"} {
		if !bytes.Contains(raw, []byte(want)) {
			t.Fatalf("lost %q: %s", want, raw)
		}
	}
	for _, want := range []string{"Policy for first", "source.xyz", "*.xyz", "not now", "ignore"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q: %s", want, &out)
		}
	}
	if strings.Contains(out.String(), "Policy for existing") || strings.Contains(out.String(), "Policy for ignored") {
		t.Fatalf("re-prompted decided packs: %s", &out)
	}
	out.Reset()
	prompt.In = strings.NewReader("\n")
	if err := prompt.Select(t.Context(), root, catalogue, cfg); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Policy for first") || strings.Contains(out.String(), "Policy for second") || !strings.Contains(out.String(), "Policy for third") {
		t.Fatalf("repeat: %s", &out)
	}
}

func TestGuidancePrompt_InterruptedSessionDiscardsEveryAnswer(t *testing.T) {
	t.Parallel()
	root, cfg, catalogue := guidancePromptFixture(t)
	before, err := os.ReadFile(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatal(err)
	}
	original := cfg.Guidance
	prompt := GuidancePrompt{In: strings.NewReader("s\n"), Out: io.Discard}
	if err = prompt.Select(t.Context(), root, catalogue, cfg); !errors.Is(err, io.EOF) {
		t.Fatalf("interruption: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("partial answers persisted: %s", after)
	}
	if diff := cmp.Diff(original, cfg.Guidance); diff != "" {
		t.Fatal(diff)
	}
}

func TestGuidancePrompt_InstallsCompletedSelectionThroughInitAndRefresh(t *testing.T) {
	testsupport.IsolateGuidanceEnvironment(t)
	for _, initial := range []bool{true, false} {
		t.Run(fmt.Sprint(initial), func(t *testing.T) {
			root := t.TempDir()
			if err := gitops.Init(t.Context(), root); err != nil {
				t.Fatal(err)
			}
			source := testsupport.GuidanceSource(t, "*.xyz")

			writeAiwfYAML(t, root, "hosts: []\nguidance:\n  source: "+source+"\n")
			if writeErr := os.WriteFile(filepath.Join(root, "source.xyz"), []byte("source"), 0o644); writeErr != nil {
				t.Fatal(writeErr)
			}
			prompt := GuidancePrompt{In: strings.NewReader("s\n"), Out: io.Discard}
			var err error
			if initial {
				_, err = initrepo.Init(t.Context(), root, initrepo.Options{SkipHook: true, SelectGuidance: prompt.Select})
			} else {
				_, err = initrepo.RefreshArtifacts(t.Context(), root, initrepo.RefreshOptions{SkipHooks: true, SelectGuidance: prompt.Select})
			}
			if err != nil {
				t.Fatal(err)
			}
			installed, err := os.ReadFile(filepath.Join(root, ".guidance", "packs", "sample", "base", "guide.md"))
			if err != nil || string(installed) != "# Sample guidance\nInitial upstream content.\n" {
				t.Fatalf("same-invocation installation: %s, %v", installed, err)
			}
		})
	}
}

func TestGuidancePrompt_ErrorsNeverSaveAnswers(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("input/output failed")
	for _, kind := range []string{"input", "output", "cancelled", "missing config", "write denied"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			root, cfg, catalogue := guidancePromptFixture(t)
			initial := cfg.Guidance
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			prompt := GuidancePrompt{In: strings.NewReader("s\nn\nn\n"), Out: io.Discard}
			expected := sentinel
			switch kind {
			case "input":
				prompt.In = iotest.ErrReader(sentinel)
			case "output":
				prompt.Out = failedGuidanceWriter{sentinel}
			case "cancelled":
				cancel()
				expected = context.Canceled
			case "missing config":
				if err := os.Remove(filepath.Join(root, config.FileName)); err != nil {
					t.Fatal(err)
				}
				expected = os.ErrNotExist
			case "write denied":
				if os.Geteuid() == 0 {
					t.Skip("permission denial requires non-root")
				}
				if err := os.Chmod(root, 0o555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
				expected = os.ErrPermission
			}
			if err := prompt.Select(ctx, root, catalogue, cfg); !errors.Is(err, expected) {
				t.Fatalf("%s: %v", kind, err)
			}
			if diff := cmp.Diff(initial, cfg.Guidance); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

type failedGuidanceWriter struct{ err error }

func (w failedGuidanceWriter) Write([]byte) (int, error) { return 0, w.err }

func TestGuidancePrompt_IgnoreOnlyPreservesUnadoptedPolicy(t *testing.T) {
	t.Parallel()
	root, cfg, catalogue := guidancePromptFixture(t)
	writeAiwfYAML(t, root, "guidance: {}\n")
	cfg.Guidance = config.Guidance{}
	prompt := GuidancePrompt{In: strings.NewReader("i\ni\ni\ni\ni\n"), Out: io.Discard}
	if err := prompt.Select(t.Context(), root, catalogue, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Guidance.Packs != nil || len(loaded.Guidance.Ignored) != 5 {
		t.Fatalf("ignore adopted policy: %+v", loaded.Guidance)
	}
}

func TestGuidancePrompt_FailurePreservesInstalledPolicy(t *testing.T) {
	testsupport.IsolateGuidanceEnvironment(t)
	for _, interrupted := range []bool{false, true} {
		t.Run(fmt.Sprint(interrupted), func(t *testing.T) {
			root, source := t.TempDir(), testsupport.GuidanceSource(t)
			if err := gitops.Init(t.Context(), root); err != nil {
				t.Fatal(err)
			}
			writeAiwfYAML(t, root, "hosts: []\nguidance:\n  source: "+source+"\n  packs: [sample/base]\n")
			if _, err := initrepo.Init(t.Context(), root, initrepo.Options{SkipHook: true}); err != nil {
				t.Fatal(err)
			}
			indexPath := filepath.Join(root, ".guidance", "index.md")
			before, err := os.ReadFile(indexPath)
			if err != nil {
				t.Fatal(err)
			}
			originalConfig, err := os.ReadFile(filepath.Join(root, config.FileName))
			if err != nil {
				t.Fatal(err)
			}
			catalogue := `{"packs":[{"id":"sample/base","description":"Existing","files":["packs/sample/base/guide.md"],"detect":[]},{"id":"sample/new","description":"New policy","files":["packs/sample/new/missing.md"],"detect":["*.xyz"]},{"id":"sample/another","description":"Another policy","files":["packs/sample/another/missing.md"],"detect":["*.xyz"]}]}`
			if err = os.WriteFile(filepath.Join(source, "catalogue.json"), []byte(catalogue), 0o644); err != nil {
				t.Fatal(err)
			}
			testsupport.CommitGuidanceSource(t, source)
			if writeErr := os.WriteFile(filepath.Join(root, "source.xyz"), []byte("source"), 0o644); writeErr != nil {
				t.Fatal(writeErr)
			}
			input := "s\nn\n"
			if interrupted {
				input = "s\n"
			}
			prompt := GuidancePrompt{In: strings.NewReader(input), Out: io.Discard}
			result, err := initrepo.RefreshArtifacts(t.Context(), root, initrepo.RefreshOptions{SkipHooks: true, SelectGuidance: prompt.Select})
			if err != nil {
				t.Fatal(err)
			}
			var detail string
			for _, step := range result.Steps {
				if strings.Contains(step.What, "project engineering guidance") {
					detail = step.Detail
				}
			}
			if !strings.Contains(detail, "guidance incomplete") {
				t.Fatalf("failure hidden: %+v", result.Steps)
			}
			after, err := os.ReadFile(indexPath)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("installed index changed: %s, %v", after, err)
			}
			cfg, err := config.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			if interrupted {
				afterConfig, err := os.ReadFile(filepath.Join(root, config.FileName))
				if err != nil || !bytes.Equal(originalConfig, afterConfig) {
					t.Fatalf("interrupted answers saved: %s, %v", afterConfig, err)
				}
				if !strings.Contains(detail, "interrupted") {
					t.Fatalf("interruption hidden: %s", detail)
				}
			} else if diff := cmp.Diff([]string{"sample/base", "sample/new"}, *cfg.Guidance.Packs); diff != "" {
				t.Fatal(diff)
			}
			if _, err := os.Stat(filepath.Join(root, ".guidance", "packs", "sample", "new", "missing.md")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed selection installed: %v", err)
			}
		})
	}
}

func TestGuidancePrompt_CancellationDuringAnswersDiscardsSession(t *testing.T) {
	t.Parallel()
	for _, final := range []bool{false, true} {
		t.Run(fmt.Sprint(final), func(t *testing.T) {
			t.Parallel()
			root, cfg, catalogue := guidancePromptFixture(t)
			if final {
				catalogue.Packs = catalogue.Packs[2:3]
			}
			before, err := os.ReadFile(filepath.Join(root, config.FileName))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			prompt := GuidancePrompt{In: cancelGuidanceInput{strings.NewReader("s\n"), cancel}, Out: io.Discard}
			if selectErr := prompt.Select(ctx, root, catalogue, cfg); !errors.Is(selectErr, context.Canceled) {
				t.Fatalf("cancellation: %v", selectErr)
			}
			after, err := os.ReadFile(filepath.Join(root, config.FileName))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("cancelled answers saved: %s, %v", after, err)
			}
		})
	}
}

type cancelGuidanceInput struct {
	io.Reader
	cancel context.CancelFunc
}

func (r cancelGuidanceInput) Read(p []byte) (int, error) { r.cancel(); return r.Reader.Read(p) }

func TestGuidancePrompt_NotNowLeavesConfigurationByteIdentical(t *testing.T) {
	t.Parallel()
	root, cfg, catalogue := guidancePromptFixture(t)
	original := "# Project\nhosts: []\n"
	writeAiwfYAML(t, root, original)
	cfg.Guidance = config.Guidance{}
	prompt := GuidancePrompt{In: strings.NewReader("\n\n\n\n\n"), Out: io.Discard}
	if err := prompt.Select(t.Context(), root, catalogue, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, config.FileName))
	if err != nil || string(got) != original {
		t.Fatalf("not now wrote policy: %s, %v", got, err)
	}
}

func TestGuidancePrompt_SharedYAMLRefusalPreservesChoices(t *testing.T) {
	t.Parallel()
	root, _, catalogue := guidancePromptFixture(t)
	original := "defaults: &g {source: local}\nguidance: *g\n"
	writeAiwfYAML(t, root, original)
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	prompt := GuidancePrompt{In: strings.NewReader("s\nn\nn\nn\nn\n"), Out: io.Discard}
	if selectErr := prompt.Select(t.Context(), root, catalogue, cfg); selectErr == nil {
		t.Fatal("shared YAML edited")
	}
	got, err := os.ReadFile(filepath.Join(root, config.FileName))
	if err != nil || string(got) != original || cfg.Guidance.Packs != nil {
		t.Fatalf("shared YAML or in-memory choices changed: %s, %v", got, err)
	}
}

func TestGuidancePrompt_NoSelectionLeavesLegacyOwnership(t *testing.T) {
	testsupport.IsolateGuidanceEnvironment(t)
	for _, answer := range []string{"\n", "i\n"} {
		t.Run(answer, func(t *testing.T) {
			root, source := t.TempDir(), testsupport.GuidanceSource(t, "*")
			if err := gitops.Init(t.Context(), root); err != nil {
				t.Fatal(err)
			}
			writeAiwfYAML(t, root, "hosts: []\nguidance:\n  source: "+source+"\n")
			prompt := GuidancePrompt{In: strings.NewReader(answer), Out: io.Discard}
			result, err := initrepo.Init(t.Context(), root, initrepo.Options{SkipHook: true, SelectGuidance: prompt.Select})
			if err != nil {
				t.Fatal(err)
			}
			var detail string
			for _, step := range result.Steps {
				if strings.Contains(step.What, "project engineering guidance") {
					detail = step.Detail
				}
			}
			if !strings.Contains(detail, "no project guidance selected") {
				t.Fatalf("unexpected result: %s", detail)
			}
			cfg, err := config.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Guidance.Packs != nil {
				t.Fatal("prompt adopted empty policy")
			}
			if _, err := os.Stat(filepath.Join(root, ".guidance", "index.md")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("claimed guidance ownership: %v", err)
			}
		})
	}
}
