package policies

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PolicyCLAUDEMDCLIConventionsLogging is the mechanical evidence for
// M-0239/AC-5: CLAUDE.md's "### CLI conventions" section (under "##
// Go conventions") must describe the ADR-0017 shipped diagnostic-
// logging behavior — opt-in, default-off — not the stale "log/slog
// to stderr" prescription ADR-0017's own Context section quotes as
// the thing it replaces, and must cross-link ADR-0017 itself.
// Ratifying the ADR (`aiwf promote ADR-0017 accepted`) certifies a
// state that must already be true; this policy is what makes that
// certification mechanical rather than a claim taken on faith.
//
// Structural, not a whole-file substring grep (CLAUDE.md §"Substring
// assertions are not structural assertions"): extractMarkdownSubsection
// walks the heading hierarchy to isolate exactly the "CLI conventions"
// subsection's own prose, so a mention of ADR-0017 or "opt-in"
// anywhere else in this large file (there are many other sections)
// does not satisfy the check.
func PolicyCLAUDEMDCLIConventionsLogging(root string) ([]Violation, error) {
	const relPath = "CLAUDE.md"
	data, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		return []Violation{{
			Policy: "claudemd-cli-conventions-logging",
			File:   relPath,
			Detail: fmt.Sprintf("CLAUDE.md is unreadable: %v", err),
		}}, nil
	}

	section, found := extractMarkdownSubsection(string(data), "Go conventions", "CLI conventions")
	if !found {
		return []Violation{{
			Policy: "claudemd-cli-conventions-logging",
			File:   relPath,
			Detail: `no "### CLI conventions" section found under "## Go conventions" — the section AC-5 requires to describe ADR-0017's shipped logging behavior is missing or has been renamed`,
		}}, nil
	}

	lower := strings.ToLower(section)
	var vs []Violation
	if strings.Contains(lower, "log/slog` → stderr") || strings.Contains(lower, "log/slog to stderr") {
		vs = append(vs, Violation{
			Policy: "claudemd-cli-conventions-logging",
			File:   relPath,
			Detail: `"### CLI conventions" still describes the pre-ADR-0017 default ("log/slog to stderr") — ADR-0017 shipped opt-in, default-off diagnostic logging to an XDG-state-home file; rewrite this paragraph to match (CLAUDE.md §"Authoring an ADR": ratifying an ADR certifies a state that must already be true)`,
		})
	}
	if !strings.Contains(lower, "opt-in") && !strings.Contains(lower, "default-off") && !strings.Contains(lower, "default off") {
		vs = append(vs, Violation{
			Policy: "claudemd-cli-conventions-logging",
			File:   relPath,
			Detail: `"### CLI conventions" does not describe diagnostic logging as opt-in/default-off (ADR-0017 Decision #2)`,
		})
	}
	if !strings.Contains(section, "ADR-0017") {
		vs = append(vs, Violation{
			Policy: "claudemd-cli-conventions-logging",
			File:   relPath,
			Detail: `"### CLI conventions" does not cross-link ADR-0017`,
		})
	}
	return vs, nil
}

// extractMarkdownSubsection isolates the prose under a `### child`
// heading nested inside a `## parent` heading — CLAUDE.md's own
// two-level structure. Matching is heading-text-exact after
// trimming (case-sensitive, since every CLAUDE.md heading in this
// repo is written in a single consistent case). Returns the child
// section's body (excluding both heading lines) and true when found;
// ("", false) when the parent or the nested child heading is absent.
//
// entity.ParseBodySections (the kernel's own markdown-section reader)
// only tracks `## ` boundaries — everything under a `## ` heading,
// including its `### ` subsections, folds into one blob. That is
// correct for entity bodies (which never nest headings), but wrong
// here: CLAUDE.md's `### CLI conventions` is one of many `###`
// subsections inside the much larger `## Go conventions` block, and
// this policy needs exactly that one subsection's own prose, not
// the whole parent section.
func extractMarkdownSubsection(doc, parentHeading, childHeading string) (string, bool) {
	scanner := bufio.NewScanner(strings.NewReader(doc))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	inParent := false
	inChild := false
	var buf strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "## "):
			if inChild {
				return strings.TrimSpace(buf.String()), true
			}
			inParent = strings.TrimSpace(strings.TrimPrefix(line, "## ")) == parentHeading
			inChild = false
		case strings.HasPrefix(line, "### "):
			if inChild {
				return strings.TrimSpace(buf.String()), true
			}
			inChild = inParent && strings.TrimSpace(strings.TrimPrefix(line, "### ")) == childHeading
		case strings.HasPrefix(line, "# "):
			if inChild {
				return strings.TrimSpace(buf.String()), true
			}
			inParent = false
			inChild = false
		default:
			if inChild {
				buf.WriteString(line)
				buf.WriteString("\n")
			}
		}
	}
	if inChild {
		return strings.TrimSpace(buf.String()), true
	}
	return "", false
}
