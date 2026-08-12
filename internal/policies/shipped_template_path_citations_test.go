package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

// A surface that materializes into a consumer repo may cite the templates
// directory, and may cite a file in it — but only one that materialization
// actually writes. A citation naming a file that never arrives sends a reader
// to run `aiwf update` for something no version of the command produces, and
// the reader most likely to follow it is an assistant with no prior about what
// the directory holds.
//
// The population is derived on both sides. Cited paths come from every shipped
// markdown surface; resolvable names come from what materialization writes. A
// hand-maintained list of either would drift from the thing it describes, which
// is the defect this guards.
var shippedSurfaceRoots = []string{
	filepath.Join("internal", "skills", "embedded"),
	filepath.Join("internal", "skills", "embedded-rituals"),
	filepath.Join("internal", "skills", "embedded-guidance"),
}

// templatePathCitation captures the path segment following the templates
// directory. The segment stops at whitespace, a backtick, or a closing
// bracket/paren, so a bare directory reference captures the empty string and a
// filename citation captures the filename.
var templatePathCitation = regexp.MustCompile("\\.claude/templates/([^\\s`)\\]]*)")

// TestShippedSurfaces_CiteOnlyMaterializedTemplatePaths fails when a shipped
// markdown surface names a templates path that materialization does not
// produce. Both citation shapes offend: a filename with no materialized
// counterpart, and a placeholder standing in for a per-kind filename — the
// templates are named for what they are (`epic-spec.md`, `adr.md`), not for
// their kind, so no single substitution resolves for every kind.
func TestShippedSurfaces_CiteOnlyMaterializedTemplatePaths(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	materialized := materializedTemplateNames(t)

	for _, dir := range shippedSurfaceRoots {
		walkRoot := filepath.Join(root, dir)
		err := filepath.WalkDir(walkRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			content, readErr := os.ReadFile(path) //nolint:gosec // walking a repo-relative tree of shipped sources
			if readErr != nil {
				return readErr
			}
			rel, _ := filepath.Rel(root, path)
			for _, m := range templatePathCitation.FindAllStringSubmatch(string(content), -1) {
				segment := m[1]
				switch {
				case segment == "":
					// A bare directory reference names no file and cannot rot.
				case strings.ContainsAny(segment, "<>"):
					t.Errorf("%s cites the placeholder templates path %q; the shipped templates are not named per kind, "+
						"so no substitution resolves for every kind — name the file, or name the directory and let the "+
						"reader read it", rel, m[0])
				case !materialized[segment]:
					t.Errorf("%s cites %q, which materialization does not write; materialized templates are %v",
						rel, m[0], sortedTemplateNames(materialized))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", dir, err)
		}
	}
}

// materializedTemplateNames is the resolvable set, read from what
// materialization writes rather than from a list beside it.
func materializedTemplateNames(t *testing.T) map[string]bool {
	t.Helper()
	templates, err := skills.ListRitualTemplates()
	if err != nil {
		t.Fatalf("listing ritual templates: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("no ritual templates found; the resolvable set is empty and every citation would fail")
	}
	names := make(map[string]bool, len(templates))
	for _, tpl := range templates {
		names[tpl.Name] = true
	}
	return names
}

func sortedTemplateNames(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
