package policies

import (
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/23min/aiwf/internal/projectguidance"
)

// PolicyInstructionFilesAtRoot holds this repository to one instruction
// file per host, at the root (M-0335 AC-5). A host loads a CLAUDE.md or
// AGENTS.md below the root when it works in that directory, so a nested one
// is primed guidance the ceiling never measures and the fence never judges;
// on-demand guidance is reached through the project router instead.
func PolicyInstructionFilesAtRoot(root string) ([]Violation, error) {
	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		return nil, fmt.Errorf("listing tracked files in %s: %w", root, err)
	}
	return instructionFilesBelowRoot(strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")), nil
}

// instructionFilesBelowRoot reports each tracked path naming a host
// instruction file anywhere but the repository root.
func instructionFilesBelowRoot(paths []string) []Violation {
	var vs []Violation
	for _, p := range paths {
		base := path.Base(p)
		if p == base || (base != projectguidance.ClaudeFile && base != projectguidance.AgentsFile) {
			continue
		}
		vs = append(vs, Violation{
			Policy: "instruction-files-at-root",
			File:   p,
			Detail: fmt.Sprintf("%s is a host instruction file below the repository root; a host loads it when working in that directory, outside the ceiling and the fence. Move its rules into a document .guidance/project.md routes to, and delete it.", p),
		})
	}
	return vs
}
