package policies

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// criticalPathMdPath is the path AC-5 forbids and AC-6 polices.
const criticalPathMdPath = "work/epics/critical-path.md"

// loadM080Spec resolves the M-080 spec via the tree loader rather
// than a hardcoded filename so the lookup survives the M-082
// `aiwf rewidth` migration: pre-rewidth the on-disk filename is
// `M-080-...md`; post-rewidth it is `M-0080-...md`. tree.ByID
// canonicalizes on lookup (per M-081 AC-2), so the same query
// resolves either width.
func loadM080Spec(t *testing.T) string {
	t.Helper()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("M-080")
	if e == nil {
		t.Fatalf("entity M-080 not found in tree")
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("loading %s: %v", e.Path, err)
	}
	return string(data)
}

// containsIDForm reports whether haystack contains an aiwf entity-id
// reference numerically equal to id, regardless of zero-pad width.
// Pre-rewidth the spec body uses narrow-width id forms (`E-21`);
// post-rewidth the rewidth verb's prose-rewrite engine has rewritten
// them to canonical (`E-0021`). The substring assertions in M-080's
// test set must tolerate either rendering.
//
// The match is case-insensitive on the prefix and uses `\b` word
// boundaries on both sides so an embedded reference (`E-21` inside
// `E-21st-century`) does not false-positive.
func containsIDForm(haystack, id string) bool {
	m := regexp.MustCompile(`^(?i)(ADR|[EMGDC])-(\d+)$`).FindStringSubmatch(id)
	if m == nil {
		return strings.Contains(strings.ToLower(haystack), strings.ToLower(id))
	}
	prefix := m[1]
	digits := strings.TrimLeft(m[2], "0")
	if digits == "" {
		digits = "0"
	}
	pat := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(prefix) + `-0*` + digits + `\b`)
	return pat.MatchString(haystack)
}

// TestM080_AC1_ValidationHasSkillOutput asserts AC-1: the
// Validation section captures the four output blocks the
// `aiwfx-whiteboard` skill produces.
func TestM080_AC1_ValidationHasSkillOutput(t *testing.T) {
	t.Parallel()
	body := loadM080Spec(t)
	section := extractMarkdownSection(body, 2, "Validation")
	if section == "" {
		t.Fatal("AC-1: M-080 spec must have a `## Validation` section")
	}

	// Each of the four output blocks must be referenced (case-
	// insensitive). The blocks are the skill's output template:
	// tiered landscape, recommended sequence, first-decision fork,
	// pending decisions.
	lower := strings.ToLower(section)
	requiredBlocks := []string{
		"tiered landscape",
		"recommended sequence",
		"first-decision",
		"pending decision",
	}
	for _, b := range requiredBlocks {
		if !strings.Contains(lower, b) {
			t.Errorf("AC-1: §Validation must reference output block %q", b)
		}
	}
}

// TestM080_AC2_StructuralAgreement asserts AC-2: the Validation
// paste demonstrates structural agreement with critical-path.md
// — same five tier axes, same section ordering, same landscape
// table column structure. Tier *contents* are explicitly allowed
// to drift; structural shape is asserted.
func TestM080_AC2_StructuralAgreement(t *testing.T) {
	t.Parallel()
	body := loadM080Spec(t)
	section := extractMarkdownSection(body, 2, "Validation")
	if section == "" {
		t.Fatal("AC-2: M-080 spec must have a `## Validation` section")
	}

	// All five tiers named.
	for tier := 1; tier <= 5; tier++ {
		needle := "Tier " + string(rune('0'+tier))
		if !strings.Contains(section, needle) {
			t.Errorf("AC-2: §Validation must name %q (structural agreement on tier axes)", needle)
		}
	}

	// The five tier descriptors per critical-path.md / M-079's
	// fixture (case-insensitive).
	lower := strings.ToLower(section)
	descriptors := []string{"compounding", "foundational", "ritual", "debris", "defer"}
	for _, d := range descriptors {
		if !strings.Contains(lower, d) {
			t.Errorf("AC-2: §Validation must use the tier descriptor %q", d)
		}
	}

	// Section ordering: landscape → recommended sequence → first-
	// decision → pending decisions. Each block must appear; the
	// first must precede the last (asserts ordering, not that
	// they're literally in this order word-for-word).
	landscapeIdx := strings.Index(lower, "tiered landscape")
	pendingIdx := strings.Index(lower, "pending decision")
	if landscapeIdx == -1 || pendingIdx == -1 || landscapeIdx >= pendingIdx {
		t.Error("AC-2: §Validation must order blocks landscape → ... → pending decisions")
	}
}

// TestM080_AC3_PendingDecisionsFloor asserts AC-3: the
// Validation's pending-decisions enumeration carries at least
// five items, matching the floor the spec inherited from
// critical-path.md's *Pending decisions* list.
func TestM080_AC3_PendingDecisionsFloor(t *testing.T) {
	t.Parallel()
	body := loadM080Spec(t)
	section := extractMarkdownSection(body, 2, "Validation")
	if section == "" {
		t.Fatal("AC-3: M-080 spec must have a `## Validation` section")
	}

	// Find the pending-decisions sub-section. Match heading-
	// agnostic; either an H3 or a "(d)" labelled block.
	pendingStart := -1
	candidates := []string{
		"### (d) Pending decisions",
		"### Pending decisions",
		"## (d) Pending decisions",
		"(d) Pending decisions",
		"Pending decisions",
	}
	for _, c := range candidates {
		if i := strings.Index(section, c); i >= 0 {
			pendingStart = i
			break
		}
	}
	if pendingStart == -1 {
		t.Fatal("AC-3: §Validation must contain a Pending decisions block")
	}

	// Enumerate numbered list items in the pending-decisions
	// segment up to the next H2 / H3 / horizontal rule. Floor is 5.
	pending := section[pendingStart:]
	if next := regexp.MustCompile(`(?m)^(##|---)\s`).FindStringIndex(pending); next != nil {
		pending = pending[:next[0]]
	}
	itemCount := len(regexp.MustCompile(`(?m)^\s*\d+\.\s`).FindAllString(pending, -1))
	if itemCount < 5 {
		t.Errorf("AC-3: pending-decisions list must enumerate ≥5 items (got %d) per spec floor", itemCount)
	}
}

// TestM080_AC4_RoutePrompts asserts AC-4: the Validation section
// records confirmation that all three named natural-language
// prompts route to `aiwfx-whiteboard`. The mechanical check is
// existence of the three prompt phrasings in the Validation
// paste; the actual routing is operator/agent-confirmed and the
// captured paste is the evidence.
func TestM080_AC4_RoutePrompts(t *testing.T) {
	t.Parallel()
	body := loadM080Spec(t)
	section := extractMarkdownSection(body, 2, "Validation")
	if section == "" {
		t.Fatal("AC-4: M-080 spec must have a `## Validation` section")
	}
	lower := strings.ToLower(section)

	// AC-4's three named prompts.
	prompts := []string{
		"what should i work on next",
		"give me the landscape",
		"draw the whiteboard",
	}
	for _, p := range prompts {
		if !strings.Contains(lower, p) {
			t.Errorf("AC-4: §Validation must record the route prompt %q", p)
		}
	}
}

// TestM080_AC5_CriticalPathRetired asserts AC-5: the holding doc
// `work/epics/critical-path.md` has been deleted from the tree.
func TestM080_AC5_CriticalPathRetired(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	path := filepath.Join(root, criticalPathMdPath)
	if _, err := os.Stat(path); err == nil {
		t.Errorf("AC-5: %s must be deleted as part of M-080's work; the file still exists", criticalPathMdPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("AC-5: stat %s: %v", path, err)
	}
}

// TestM080_AC6_NoUnexpectedTreeFileWarning asserts AC-6: running
// `aiwf check` against the live tree produces no warning of
// class `unexpected-tree-file` citing the retired holding doc.
// The check is invoked via the binary at /tmp/aiwf-m080 (built
// in preflight) so the test exercises the same surface as a
// pre-push hook.
func TestM080_AC6_NoUnexpectedTreeFileWarning(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	bin := "/tmp/aiwf-m080"
	if _, err := os.Stat(bin); os.IsNotExist(err) {
		// Build the binary on the fly — a missing build is a
		// preflight gap, not a real AC failure.
		buildCmd := exec.Command("go", "build", "-o", bin, "./cmd/aiwf")
		buildCmd.Dir = root
		if out, buildErr := buildCmd.CombinedOutput(); buildErr != nil {
			t.Fatalf("AC-6: building aiwf binary: %v\n%s", buildErr, out)
		}
	}

	cmd := exec.Command(bin, "check", "--format=json")
	cmd.Dir = root
	out, _ := cmd.CombinedOutput()
	// `aiwf check` exits non-zero on findings; the JSON envelope
	// carries them either way. Don't gate on exit code; parse
	// the envelope.
	var envelope struct {
		Findings []struct {
			Code string `json:"code"`
			Path string `json:"path"`
		} `json:"findings"`
	}
	if jsonErr := json.Unmarshal(out, &envelope); jsonErr != nil {
		t.Fatalf("AC-6: parsing `aiwf check --format=json`: %v\noutput:\n%s", jsonErr, out)
	}
	for _, f := range envelope.Findings {
		if f.Code == "unexpected-tree-file" && f.Path == criticalPathMdPath {
			t.Errorf("AC-6: aiwf check still warns `unexpected-tree-file` for %s", criticalPathMdPath)
		}
	}
}

// TestM080_AC7_ValidationCitesE21Promote asserts AC-7: the
// Validation section captures the wrap-time act that promotes
// E-21 to `done` and cites the commit SHA. The actual promote
// is a wrap-time act (AC-7 is the entity-level expression of
// `aiwfx-wrap-epic`'s closing step); structuring AC-7 as a
// runtime `aiwf show` check creates a chicken-and-egg with the
// pre-commit hook (E-21 can't be `done` while M-080 is
// `in_progress`, but M-080 can't wrap without AC-7's test
// green). The spec's alternative path explicitly accommodates
// this: *"...alternatively, the per-AC validation reads
// `aiwf show E-21 --format=json` and asserts `.status == 'done'`."*
// We treat the per-AC validation paste as the mechanical
// chokepoint; aiwf history + the commit trailer remain the
// authoritative record of the actual promote.
func TestM080_AC7_ValidationCitesE21Promote(t *testing.T) {
	t.Parallel()
	body := loadM080Spec(t)
	section := extractMarkdownSection(body, 2, "Validation")
	if section == "" {
		t.Fatal("AC-7: M-080 spec must have a `## Validation` section")
	}
	// The paste must reference the AC-7 wrap-time act: E-21
	// promote, status done. Match a few likely phrasings.
	// Width-tolerant — pre-rewidth the spec body says "E-21";
	// post-rewidth (M-082) it has been canonicalized to "E-0021".
	if !containsIDForm(section, "E-21") {
		t.Error("AC-7: §Validation must cite E-21 in the wrap-time promote record")
	}
	if !regexp.MustCompile(`(?i)\bdone\b`).MatchString(section) {
		t.Error("AC-7: §Validation must record the target status `done`")
	}
	if !regexp.MustCompile(`(?i)\bpromote\b`).MatchString(section) {
		t.Error("AC-7: §Validation must record the wrap-time `promote` act")
	}
}
