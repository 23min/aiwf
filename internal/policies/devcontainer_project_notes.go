package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/23min/aiwf/internal/entity"
)

const (
	projectNotesPath   = ".devcontainer/project/README.md"
	kitPostCreatePath  = ".devcontainer/post-create.sh"
	playwrightHeading  = "Playwright tests"
	recoveryHeading    = "Recovery"
	rebuildInstruction = "Rebuild Container"
)

var (
	// e2eOptInRe finds the variable the project hook gates Playwright on,
	// written as ${NAME:-false}.
	e2eOptInRe = regexp.MustCompile(`\$\{([A-Z0-9_]+):-false\}`)
	// kitRecoveryRe finds the command the kit's post-create prints when a
	// step fails.
	kitRecoveryRe = regexp.MustCompile(`then run: ([^'\\]+)`)
)

// PolicyDevcontainerProjectNotes asserts that aiwf's container notes,
// .devcontainer/project/README.md, tell an operator how to opt into the
// Playwright install and how to recover a failed container creation, in
// terms taken from the scripts themselves: the Playwright section names the
// variable .devcontainer/project/post-create.sh reads, set to true, and says
// to rebuild; the Recovery section carries the command the kit's
// post-create prints. Either side changing without the other fails here.
func PolicyDevcontainerProjectNotes(root string) ([]Violation, error) {
	notes, err := os.ReadFile(filepath.Join(root, projectNotesPath))
	if err != nil {
		return []Violation{{Policy: "devcontainer-project-notes", File: projectNotesPath, Detail: fmt.Sprintf("missing or unreadable: %v", err)}}, nil
	}
	sections := entity.ParseBodySections(notes)
	var vs []Violation

	hook, err := os.ReadFile(filepath.Join(root, projectPostCreatePath))
	optIn := e2eOptInRe.FindSubmatch(hook)
	switch {
	case err != nil || optIn == nil:
		vs = append(vs, Violation{Policy: "devcontainer-project-notes", File: projectPostCreatePath, Detail: "no ${NAME:-false} Playwright opt-in to check the notes against"})
	default:
		body := sections[entity.SectionSlug(playwrightHeading)]
		if want := string(optIn[1]) + "=true"; !strings.Contains(body, want) || !strings.Contains(body, rebuildInstruction) {
			vs = append(vs, Violation{Policy: "devcontainer-project-notes", File: projectNotesPath, Detail: fmt.Sprintf(
				"the %q section must say to set %s and %s", playwrightHeading, want, rebuildInstruction)})
		}
	}

	kit, err := os.ReadFile(filepath.Join(root, kitPostCreatePath))
	recovery := kitRecoveryRe.FindSubmatch(kit)
	switch {
	case err != nil || recovery == nil:
		vs = append(vs, Violation{Policy: "devcontainer-project-notes", File: kitPostCreatePath, Detail: "no recovery command (\"then run: ...\") to check the notes against"})
	default:
		if want := strings.TrimSpace(string(recovery[1])); !strings.Contains(sections[entity.SectionSlug(recoveryHeading)], want) {
			vs = append(vs, Violation{Policy: "devcontainer-project-notes", File: projectNotesPath, Detail: fmt.Sprintf(
				"the %q section must carry the recovery command the kit prints: %s", recoveryHeading, want)})
		}
	}
	return vs, nil
}
