package skills

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRenderSkills_SubstitutesOnlyExplicitBindings(t *testing.T) {
	t.Parallel()
	bindings := RenderBindings{
		Target: Target{Name: "fixture-host", SkillsDir: "native/skills", AgentsDir: "native/agents", TemplatesDir: "support/templates", HooksDir: "native/hooks"},
		Fragments: HostFragments{
			SkillInvocation: "invoke the skill",
			WorktreeEntry:   "enter the checkout",
			ReviewDispatch:  "review with {{aiwf:agents_dir}}/reviewer.md",
		},
	}
	for _, tc := range []struct {
		name, source, want string
	}{
		{"empty", "", ""},
		{"literal", "Unicode λ\r\n.claude/worktrees/\n{{ordinary}}\t", "Unicode λ\r\n.claude/worktrees/\n{{ordinary}}\t"},
		{"paths", "{{aiwf:host}} {{aiwf:host_label}} {{aiwf:skills_dir}} {{aiwf:agents_dir}} {{aiwf:templates_dir}} {{aiwf:hooks_dir}}", "fixture-host fixture-host native/skills native/agents support/templates native/hooks"},
		{"fragments", "{{aiwf:fragment:skill_invocation}}\n{{aiwf:fragment:worktree_entry}}\n{{aiwf:fragment:review_dispatch}}", "invoke the skill\nenter the checkout\nreview with native/agents/reviewer.md"},
		{"adjacent and repeated", "{{aiwf:templates_dir}}{{aiwf:templates_dir}}\n", "support/templatessupport/templates\n"},
		{"unreserved delimiters", "{{anything:literal}} }} { {{aiwf", "{{anything:literal}} }} { {{aiwf"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sources := []Skill{{Name: "source.md", Content: []byte(tc.source)}}
			want := []Skill{{Name: "source.md", Content: []byte(tc.want)}}
			for range 2 {
				got, err := RenderSkills(sources, bindings)
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("rendered sources (-want +got):\n%s", diff)
				}
				if string(sources[0].Content) != tc.source {
					t.Fatal("rendering mutated the input")
				}
				if len(got[0].Content) > 0 {
					got[0].Content[0] = '!'
				}
			}
		})
	}
}

func TestRenderSkills_RequiresWorkflowSpecificFragments(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"epic_worktree_entry", "milestone_worktree_entry", "epic_worktree_placement", "milestone_worktree_placement", "epic_external_worktree", "milestone_external_worktree"} {
		sources := []Skill{{Name: "fixture", Content: []byte("{{aiwf:fragment:" + name + "}}")}}
		got, err := RenderSkills(sources, RenderBindings{Target: ClaudeTarget})
		if !errors.Is(err, ErrMissingRenderBinding) || got != nil {
			t.Errorf("%s: output = %v, error = %v", name, got, err)
		}
	}
}

func TestRenderSkills_RejectsInvalidInputsWithoutPartialOutput(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, source, token string
		bindings            RenderBindings
		want                error
	}{
		{"unknown binding", "{{aiwf:unknown}}", "unknown", RenderBindings{Target: ClaudeTarget}, ErrUnknownRenderBinding},
		{"unknown fragment", "{{aiwf:fragment:unknown}}", "fragment:unknown", RenderBindings{Target: ClaudeTarget}, ErrUnknownRenderFragment},
		{"missing host", "literal", "host", RenderBindings{}, ErrMissingRenderBinding},
		{"missing skills", "{{aiwf:skills_dir}}", "skills_dir", RenderBindings{Target: Target{Name: "claude"}}, ErrMissingRenderBinding},
		{"missing agents", "{{aiwf:agents_dir}}", "agents_dir", RenderBindings{Target: Target{Name: "claude"}}, ErrMissingRenderBinding},
		{"missing templates", "{{aiwf:templates_dir}}", "templates_dir", RenderBindings{Target: Target{Name: "claude"}}, ErrMissingRenderBinding},
		{"missing hooks", "{{aiwf:hooks_dir}}", "hooks_dir", RenderBindings{Target: Target{Name: "claude"}}, ErrMissingRenderBinding},
		{"missing invocation", "{{aiwf:fragment:skill_invocation}}", "fragment:skill_invocation", RenderBindings{Target: ClaudeTarget}, ErrMissingRenderBinding},
		{"missing worktree entry", "{{aiwf:fragment:worktree_entry}}", "fragment:worktree_entry", RenderBindings{Target: ClaudeTarget}, ErrMissingRenderBinding},
		{"missing review", "{{aiwf:fragment:review_dispatch}}", "fragment:review_dispatch", RenderBindings{Target: ClaudeTarget}, ErrMissingRenderBinding},
		{"unclosed token", "{{aiwf:templates_dir", "templates_dir", RenderBindings{Target: ClaudeTarget}, ErrInvalidRenderSyntax},
		{"empty token", "{{aiwf:}}", "{{aiwf:}}", RenderBindings{Target: ClaudeTarget}, ErrInvalidRenderSyntax},
		{"whitespace", "{{aiwf: templates_dir}}", " templates_dir", RenderBindings{Target: ClaudeTarget}, ErrInvalidRenderSyntax},
		{"newline", "{{aiwf:templates_dir\n}}", "templates_dir", RenderBindings{Target: ClaudeTarget}, ErrInvalidRenderSyntax},
		{"nested token", "{{aiwf:{{aiwf:host}}}}", "{{aiwf:host", RenderBindings{Target: ClaudeTarget}, ErrInvalidRenderSyntax},
		{"empty fragment name", "{{aiwf:fragment:}}", "fragment:", RenderBindings{Target: ClaudeTarget}, ErrInvalidRenderSyntax},
		{"fragment binding missing", "{{aiwf:fragment:review_dispatch}}", "agents_dir", RenderBindings{Target: Target{Name: "claude"}, Fragments: HostFragments{ReviewDispatch: "{{aiwf:agents_dir}}"}}, ErrMissingRenderBinding},
		{"fragment syntax invalid", "{{aiwf:fragment:review_dispatch}}", "host", RenderBindings{Target: ClaudeTarget, Fragments: HostFragments{ReviewDispatch: "{{aiwf:host"}}, ErrInvalidRenderSyntax},
		{"fragment nesting", "{{aiwf:fragment:review_dispatch}}", "fragment:review_dispatch", RenderBindings{Target: ClaudeTarget, Fragments: HostFragments{ReviewDispatch: "{{aiwf:fragment:review_dispatch}}"}}, ErrInvalidRenderSyntax},
		{"binding injects a token", "{{aiwf:templates_dir}}", "templates_dir", RenderBindings{Target: Target{Name: "claude", TemplatesDir: "{{aiwf:host}}"}}, ErrInvalidRenderSyntax},
		{"token assembled at boundary", "{{aiwf:templates_dir}}:host}}", "{{aiwf:host}}", RenderBindings{Target: Target{Name: "claude", TemplatesDir: "{{aiwf"}}, ErrInvalidRenderSyntax},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// A later invalid document must discard earlier rendered output.
			sources := []Skill{{Name: "first.md", Content: []byte("unchanged")}, {Name: "bad.md", Content: []byte(tc.source)}}
			if tc.bindings.Target.Name == "" {
				sources = sources[1:]
			}
			got, err := RenderSkills(sources, tc.bindings)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v; want %v", err, tc.want)
			}
			if got != nil {
				t.Errorf("returned partial output: %v", got)
			}
			for _, context := range []string{"bad.md", tc.bindings.Target.Name, tc.token} {
				if !strings.Contains(err.Error(), context) {
					t.Errorf("error %q lacks context %q", err, context)
				}
			}
		})
	}
}

func TestRenderSkills_PreservesBatchOrder(t *testing.T) {
	t.Parallel()
	sources := []Skill{
		{Name: "last.md", Content: []byte("{{aiwf:host}}")},
		{Name: "first.md", Content: []byte("{{aiwf:templates_dir}}")},
	}
	want := []Skill{
		{Name: "last.md", Content: []byte("claude")},
		{Name: "first.md", Content: []byte(TemplatesDir)},
	}
	got, err := RenderSkills(sources, RenderBindings{Target: ClaudeTarget})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("batch output (-want +got):\n%s", diff)
	}
}

func TestRenderSkills_EmptyBatchAndUnusedBindings(t *testing.T) {
	t.Parallel()
	got, err := RenderSkills(nil, RenderBindings{})
	if err != nil || got != nil {
		t.Fatalf("empty batch = %v, %v; want nil, nil", got, err)
	}
	// Optional host capabilities need no value unless a source uses them.
	got, err = RenderSkills([]Skill{{Name: "plain.md", Content: []byte("plain")}}, RenderBindings{Target: Target{Name: "host"}})
	if err != nil || len(got) != 1 || string(got[0].Content) != "plain" {
		t.Fatalf("unused bindings = %v, %v", got, err)
	}
}
