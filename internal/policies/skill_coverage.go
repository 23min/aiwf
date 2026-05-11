package policies

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// PolicySkillCoverageMatchesVerbs is M-072 / E-20's third leg of the
// AI-discoverability surface, modeled on PolicyFindingCodesAreDiscoverable
// and PolicyConfigFieldsAreDiscoverable. Asserts four mechanically
// evaluable invariants over the embedded skills + Cobra command tree:
//
//  1. Every embedded skill carries non-empty `name:` and `description:`
//     frontmatter fields (the host's discovery surface depends on
//     description-match scoring).
//  2. Every embedded skill's `name:` matches its directory name and
//     conforms to the `aiwf-<topic>` convention.
//  3. Every top-level Cobra command (i.e. every `cmd.AddCommand(...)`
//     called from `newRootCmd`) is either covered by a same-named
//     embedded skill (`aiwf-<verb>`) or appears in
//     skillCoverageAllowlist with a rationale.
//  4. Every backticked `aiwf <verb>` mention inside a skill body
//     resolves to a registered top-level Cobra command. G-061's
//     repro — a shipped skill body referencing a non-existent verb
//     (`aiwf list contracts` before M-072 landed) — fails this check.
//
// Judgment-shaped decisions (when to use a per-verb skill vs. a
// topical multi-verb skill, when --help suffices) live in the
// companion ADR and CLAUDE.md `Skills policy` section. This policy
// captures only the mechanical companion.
func PolicySkillCoverageMatchesVerbs(root string) ([]Violation, error) {
	skills, err := loadEmbeddedSkillsForPolicy(root)
	if err != nil {
		return nil, err
	}
	verbs, err := findTopLevelVerbs(root)
	if err != nil {
		return nil, err
	}
	return runSkillCoverageChecks(skills, verbs, skillCoverageAllowlist), nil
}

// runSkillCoverageChecks applies every skill-coverage invariant to the
// already-loaded inputs. Split from PolicySkillCoverageMatchesVerbs so
// negative-case unit tests can drive the checks with synthetic inputs
// without a tempdir fixture (CLAUDE.md §"Test untested code paths"
// — a positive-only test of the full policy proves nothing about
// whether the policy fires on real drift).
func runSkillCoverageChecks(
	skills []embeddedSkillEntry,
	verbs map[string]string,
	allowlist map[string]string,
) []Violation {
	var out []Violation
	out = append(out, checkSkillFrontmatter(skills)...)
	out = append(out, checkVerbCoverage(skills, verbs, allowlist)...)
	out = append(out, checkSkillBodyMentionsResolve(skills, verbs)...)
	return out
}

// checkSkillFrontmatter enforces M-074 AC-2 and AC-3: every embedded
// skill carries a non-empty `name:` matching its directory and the
// `aiwf-<topic>` convention, and a non-empty `description:` (the host's
// match-scoring depends on it).
func checkSkillFrontmatter(skills []embeddedSkillEntry) []Violation {
	var out []Violation
	for _, s := range skills {
		switch {
		case s.frontmatterName == "":
			out = append(out, Violation{
				Policy: "skill-coverage",
				File:   s.relPath,
				Detail: "embedded skill is missing a `name:` frontmatter field",
			})
		case s.frontmatterName != s.dirName:
			out = append(out, Violation{
				Policy: "skill-coverage",
				File:   s.relPath,
				Detail: fmt.Sprintf("embedded skill `name: %s` does not match its directory %q", s.frontmatterName, s.dirName),
			})
		case !strings.HasPrefix(s.frontmatterName, "aiwf-") || len(s.frontmatterName) <= len("aiwf-"):
			out = append(out, Violation{
				Policy: "skill-coverage",
				File:   s.relPath,
				Detail: fmt.Sprintf("embedded skill name %q does not match the `aiwf-<topic>` convention", s.frontmatterName),
			})
		}
		if strings.TrimSpace(s.description) == "" {
			out = append(out, Violation{
				Policy: "skill-coverage",
				File:   s.relPath,
				Detail: "embedded skill is missing a `description:` frontmatter field — the host's match-scoring depends on it",
			})
		}
	}
	return out
}

// checkVerbCoverage enforces M-074 AC-4: every top-level Cobra verb is
// either covered by a same-named `aiwf-<verb>` skill or appears in the
// allowlist with a rationale. Topical skills (precedent:
// `aiwf-contract`) cover only their primary verb; additional verbs in
// the topical bundle must be explicitly allowlisted so the rationale
// stays visible.
func checkVerbCoverage(
	skills []embeddedSkillEntry,
	verbs map[string]string,
	allowlist map[string]string,
) []Violation {
	skillCovered := map[string]bool{}
	for _, s := range skills {
		if strings.HasPrefix(s.frontmatterName, "aiwf-") {
			skillCovered[strings.TrimPrefix(s.frontmatterName, "aiwf-")] = true
		}
	}
	var out []Violation
	for verb := range verbs {
		if skillCovered[verb] {
			continue
		}
		if _, ok := allowlist[verb]; ok {
			continue
		}
		out = append(out, Violation{
			Policy: "skill-coverage",
			File:   "cmd/aiwf/main.go",
			Detail: fmt.Sprintf("top-level verb %q has no embedded skill (no `internal/skills/embedded/aiwf-%s/`) and no entry in skillCoverageAllowlist — add a skill or allowlist the verb with a one-line rationale", verb, verb),
		})
	}
	return out
}

// checkSkillBodyMentionsResolve enforces M-074 AC-5: every backticked
// `aiwf <verb>` mention inside a skill body resolves to a registered
// top-level Cobra verb. The first word after `aiwf` is the chokepoint;
// full-path validation (subverbs, args) is out of scope by design —
// see the policy header godoc.
func checkSkillBodyMentionsResolve(
	skills []embeddedSkillEntry,
	verbs map[string]string,
) []Violation {
	verbSet := map[string]bool{}
	for v := range verbs {
		verbSet[v] = true
	}
	// Cobra auto-adds `help` and `completion` at the root; mentions of
	// those resolve too, even though they're not in cmd/aiwf source.
	verbSet["help"] = true
	verbSet["completion"] = true

	var out []Violation
	for _, s := range skills {
		for _, m := range backtickedAiwfMentions(s.body) {
			if !verbSet[m.verb] {
				out = append(out, Violation{
					Policy: "skill-coverage",
					File:   s.relPath,
					Detail: fmt.Sprintf("skill body references `aiwf %s` but %q is not a registered top-level verb (the inverse of the kernel principle that AI-discoverable functionality must resolve)", m.verb, m.verb),
				})
			}
		}
	}
	return out
}

// skillCoverageAllowlist names every top-level Cobra verb that ships
// without a same-named `aiwf-<verb>` embedded skill. Each entry must
// carry a one-line rationale comment so a reviewer (human or AI) can
// see at a glance why this verb skips skill coverage.
//
// The entry for `show` is deliberately marked "deferred" — `aiwf show`
// is the per-entity inspection verb every AI assistant reaches for,
// and warrants its own skill. The follow-up gap (filed under
// `work/gaps/`) tracks the absence so it is not papered over.
var skillCoverageAllowlist = map[string]string{
	// Operator / install-time verbs — invoked at setup, not in everyday flow.
	"init":    "ops verb; one-time consumer-repo setup, --help suffices",
	"update":  "ops verb; refresh embedded skills + hooks, --help suffices",
	"upgrade": "ops verb; binary upgrade via `go install`, --help suffices",
	"doctor":  "ops verb; health/drift check, --help suffices",
	"import":  "ops verb; bulk-create from manifest, --help suffices",

	// Trivially documented verbs — closed-set, single-purpose, --help is authoritative.
	"version":  "trivial read-only; prints semver, --help suffices",
	"whoami":   "trivial read-only; prints actor, --help suffices",
	"schema":   "trivial read-only; prints frontmatter contract, --help suffices",
	"template": "trivial read-only; prints body-section template, --help suffices",

	// Mutation-light verbs whose closed-set semantics are obvious from --help.
	"cancel":  "convenience wrapper over promote (cancel = promote-to-terminal); aiwf-promote skill covers the lifecycle",
	"move":    "rare cross-epic milestone move; --help + the `aiwf-promote`/`aiwf-add` skills cover the surrounding flow",
	"rewidth": "one-shot migration ritual per ADR-0008; --help is sufficient discovery surface (ADR-0006 'no skill when --help suffices' case)",

	// Kind-namespace parent commands — non-Runnable; subverbs are documented elsewhere.
	"milestone": "kind-namespace parent (subverb `depends-on` for cross-milestone deps); the surrounding `aiwf-add` and `aiwf-promote` skills cover the flow",

	// Deferred — explicitly tracked, not papered over.
	"show":    "deferred — see G-087 (a per-entity inspection skill warrants its own design pass; --help covers the surface mechanically in the meantime)",
	"archive": "embedded skill lands in M-0088 (E-0024 epic); --help suffices in the M-0085 verb landing window",
}

// embeddedSkillEntry is the parsed shape of one
// internal/skills/embedded/aiwf-<topic>/SKILL.md file.
type embeddedSkillEntry struct {
	relPath         string // forward-slash, repo-relative
	dirName         string // e.g. "aiwf-list"
	frontmatterName string // value of `name:` in frontmatter
	description     string // value of `description:`
	body            string // markdown after the closing `---`
}

// loadEmbeddedSkillsForPolicy reads every SKILL.md under
// internal/skills/embedded/ and parses its frontmatter + body. Returns
// in directory-name-sorted order so violations surface deterministically.
func loadEmbeddedSkillsForPolicy(root string) ([]embeddedSkillEntry, error) {
	embeddedRoot := filepath.Join(root, "internal", "skills", "embedded")
	dirs, err := os.ReadDir(embeddedRoot)
	if err != nil {
		return nil, err
	}
	var out []embeddedSkillEntry
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		path := filepath.Join(embeddedRoot, d.Name(), "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil {
			// Missing SKILL.md surfaces as a violation on the directory
			// rather than an error here — every embedded/* dir should
			// hold a SKILL.md.
			out = append(out, embeddedSkillEntry{
				relPath: filepath.ToSlash(filepath.Join("internal", "skills", "embedded", d.Name(), "SKILL.md")),
				dirName: d.Name(),
			})
			continue
		}
		entry := parseSkillMarkdown(data)
		entry.relPath = filepath.ToSlash(filepath.Join("internal", "skills", "embedded", d.Name(), "SKILL.md"))
		entry.dirName = d.Name()
		out = append(out, entry)
	}
	return out, nil
}

// parseSkillMarkdown splits the SKILL.md into frontmatter (the `---`
// fenced block at the top) and body. From the frontmatter it extracts
// `name:` and `description:` line values. Frontmatter lines may wrap;
// the parser collects continuation lines (indented or non-key) into
// the previous field's value.
func parseSkillMarkdown(data []byte) embeddedSkillEntry {
	s := string(data)
	if !strings.HasPrefix(s, "---\n") {
		return embeddedSkillEntry{body: s}
	}
	end := strings.Index(s[4:], "\n---")
	if end < 0 {
		return embeddedSkillEntry{body: s}
	}
	frontmatter := s[4 : 4+end]
	body := s[4+end:]
	body = strings.TrimPrefix(body, "\n---")
	body = strings.TrimPrefix(body, "\n")

	var entry embeddedSkillEntry
	entry.body = body

	var currentKey string
	var name, description strings.Builder
	for _, line := range strings.Split(frontmatter, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			// New key.
			if i := strings.Index(line, ":"); i > 0 {
				currentKey = strings.TrimSpace(line[:i])
				val := strings.TrimSpace(line[i+1:])
				switch currentKey {
				case "name":
					name.WriteString(val)
				case "description":
					description.WriteString(val)
				}
				continue
			}
		}
		// Continuation — append to the current field's accumulator.
		switch currentKey {
		case "name":
			name.WriteString(" " + strings.TrimSpace(line))
		case "description":
			description.WriteString(" " + strings.TrimSpace(line))
		}
	}

	entry.frontmatterName = strings.TrimSpace(name.String())
	entry.description = strings.TrimSpace(description.String())
	return entry
}

// findTopLevelVerbs returns the set of top-level verb names registered
// at the root of newRootCmd. Each is mapped to the `newXCmd` function
// that constructs it (used in error messages so a violation points at
// the right file).
//
// The walk: AST-parse main.go, find newRootCmd, collect every
// `cmd.AddCommand(newXCmd())` call. For each newXCmd, parse its
// definition's body and extract the `Use:` field's string literal —
// that's the verb name as the user types it.
func findTopLevelVerbs(root string) (map[string]string, error) {
	cmdDir := filepath.Join(root, "cmd", "aiwf")
	mainPath := filepath.Join(cmdDir, "main.go")
	fset := token.NewFileSet()
	mainAST, err := parser.ParseFile(fset, mainPath, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	builders := map[string]bool{}
	ast.Inspect(mainAST, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != "newRootCmd" {
			return true
		}
		ast.Inspect(fn, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "AddCommand" {
				return true
			}
			for _, arg := range call.Args {
				inner, ok := arg.(*ast.CallExpr)
				if !ok {
					continue
				}
				id, ok := inner.Fun.(*ast.Ident)
				if !ok {
					continue
				}
				if strings.HasPrefix(id.Name, "new") && strings.HasSuffix(id.Name, "Cmd") {
					builders[id.Name] = true
				}
			}
			return true
		})
		return false
	})

	out := map[string]string{}
	files, err := filepath.Glob(filepath.Join(cmdDir, "*.go"))
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		fast, parseErr := parser.ParseFile(fset, file, nil, parser.AllErrors)
		if parseErr != nil {
			continue
		}
		ast.Inspect(fast, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Name == nil || !builders[fn.Name.Name] {
				return true
			}
			use := extractCobraUseField(fn)
			if use != "" {
				// Strip subverb-positional shape ("ac <milestone-id>" → "ac").
				first := strings.SplitN(use, " ", 2)[0]
				out[first] = fn.Name.Name
			}
			return false
		})
	}
	return out, nil
}

// extractCobraUseField walks a function body looking for a
// `&cobra.Command{Use: "<verb>"}` literal and returns the Use string.
// Returns "" if no Use field is set or the value is non-literal.
func extractCobraUseField(fn *ast.FuncDecl) string {
	var found string
	ast.Inspect(fn, func(n ast.Node) bool {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		// Only descend into cobra.Command literals.
		sel, ok := comp.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Command" {
			return true
		}
		for _, elt := range comp.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "Use" {
				continue
			}
			lit, ok := kv.Value.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			v, err := strconv.Unquote(lit.Value)
			if err == nil && found == "" {
				found = v
			}
		}
		return true
	})
	return found
}

// backtickedAiwfMention is one parsed “ `aiwf <verb> ...` “ reference.
type backtickedAiwfMention struct {
	verb string
}

// aiwfWordRE captures the first lowercase word following `aiwf` (with
// at least one space). Flag-shaped tokens (`--xxx`, `-v`) and tokens
// starting with non-letters never match — those aren't verb names.
var aiwfWordRE = regexp.MustCompile(`aiwf\s+([a-z][a-z-]*)`)

// backtickedAiwfMentions returns every “ `aiwf <verb>` “ reference
// found in code regions of body — both inline-code spans
// (single-backtick fenced) and fenced code blocks (triple-backtick).
// Prose mentions outside code regions are skipped: skill bodies use
// phrases like "aiwf is the framework" or "## What aiwf does"
// legitimately, and treating them as verb references would fire false
// positives on the policy.
func backtickedAiwfMentions(body string) []backtickedAiwfMention {
	var out []backtickedAiwfMention
	lines := strings.Split(body, "\n")
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			// Whole line is code; match anywhere on it.
			for _, m := range aiwfWordRE.FindAllStringSubmatch(line, -1) {
				out = append(out, backtickedAiwfMention{verb: m[1]})
			}
			continue
		}
		// Outside a fence: match only inside single-backtick spans on
		// this line. A span is delimited by a pair of unescaped
		// backticks; toggle on each backtick to track inside/outside.
		var span strings.Builder
		inSpan := false
		for _, r := range line {
			if r == '`' {
				if inSpan {
					for _, m := range aiwfWordRE.FindAllStringSubmatch(span.String(), -1) {
						out = append(out, backtickedAiwfMention{verb: m[1]})
					}
					span.Reset()
				}
				inSpan = !inSpan
				continue
			}
			if inSpan {
				span.WriteRune(r)
			}
		}
	}
	return out
}
