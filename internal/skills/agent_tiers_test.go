package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const sampleCard = "---\nname: reviewer\ndescription: does things\ntools: Read\ncolor: yellow\n---\n\n# Reviewer\n\nBody.\n"

func TestInjectAgentFrontmatter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		in          string
		tier        AgentTier
		wantHas     []string
		wantMissing []string
		wantSame    bool
	}{
		{
			name:    "model and effort",
			in:      sampleCard,
			tier:    AgentTier{Model: "sonnet", Effort: "high"},
			wantHas: []string{"model: sonnet", "effort: high", "name: reviewer", "# Reviewer"},
		},
		{
			name:        "model only omits effort",
			in:          sampleCard,
			tier:        AgentTier{Model: "haiku"},
			wantHas:     []string{"model: haiku"},
			wantMissing: []string{"effort:"},
		},
		{
			name:        "effort only omits model",
			in:          sampleCard,
			tier:        AgentTier{Effort: "xhigh"},
			wantHas:     []string{"effort: xhigh"},
			wantMissing: []string{"model:"},
		},
		{
			name:     "empty tier leaves card unchanged",
			in:       sampleCard,
			tier:     AgentTier{},
			wantSame: true,
		},
		{
			name:     "no frontmatter fence leaves card unchanged",
			in:       "# Reviewer\n\nNo frontmatter here.\n",
			tier:     AgentTier{Model: "sonnet"},
			wantSame: true,
		},
		{
			name:     "unterminated frontmatter leaves card unchanged",
			in:       "---\nname: reviewer\nno closing fence\n",
			tier:     AgentTier{Model: "sonnet"},
			wantSame: true,
		},
		{
			name:        "replaces existing model and effort (idempotent)",
			in:          "---\nname: reviewer\nmodel: opus\neffort: max\ncolor: yellow\n---\n# R\n",
			tier:        AgentTier{Model: "sonnet", Effort: "high"},
			wantHas:     []string{"model: sonnet", "effort: high", "name: reviewer", "color: yellow"},
			wantMissing: []string{"model: opus", "effort: max"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := string(injectAgentFrontmatter([]byte(tc.in), tc.tier))
			if tc.wantSame {
				if got != tc.in {
					t.Fatalf("content changed, want identical:\n%s", got)
				}
				return
			}
			for _, s := range tc.wantHas {
				if !strings.Contains(got, s) {
					t.Errorf("output missing %q\n---\n%s", s, got)
				}
			}
			for _, s := range tc.wantMissing {
				if strings.Contains(got, s) {
					t.Errorf("output unexpectedly contains %q\n---\n%s", s, got)
				}
			}
			// Injected keys must live inside the frontmatter, before the
			// closing fence — not leaked into the body.
			fmEnd := strings.Index(got, "\n---\n")
			if fmEnd < 0 {
				t.Fatalf("no closing frontmatter fence in output:\n%s", got)
			}
			fm := got[:fmEnd]
			for _, s := range tc.wantHas {
				if (strings.HasPrefix(s, "model:") || strings.HasPrefix(s, "effort:")) && !strings.Contains(fm, s) {
					t.Errorf("%q not inside frontmatter block", s)
				}
			}
		})
	}
}

func TestApplyAgentTiers(t *testing.T) {
	t.Parallel()
	newCards := func() []Skill {
		return []Skill{
			{Name: "reviewer.md", Content: []byte("---\nname: reviewer\n---\n# R\n")},
			{Name: "planner.md", Content: []byte("---\nname: planner\n---\n# P\n")},
		}
	}

	t.Run("nil tiers returns cards unchanged", func(t *testing.T) {
		t.Parallel()
		cards := newCards()
		got := applyAgentTiers(cards, nil)
		if len(got) != len(cards) {
			t.Fatalf("len = %d, want %d", len(got), len(cards))
		}
		for i := range cards {
			if !bytes.Equal(got[i].Content, cards[i].Content) {
				t.Errorf("card %d changed under nil tiers", i)
			}
		}
	})

	t.Run("matching card tiered, non-matching untouched, input not mutated", func(t *testing.T) {
		t.Parallel()
		cards := newCards()
		got := applyAgentTiers(cards, map[string]AgentTier{"reviewer": {Model: "sonnet", Effort: "high"}})
		byName := map[string]string{}
		for _, c := range got {
			byName[c.Name] = string(c.Content)
		}
		if !strings.Contains(byName["reviewer.md"], "model: sonnet") {
			t.Errorf("reviewer not tiered:\n%s", byName["reviewer.md"])
		}
		if strings.Contains(byName["planner.md"], "model:") {
			t.Errorf("planner unexpectedly tiered:\n%s", byName["planner.md"])
		}
		if strings.Contains(string(cards[0].Content), "model:") {
			t.Error("applyAgentTiers mutated the input slice's card content")
		}
	})
}

func TestAgentNames(t *testing.T) {
	t.Parallel()
	names, err := AgentNames()
	if err != nil {
		t.Fatalf("AgentNames: %v", err)
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("AgentNames not sorted: %v", names)
	}
	got := map[string]bool{}
	for _, n := range names {
		if strings.HasSuffix(n, ".md") {
			t.Errorf("name %q retains .md suffix", n)
		}
		got[n] = true
	}
	// The four shipped role agents must be present (subset check, so adding a
	// fifth agent later does not break this test).
	for _, want := range []string{"planner", "builder", "reviewer", "deployer"} {
		if !got[want] {
			t.Errorf("AgentNames missing %q; got %v", want, names)
		}
	}
}

func TestMaterializeWithTiersWritesFrontmatter(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tiers := map[string]AgentTier{"reviewer": {Model: "sonnet", Effort: "high"}}
	if err := MaterializeWithTiers(root, tiers); err != nil {
		t.Fatalf("MaterializeWithTiers: %v", err)
	}

	rev, err := os.ReadFile(filepath.Join(root, AgentsDir, "reviewer.md"))
	if err != nil {
		t.Fatalf("reading reviewer.md: %v", err)
	}
	for _, want := range []string{"model: sonnet", "effort: high"} {
		if !strings.Contains(string(rev), want) {
			t.Errorf("reviewer.md missing %q:\n%s", want, rev)
		}
	}

	// An agent with no config entry materializes without a model line.
	pl, err := os.ReadFile(filepath.Join(root, AgentsDir, "planner.md"))
	if err != nil {
		t.Fatalf("reading planner.md: %v", err)
	}
	if strings.Contains(string(pl), "\nmodel:") {
		t.Errorf("planner.md unexpectedly carries a model: line:\n%s", pl)
	}
}
