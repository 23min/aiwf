package cliutil

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/config"
)

func TestReportGuidance_ExplainsSuggestionsAndRetainsUnmatchedSelection(t *testing.T) {
	t.Parallel()
	root, cfg, catalogue := guidancePromptFixture(t)
	catalogue.Packs[0].Detect = []string{"*.absent"}
	before, err := os.ReadFile(filepath.Join(root, config.FileName))
	if err != nil {
		t.Fatal(err)
	}
	original := cfg.Guidance
	var out bytes.Buffer
	if err = reportGuidance(t.Context(), root, catalogue, cfg, &out); err != nil {
		t.Fatal(err)
	}
	want := "Selected guidance existing has no current matching files; retained.\n" +
		"Suggested guidance first — Policy for first; matches source.xyz (*.xyz).\n" +
		"Suggested guidance second — Policy for second; matches source.xyz (*.xyz).\n" +
		"Suggested guidance third — Policy for third; matches source.xyz (*.xyz).\n" +
		"To select or ignore suggestions, edit guidance.packs or guidance.ignored in aiwf.yaml.\n"
	if diff := cmp.Diff(want, out.String()); diff != "" {
		t.Fatal(diff)
	}
	after, err := os.ReadFile(filepath.Join(root, config.FileName))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("configuration changed: %s, %v", after, err)
	}
	if diff := cmp.Diff(original, cfg.Guidance); diff != "" {
		t.Fatal(diff)
	}
}

func TestReportGuidance_NoMatchesOrDecidedMatchesStayQuiet(t *testing.T) {
	t.Parallel()
	for _, selected := range []bool{false, true} {
		t.Run(map[bool]string{false: "no matches", true: "decided matches"}[selected], func(t *testing.T) {
			t.Parallel()
			root, cfg, catalogue := guidancePromptFixture(t)
			if selected {
				catalogue.Packs = catalogue.Packs[:2]
			} else {
				cfg.Guidance.Packs = nil
				if err := os.Remove(filepath.Join(root, "source.xyz")); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			if err := reportGuidance(t.Context(), root, catalogue, cfg, &out); err != nil {
				t.Fatal(err)
			}
			if out.Len() != 0 {
				t.Fatalf("unexpected report: %s", &out)
			}
		})
	}
}

func TestReportGuidance_ReportsReadAndWriteErrors(t *testing.T) {
	t.Parallel()
	for _, stage := range []string{"detect", "selected", "suggested", "hint"} {
		t.Run(stage, func(t *testing.T) {
			t.Parallel()
			root, cfg, catalogue := guidancePromptFixture(t)
			failure := errors.New("output failed")
			var out bytes.Buffer
			writer := &failAfterLines{remaining: 0, err: failure}
			switch stage {
			case "detect":
				root = filepath.Join(root, "missing")
			case "selected":
				catalogue.Packs[0].Detect = nil
			case "hint":
				writer.remaining = 3
			}
			if stage == "detect" {
				if err := reportGuidance(t.Context(), root, catalogue, cfg, &out); err == nil {
					t.Fatal("missing detection error")
				}
			} else if err := reportGuidance(t.Context(), root, catalogue, cfg, writer); !errors.Is(err, failure) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

type failAfterLines struct {
	remaining int
	err       error
}

func (w *failAfterLines) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, w.err
	}
	w.remaining -= strings.Count(string(p), "\n")
	return len(p), nil
}
