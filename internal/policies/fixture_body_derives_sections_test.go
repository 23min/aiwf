package policies

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// fixtureBodyBuilders are the sources that construct entity bodies for the four
// born-complete kinds. Those kinds refuse an empty body at creation — they have
// no draft phase in which to fill one in — so a fixture creating one must supply
// real prose under every section its kind names, which is why these two hand-rolled
// a filled variant of the scaffold rather than calling entity.BodyTemplate.
//
// The population is these files rather than a pattern; a further builder is
// caught at review, not here. Three is what a sweep found, against the two the
// milestone's inventory named — the third being a byte-identical copy of the
// second, which is how transcription spreads.
var fixtureBodyBuilders = []string{
	filepath.Join("internal", "cli", "doctor", "selfcheck.go"),
	filepath.Join("internal", "cellcoverage", "fixture.go"),
	filepath.Join("internal", "verb", "verb_test.go"),
}

// TestFixtureBodiesDeriveSectionsFromOwnedSet pins that neither body-building
// fixture spells out a section heading.
//
// Both fixtures name the right sections today, so comparing their output against
// the owned set would pass whether or not they derive it — and would be a
// tautology once they did. What can fail is the spelling itself: a heading
// literal in these files means the sections were transcribed, and a transcription
// is what goes stale.
//
// The failure this forecloses is the worst-shaped in the epic's inventory. Should
// a kind's set gain a section, a transcribing fixture builds an entity missing it,
// and because nothing reports an absent heading, `aiwf doctor --self-check` goes
// on passing while creating exactly the defect the epic exists to prevent — in the
// one command a consumer runs to ask whether their install is healthy.
func TestFixtureBodiesDeriveSectionsFromOwnedSet(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, rel := range fixtureBodyBuilders {
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, filepath.Join(root, rel), nil, 0)
			if err != nil {
				t.Fatalf("parsing %s: %v", rel, err)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					return true
				}
				if !carriesSectionHeading(value) {
					return true
				}
				t.Errorf("%s:%d: string literal spells out a section heading; build the body from entity.RequiredSections instead, or a kind that later gains a section leaves this fixture creating an entity without it — which no surface reports",
					rel, fset.Position(lit.Pos()).Line)
				return true
			})
		})
	}
}

// carriesSectionHeading reports whether a string literal's value contains a
// markdown `## ` heading — at its start or after a newline, so prose merely
// mentioning the characters is not flagged.
func carriesSectionHeading(value string) bool {
	return strings.HasPrefix(value, "## ") || strings.Contains(value, "\n## ")
}
