package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// Checks that hold a shipped skill against a second artefact rather than
// against a phrase: the kernel's own enumerations, and the filesystem. Each
// derives its expectation by asking the code, so it fails whether the skill
// drifts or the kernel does — the reach that earns the class its place under
// D-0070, where a restated phrase would only pin a reading.
//
// The prose halves these once shared a file with are retired; what remains is
// the derived half alone.

// TestAiwfCheckSkill_ACStatusSetMatchesKernel holds the aiwf-check skill's
// acs-shape/status row against entity.AllowedACStatuses(). Adding an AC status
// to the kernel without documenting it goes red here.
func TestAiwfCheckSkill_ACStatusSetMatchesKernel(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, aiwfCheckSkillPath)

	row := ""
	for _, ln := range strings.Split(body, "\n") {
		if strings.Contains(ln, "acs-shape/status") {
			row = ln
			break
		}
	}
	if row == "" {
		t.Fatal("aiwf-check skill has no acs-shape/status row")
	}
	for _, s := range entity.AllowedACStatuses() {
		if !strings.Contains(row, string(s)) {
			t.Errorf("acs-shape/status row omits kernel AC status %q; row = %q", s, row)
		}
	}
}

// TestAiwfArchiveSkill_KindFlagsResolve holds every `--kind <x>` the
// aiwf-archive skill demonstrates against entity.AllKinds(), so a fabricated
// kind cannot be taught. This is the source-derived half of the retired
// prose guard, which searched for one phrasing of one mistake.
func TestAiwfArchiveSkill_KindFlagsResolve(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, aiwfArchiveSkillPath)

	valid := map[string]bool{}
	for _, k := range entity.AllKinds() {
		valid[string(k)] = true
	}
	shown := regexp.MustCompile(`--kind\s+([a-z]+)`).FindAllStringSubmatch(body, -1)
	if len(shown) == 0 {
		t.Fatal("aiwf-archive skill demonstrates no `--kind` flag; this check has stopped covering anything")
	}
	for _, m := range shown {
		if !valid[m[1]] {
			t.Errorf("aiwf-archive shows `--kind %s`, which is not a real entity kind (kinds: %v)", m[1], entity.AllKinds())
		}
	}
}

// TestAiwfContractSkill_RecipePathResolves resolves the recipe path the
// aiwf-contract skill cites against the filesystem — a reference checked
// against its target, which no rewording satisfies falsely.
func TestAiwfContractSkill_RecipePathResolves(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, aiwfContractSkillPath)

	cited := regexp.MustCompile(`internal/recipe/[a-z/]+`).FindString(body)
	if cited == "" {
		t.Fatal("aiwf-contract skill cites no recipe path")
	}
	if _, err := os.Stat(filepath.Join(repoRoot(t), cited)); err != nil {
		t.Errorf("recipe path %q cited by aiwf-contract does not exist on disk: %v", cited, err)
	}
}

// TestAiwfContractSkill_CancelFSMIsComplete holds the skill's cancel section
// against entity.CancelTarget, so a kernel transition that gains a target
// without being documented goes red.
func TestAiwfContractSkill_CancelFSMIsComplete(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, aiwfContractSkillPath)

	cancelSection := sectionUnder(body, "Cancel a contract entirely")
	if cancelSection == "" {
		t.Fatal(`aiwf-contract skill has no "Cancel a contract entirely" section`)
	}
	covered := 0
	for _, from := range []entity.Status{
		entity.StatusProposed, entity.StatusAccepted,
		entity.StatusDeprecated, entity.StatusRetired, entity.StatusRejected,
	} {
		to := entity.CancelTarget(entity.KindContract, from)
		if to == "" {
			continue
		}
		covered++
		if !strings.Contains(cancelSection, string(to)) {
			t.Errorf("cancel section omits cancel target %q (from %q)", to, from)
		}
	}
	if covered == 0 {
		t.Fatal("no contract status has a cancel target; this check has stopped covering anything")
	}
}
