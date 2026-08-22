package policies

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// changelogCheckWorkflowPath is the tag-push gate that verifies a
// pushed tag has a matching section in CHANGELOG.md. It is the second
// artefact TestReleaseRitualHeadingMatchesTagGate compares against;
// the first is the `aiwfx-release` ritual, which tells an operator
// what heading to write.
const changelogCheckWorkflowPath = ".github/workflows/changelog-check.yml"

// aiwfxWrapEpicChangelogSourcePath is the ritual that authors the
// per-epic entries `aiwfx-release` later promotes. It is the second
// artefact TestReleaseRitualShowsTheEntryShapeWrapProduces compares
// against.
const aiwfxWrapEpicChangelogSourcePath = aiwfxWrapEpicFixturePath

// sampleReleaseVersion stands in for a real version so the two
// artefacts can be compared by execution rather than by reading. Any
// three-part version works; this one is chosen to be obviously
// synthetic.
const sampleReleaseVersion = "1.2.3"

// versionPlaceholder is the token the ritual's templated heading uses
// where a real version goes.
const versionPlaceholder = "X.Y.Z"

// tagTriggerGlob matches the workflow's own push trigger — `- "v*"` —
// whose captured group is the tag namespace the gate fires on. Taking
// the tag shape from here rather than from the strip expression keeps
// the two independent: a gate that fires on one prefix and strips a
// different one is exactly the misconfiguration this check must catch.
var tagTriggerGlob = regexp.MustCompile(`(?m)^\s+-\s+"([^"*]*)\*"`)

// tagVersionDerivation matches the workflow's shell parameter
// expansion that turns a tag name into the version it greps for —
// `version="${tag#v}"`. The captured group is the prefix stripped
// from the tag, which is what the CHANGELOG heading must not carry.
var tagVersionDerivation = regexp.MustCompile(`version="\$\{tag#([^}]*)\}"`)

// changelogGrepPattern matches the workflow's `grep -E "<pattern>"`
// against CHANGELOG.md. The captured group is the pattern, still
// carrying its `$version` shell variable.
var changelogGrepPattern = regexp.MustCompile(`grep -E "([^"]+)"\s+CHANGELOG\.md`)

// bracketedH2 matches a top-level bracketed heading at line start —
// the shape both `## [Unreleased]` and a templated
// `## [X.Y.Z] — YYYY-MM-DD` take inside the ritual's fenced example.
var bracketedH2 = regexp.MustCompile(`(?m)^## \[([^\]]+)\].*$`)

// templatedEntryHeading matches the changelog-entry heading a wrap
// ritual templates: a heading whose text carries an id placeholder
// after an em-dash separator, e.g.
// `### <Added|Changed|Fixed> — E-NNNN: <one-line summary>`. The groups
// are the heading's hashes and its id placeholder.
var templatedEntryHeading = regexp.MustCompile(`(?m)^(#+) .* — ([A-Z]+-N+):`)

// fencedMarkdownBlock matches a ```markdown fenced example.
var fencedMarkdownBlock = regexp.MustCompile("(?s)```markdown\n(.*?)\n```")

// TestReleaseRitualHeadingMatchesTagGate pins G-0611 by comparing two
// artefacts rather than reading either one for a phrase: the heading
// the `aiwfx-release` ritual templates, and the pattern the tag-push
// gate greps for. It synthesises a concrete heading from the ritual's
// template, derives the gate's concrete pattern from a tag the gate's
// own trigger admits, and runs one against the other.
//
// The failure it exists to catch is not prose drift but a released
// tag that fails CI after it has been pushed: the gate fires on the
// tag push, and a pushed tag is not cheaply retracted. Either artefact
// moving independently turns this red — a ritual that reintroduces a
// prefix on the heading, a gate that stops stripping one, or a gate
// whose trigger and strip disagree.
func TestReleaseRitualHeadingMatchesTagGate(t *testing.T) {
	t.Parallel()

	workflow := readRepoFile(t, changelogCheckWorkflowPath)
	example := releaseRitualExample(t)

	// The tag the gate actually fires on, per its own trigger.
	trigger := tagTriggerGlob.FindStringSubmatch(workflow)
	if trigger == nil {
		t.Fatalf("G-0611: %s declares no `- \"<prefix>*\"` tag trigger; this check builds the sample tag from "+
			"that trigger so the gate's strip expression is tested against a tag it would really receive",
			changelogCheckWorkflowPath)
	}
	tag := trigger[1] + sampleReleaseVersion

	// The version it derives from that tag, and the pattern it greps.
	derivation := tagVersionDerivation.FindStringSubmatch(workflow)
	if derivation == nil {
		t.Fatalf("G-0611: %s no longer derives a version from the tag via `version=\"${tag#...}\"`; "+
			"this check compares that derivation against the ritual's templated heading and cannot run without it",
			changelogCheckWorkflowPath)
	}
	gateVersion := strings.TrimPrefix(tag, derivation[1])

	grepMatch := changelogGrepPattern.FindStringSubmatch(workflow)
	if grepMatch == nil {
		t.Fatalf("G-0611: %s no longer greps CHANGELOG.md via `grep -E \"<pattern>\"`; "+
			"this check needs that pattern to know what heading the gate accepts",
			changelogCheckWorkflowPath)
	}
	gatePattern, err := regexp.Compile(strings.ReplaceAll(grepMatch[1], "$version", gateVersion))
	if err != nil {
		t.Fatalf("G-0611: the pattern %s greps for does not compile as a Go regexp: %v",
			changelogCheckWorkflowPath, err)
	}

	// What the ritual tells the operator to write. `## [Unreleased]`
	// appears in the same fenced example and is not a version
	// heading, so it is excluded by name.
	var templated []string
	for _, m := range bracketedH2.FindAllStringSubmatch(example, -1) {
		if strings.TrimSpace(m[1]) == "Unreleased" {
			continue
		}
		templated = append(templated, m[0])
	}
	if len(templated) == 0 {
		t.Fatalf("G-0611: %s templates no versioned `## [...]` heading; the ritual must show the operator "+
			"the heading shape, and this check has nothing to compare against the tag gate",
			aiwfxReleaseFixturePath)
	}
	if len(templated) > 1 {
		t.Fatalf("G-0611: %s templates %d versioned `## [...]` headings (%q); the gate accepts one shape, "+
			"so the ritual must show exactly one",
			aiwfxReleaseFixturePath, len(templated), templated)
	}

	heading := strings.ReplaceAll(templated[0], versionPlaceholder, sampleReleaseVersion)
	if heading == templated[0] {
		t.Fatalf("G-0611: the ritual's heading %q carries no %q placeholder, so no concrete heading can be "+
			"synthesised from it to run against the tag gate",
			templated[0], versionPlaceholder)
	}

	if !gatePattern.MatchString(heading) {
		t.Errorf("G-0611: the heading %s tells the operator to write does not satisfy the gate in %s.\n"+
			"  ritual template:  %s\n"+
			"  synthesised:      %s\n"+
			"  gate pattern:     %s  (for pushed tag %q)\n"+
			"A release cut to the ritual fails CI *after* the tag is pushed, which is not cheaply retracted.",
			aiwfxReleaseFixturePath, changelogCheckWorkflowPath,
			templated[0], heading, gatePattern, tag)
	}
}

// TestReleaseRitualShowsTheEntryShapeWrapProduces pins G-0612 the same
// way: by comparing the release ritual's fenced example against the
// entry shape `aiwfx-wrap-epic` actually templates, rather than
// reading either for a phrase.
//
// The failure it catches is a release step that tells the operator to
// regroup the accumulated entries under category headings of its own.
// Those entries are already headings at the level the wrap ritual
// templates, so there is no nesting that puts one inside the other,
// and an operator following such a step has to invent the
// reconciliation. Restoring a category-group template turns this red,
// because a bare `### Added` carries no entry id.
//
// What it does not reach: the step's prose. A regrouping instruction
// written in the surrounding sentences, with the worked example left
// correct, keeps this green — an example is a shape two artefacts can
// be compared on and a sentence is not, and D-0070 retires the phrase
// assertion that would otherwise cover it. The `deployer` agent card,
// which paraphrases this step in one line and carries no example, is
// unreachable for the same reason. Both are held at review.
func TestReleaseRitualShowsTheEntryShapeWrapProduces(t *testing.T) {
	t.Parallel()

	wrap := readRepoFile(t, aiwfxWrapEpicChangelogSourcePath)
	example := releaseRitualExample(t)

	produced := templatedEntryHeading.FindStringSubmatch(wrap)
	if produced == nil {
		t.Fatalf("G-0612: %s templates no changelog-entry heading carrying an id (e.g. "+
			"`### <Category> — E-NNNN: <summary>`); this check derives the shape the release ritual must "+
			"preserve from that template and cannot run without it",
			aiwfxWrapEpicChangelogSourcePath)
	}
	level, idPlaceholder := produced[1], produced[2]

	// The release example must show an entry at the level the wrap
	// ritual emits, still carrying its id — not a bare category
	// group with the entries folded into bullets beneath.
	want := regexp.MustCompile(fmt.Sprintf(`(?m)^%s .* — [A-Z]+-N+:`, regexp.QuoteMeta(level)))
	if !want.MatchString(example) {
		t.Errorf("G-0612: %s's fenced example shows no entry at the shape %s produces.\n"+
			"  wrap templates:  %s ... — %s: ...\n"+
			"  release example:\n%s\n"+
			"The accumulated entries are already headings at that level; a release step that regroups them "+
			"under category headings of its own describes a nesting the two shapes do not admit.",
			aiwfxReleaseFixturePath, aiwfxWrapEpicChangelogSourcePath,
			level, idPlaceholder, example)
	}
}

// releaseRitualExample returns the `aiwfx-release` ritual's fenced
// worked example — the only part of the ritual with a shape two
// artefacts can be compared on. Both checks in this file read it
// rather than the whole ritual, so a heading appearing in the
// surrounding prose neither satisfies a check nor trips one.
//
// Exactly one ```markdown fence is required, for the same reason the
// caller requires exactly one version heading within it: the checks
// judge what they read, so a second one would ship a shape nothing
// here had judged. That guard reaches only fences opened as
// ```markdown — the ritual's other fences are ```bash, and widening
// it to any fence would swallow those. An example rendered in a bare
// or differently-tagged fence is not judged, and is caught at review
// like the step's prose.
func releaseRitualExample(t *testing.T) string {
	t.Helper()
	fences := fencedMarkdownBlock.FindAllStringSubmatch(loadAiwfxReleaseFixture(t), -1)
	if len(fences) == 0 {
		t.Fatalf("G-0612: %s shows no ```markdown fenced example of the section it produces; the example is "+
			"what tells the operator the accumulated entries stay as they are, and both checks here read it",
			aiwfxReleaseFixturePath)
	}
	if len(fences) > 1 {
		t.Fatalf("G-0612: %s shows %d ```markdown fenced examples; the checks here read one, so a second "+
			"ships a shape to the operator that nothing judged. Fold them into one example, or widen this "+
			"helper to return every fence and assert against each.",
			aiwfxReleaseFixturePath, len(fences))
	}
	return fences[0][1]
}
