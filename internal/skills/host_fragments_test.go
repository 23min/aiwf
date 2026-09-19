package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Expected fragments come from their authoring files, independently of the
// production binding constructors and renderer. Literal shared bytes survive.
func TestHostFragments_SelectNativeSourcesAndPreserveSharedBytes(t *testing.T) {
	t.Parallel()
	for _, host := range []string{"claude", "codex"} {
		t.Run(host, func(t *testing.T) {
			t.Parallel()
			bindings := ClaudeRenderBindings()
			if host == "codex" {
				bindings = CodexRenderBindings()
			}
			root := t.TempDir()
			if err := MaterializeTo(root, bindings.Target); err != nil {
				t.Fatal(err)
			}
			verbs, err := listVerbSources()
			if err != nil {
				t.Fatal(err)
			}
			rituals, err := listRitualSources()
			if err != nil {
				t.Fatal(err)
			}
			sources := verbs
			sources = append(sources, rituals...)
			sources = append(sources, Skill{Name: GuidanceFile, Content: GuidanceBytes()})
			fragments := map[string]string{
				"skill_invocation": "skill-invocation", "review_dispatch": "review-dispatch", "worktree_entry": "worktree-entry",
				"epic_worktree_entry": "epic-worktree-entry", "milestone_worktree_entry": "milestone-worktree-entry",
				"epic_worktree_placement": "epic-worktree-placement", "milestone_worktree_placement": "milestone-worktree-placement",
				"epic_external_worktree": "epic-external-worktree", "milestone_external_worktree": "milestone-external-worktree",
			}
			label := "Claude Code"
			if host == "codex" {
				label = "Codex"
			}
			paths := strings.NewReplacer("{{aiwf:host}}", host, "{{aiwf:host_label}}", label,
				"{{aiwf:skills_dir}}", bindings.Target.SkillsDir, "{{aiwf:templates_dir}}", bindings.Target.TemplatesDir,
				"{{aiwf:agents_dir}}", bindings.Target.AgentsDir, "{{aiwf:hooks_dir}}", bindings.Target.HooksDir)
			var pairs []string
			for token, file := range fragments {
				if host == "codex" {
					switch {
					case strings.HasSuffix(token, "worktree_entry"):
						file = "worktree-entry"
					case strings.HasSuffix(token, "worktree_placement"):
						file = "worktree-placement"
					case strings.HasSuffix(token, "external_worktree"):
						file = "external-worktree"
					}
				}
				content, readErr := os.ReadFile(filepath.Join("embedded-guidance", host, file+".md"))
				if readErr != nil {
					t.Fatal(readErr)
				}
				pairs = append(pairs, "{{aiwf:fragment:"+token+"}}", paths.Replace(strings.TrimSuffix(string(content), "\n")))
			}
			substitute := strings.NewReplacer(pairs...)
			seen := map[string]bool{}
			for _, source := range sources {
				for token := range fragments {
					if bytes.Contains(source.Content, []byte("{{aiwf:fragment:"+token+"}}")) {
						seen[token] = true
					}
				}
				want := paths.Replace(substitute.Replace(string(source.Content)))
				var got []byte
				if source.Name == GuidanceFile {
					want = strings.ReplaceAll(want, guidanceVersionSentinel, "fragment-test")
					if host == "claude" {
						got, err = RenderGuidance("fragment-test")
					} else {
						got, err = RenderCodexGuidance("fragment-test")
					}
				} else {
					got, err = os.ReadFile(filepath.Join(root, bindings.Target.SkillsDir, source.Name, "SKILL.md"))
				}
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != want {
					t.Errorf("%s differs from its host fragments or shared source", source.Name)
				}
			}
			for token := range fragments {
				if !seen[token] {
					t.Errorf("operational fragment %s has no canonical call site", token)
				}
			}
			// Every concrete skill path in a selected fragment must be readable.
			refs := regexp.MustCompile(regexp.QuoteMeta(bindings.Target.SkillsDir) + `/[a-z0-9-]+/SKILL\.md`)
			for i := 1; i < len(pairs); i += 2 {
				for _, ref := range refs.FindAllString(pairs[i], -1) {
					if _, readErr := os.ReadFile(filepath.Join(root, ref)); readErr != nil {
						t.Errorf("fragment reference %s: %v", ref, readErr)
					}
				}
			}
		})
	}
}
