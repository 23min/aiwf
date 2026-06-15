package skills

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/aiwf/internal/pathutil"
	"github.com/23min/aiwf/internal/version"
)

//go:embed embedded-guidance/aiwf-guidance.md
var guidanceEmbed []byte

// guidanceVersionSentinel is the placeholder in the embedded guidance
// fragment that RenderGuidance replaces with the binary's version
// string at materialization time (M-0163/AC-2).
const guidanceVersionSentinel = "__AIWF_VERSION__"

// GuidanceFile is the host-relative path of the materialized consumer
// CLAUDE.md guidance fragment. Unlike the scaffold-once statusline, it
// is byte-refreshed on every `aiwf init` / `aiwf update` (M-0163).
const GuidanceFile = ".claude/aiwf-guidance.md"

// GuidanceBytes returns the raw embedded consumer CLAUDE.md guidance
// fragment, with the version sentinel left unsubstituted.
func GuidanceBytes() []byte {
	return guidanceEmbed
}

// RenderGuidance returns the consumer CLAUDE.md guidance fragment with
// the version sentinel replaced by the given version string. This is
// the content aiwf materializes to `.claude/aiwf-guidance.md`.
func RenderGuidance(ver string) []byte {
	return bytes.ReplaceAll(guidanceEmbed, []byte(guidanceVersionSentinel), []byte(ver))
}

// MaterializeGuidance writes the guidance fragment to
// <root>/.claude/aiwf-guidance.md with the binary's current version
// substituted. Idempotent: rewriting identical content is a no-op diff.
func MaterializeGuidance(root string) error {
	dest := filepath.Join(root, filepath.FromSlash(GuidanceFile))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(dest), err)
	}
	content := RenderGuidance(version.Current().Version)
	if err := pathutil.AtomicWriteFile(dest, content, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	return nil
}
