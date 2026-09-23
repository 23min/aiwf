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

// guidanceVersionStamp is the header's version field. The Codex block
// omits it: that block lives in the tracked AGENTS.md, where a stamp
// would change with every writer's version while the guidance did not.
const guidanceVersionStamp = " aiwf-version: " + guidanceVersionSentinel

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
func RenderGuidance(ver string) ([]byte, error) {
	return renderGuidance(guidanceEmbed, ver, ClaudeRenderBindings())
}

// RenderCodexGuidance returns native instructions for an AGENTS.md block,
// resolving canonical paths against the Codex layout. The output carries
// no version, so every aiwf release writing the same guidance writes the
// same bytes.
func RenderCodexGuidance() ([]byte, error) {
	return renderGuidance(bytes.Replace(guidanceEmbed, []byte(guidanceVersionStamp), nil, 1), "", CodexRenderBindings())
}

func renderGuidance(source []byte, ver string, bindings RenderBindings) ([]byte, error) {
	stamped := bytes.ReplaceAll(source, []byte(guidanceVersionSentinel), []byte(ver))
	rendered, err := RenderSkills([]Skill{{Name: GuidanceFile, Content: stamped}}, bindings)
	if err != nil {
		return nil, err
	}
	return rendered[0].Content, nil
}

// MaterializeGuidance writes the guidance fragment to
// <root>/.claude/aiwf-guidance.md with the binary's current version
// substituted. Idempotent: rewriting identical content is a no-op diff.
func MaterializeGuidance(root string) error {
	return materializeGuidance(root, guidanceEmbed, version.Current().Version)
}

func materializeGuidance(root string, source []byte, ver string) error {
	content, err := renderGuidance(source, ver, ClaudeRenderBindings())
	if err != nil {
		return err
	}
	dest := filepath.Join(root, filepath.FromSlash(GuidanceFile))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(dest), err)
	}
	if err := pathutil.AtomicWriteFile(dest, content, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	return nil
}
