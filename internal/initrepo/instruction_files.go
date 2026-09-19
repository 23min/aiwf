package initrepo

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type instructionFileState struct {
	info    fs.FileInfo
	refusal string
}

type instructionFiles struct {
	claude instructionFileState
	agents instructionFileState
}

// inspectInstructionFiles guards all root instruction writers, including the
// first-time Claude scaffold. Both paths are inspected before either may write.
// Stat is used only to detect aliases, never to authorize writing through links.
// An unresolved link blocks both writers because disjoint targets cannot be
// established (including a dangling link to the other, not-yet-created file).
// This preflight does not lock paths against concurrent filesystem replacement.
func inspectInstructionFiles(ctx context.Context, root string) (instructionFiles, error) {
	var files instructionFiles
	if err := ctx.Err(); err != nil {
		return files, fmt.Errorf("inspecting instruction files: %w", err)
	}
	names := [2]string{"CLAUDE.md", "AGENTS.md"}
	states := [2]*instructionFileState{&files.claude, &files.agents}
	const remediation = "use separate regular files, or manage guidance manually and opt out via guidance.wire_claudemd / guidance.wire_agentsmd"
	for i, name := range names {
		info, err := os.Lstat(filepath.Join(root, name))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return instructionFiles{}, fmt.Errorf("inspecting %s: %w", name, err)
		}
		states[i].info = info
		if !info.Mode().IsRegular() {
			condition := "not a regular file"
			if info.Mode()&os.ModeSymlink != 0 {
				condition = "a symlink"
			}
			states[i].refusal = fmt.Sprintf("guidance incomplete: %s is %s; left untouched; %s", name, condition, remediation)
		}
	}
	resolved := [2]fs.FileInfo{files.claude.info, files.agents.info}
	for i, name := range names {
		if resolved[i] == nil || resolved[i].Mode()&os.ModeSymlink == 0 {
			continue
		}
		info, err := os.Stat(filepath.Join(root, name))
		if err != nil {
			detail := fmt.Sprintf("guidance incomplete: alias check for CLAUDE.md and AGENTS.md cannot inspect the target of the %s symlink (%v); both left untouched; %s", name, err, remediation)
			files.claude.refusal = detail
			files.agents.refusal = detail
			return files, nil
		}
		resolved[i] = info
	}
	if resolved[0] != nil && resolved[1] != nil && os.SameFile(resolved[0], resolved[1]) {
		detail := "guidance incomplete: CLAUDE.md and AGENTS.md alias the same underlying file; both left untouched; " + remediation
		files.claude.refusal = detail
		files.agents.refusal = detail
	}
	return files, nil
}
