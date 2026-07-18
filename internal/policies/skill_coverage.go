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
// and PolicyConfigFieldsAreDiscoverable. Asserts five mechanically
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
//  4. Every runnable command reached through one or more namespace
//     parents (a "subverb", e.g. `milestone depends-on`, `contract
//     recipe show`) is either documented by a resolved `aiwf <full
//     path>` mention in some skill body, or carries its own
//     skillCoverageAllowlist entry keyed on the full space-joined
//     path. G-0284: earlier this axis didn't exist — a new subverb
//     could ship undocumented and the policy stayed green.
//  5. Every backticked `aiwf <verb> [<subverb>...]` mention inside a
//     skill body resolves to a registered command path. G-061's
//     repro — a shipped skill body referencing a non-existent verb
//     (`aiwf list contracts` before M-072 landed) — fails this check.
//     G-0284 extended resolution past the first token: `aiwf contract
//     bogus-subverb` now fails too, since `contract` is a namespace
//     parent and `bogus-subverb` isn't one of its children.
//
// Judgment-shaped decisions (when to use a per-verb skill vs. a
// topical multi-verb skill, when --help suffices) live in the
// companion ADR and CLAUDE.md `Skills policy` section. This policy
// captures only the mechanical companion.
func PolicySkillCoverageMatchesVerbs(root string) ([]Violation, error) {
	kernelSkills, err := loadEmbeddedSkillsForPolicy(root)
	if err != nil {
		return nil, err
	}
	pluginSkills, err := loadPluginSkillFixturesForPolicy(root)
	if err != nil {
		return nil, err
	}
	verbs, err := findAllVerbs(root)
	if err != nil {
		return nil, err
	}
	return runSkillCoverageChecks(kernelSkills, pluginSkills, verbs, skillCoverageAllowlist), nil
}

// runSkillCoverageChecks applies every skill-coverage invariant to the
// already-loaded inputs. Split from PolicySkillCoverageMatchesVerbs so
// negative-case unit tests can drive the checks with synthetic inputs
// without a tempdir fixture (CLAUDE.md §"Test untested code paths"
// — a positive-only test of the full policy proves nothing about
// whether the policy fires on real drift).
//
// kernelSkills: in-repo embedded skills at internal/skills/embedded/aiwf-*/.
// pluginSkills: embedded ritual plugin skills at internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-*/ (G-0182 consolidated this onto the embedded snapshot; previously a duplicated fixture under internal/policies/testdata/).
// Frontmatter invariants apply to both with the respective prefix.
// verbs carries every command path in the tree (top-level AND
// subverbs) keyed by its full space-joined path, e.g. "list",
// "milestone", "milestone depends-on". Verb-coverage (AC-3) is
// top-level-only and kernel-skill-only (plugin skills don't define
// top-level verbs); subverb-coverage (AC-4) and body-mention
// resolution (AC-5) both walk the full tree. G-0088 added plugin-
// skill coverage to this policy; G-0284 added the subverb axis.
func runSkillCoverageChecks(
	kernelSkills, pluginSkills []embeddedSkillEntry,
	verbs map[string]verbEntry,
	allowlist map[string]string,
) []Violation {
	var out []Violation
	out = append(out, checkSkillFrontmatter(kernelSkills, "aiwf-")...)
	out = append(out, checkSkillFrontmatter(pluginSkills, "aiwfx-")...)

	topVerbs := make(map[string]string, len(verbs))
	for path, e := range verbs {
		if !strings.Contains(path, " ") {
			topVerbs[path] = e.builder
		}
	}
	out = append(out, checkVerbCoverage(kernelSkills, topVerbs, allowlist)...)

	allSkills := make([]embeddedSkillEntry, 0, len(kernelSkills)+len(pluginSkills))
	allSkills = append(allSkills, kernelSkills...)
	allSkills = append(allSkills, pluginSkills...)
	out = append(out, checkSkillBodyMentionsResolve(allSkills, verbs)...)
	out = append(out, checkSubverbCoverage(verbs, collectResolvedMentionPaths(allSkills, verbs), allowlist)...)
	return out
}

// checkSkillFrontmatter enforces M-074 AC-2 and AC-3: every skill
// carries a non-empty `name:` matching its directory and the
// `<prefix><topic>` convention, and a non-empty `description:` (the
// host's match-scoring depends on it). prefix is the expected name-
// prefix (e.g. "aiwf-" for kernel-embedded skills, "aiwfx-" for plugin
// skill fixtures). G-0088 extended the policy to both surfaces.
func checkSkillFrontmatter(skills []embeddedSkillEntry, prefix string) []Violation {
	var out []Violation
	for _, s := range skills {
		switch {
		case s.frontmatterName == "":
			out = append(out, Violation{
				Policy: "skill-coverage",
				File:   s.relPath,
				Detail: "skill is missing a `name:` frontmatter field",
			})
		case s.frontmatterName != s.dirName:
			out = append(out, Violation{
				Policy: "skill-coverage",
				File:   s.relPath,
				Detail: fmt.Sprintf("skill `name: %s` does not match its directory %q", s.frontmatterName, s.dirName),
			})
		case !strings.HasPrefix(s.frontmatterName, prefix) || len(s.frontmatterName) <= len(prefix):
			out = append(out, Violation{
				Policy: "skill-coverage",
				File:   s.relPath,
				Detail: fmt.Sprintf("skill name %q does not match the `%s<topic>` convention", s.frontmatterName, prefix),
			})
		}
		if strings.TrimSpace(s.description) == "" {
			out = append(out, Violation{
				Policy: "skill-coverage",
				File:   s.relPath,
				Detail: "skill is missing a `description:` frontmatter field — the host's match-scoring depends on it",
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

// checkSubverbCoverage is G-0284's fix: every runnable command reached
// through one or more namespace parents (a path with a space in it,
// e.g. "milestone depends-on", "contract recipe show") must be
// documented somewhere. "Documented" means either a resolved `aiwf
// <full path>` mention in some skill body (documented, computed by
// collectResolvedMentionPaths) or an explicit skillCoverageAllowlist
// entry keyed on the full path. Non-runnable namespace parents (e.g.
// "contract recipe", itself just a grouping node with no behavior of
// its own) are not required to carry their own entry — what matters
// is that their runnable descendants resolve to real documentation.
func checkSubverbCoverage(
	verbs map[string]verbEntry,
	documented map[string]bool,
	allowlist map[string]string,
) []Violation {
	var out []Violation
	for path, e := range verbs {
		if !e.runnable || !strings.Contains(path, " ") {
			continue
		}
		if documented[path] {
			continue
		}
		if _, ok := allowlist[path]; ok {
			continue
		}
		out = append(out, Violation{
			Policy: "skill-coverage",
			File:   "cmd/aiwf/main.go",
			Detail: fmt.Sprintf("subverb %q has no resolved `aiwf %s` mention in any skill body and no entry in skillCoverageAllowlist — document it in a skill or allowlist it with a one-line rationale", path, path),
		})
	}
	return out
}

// checkSkillBodyMentionsResolve enforces M-074 AC-5 (extended by
// G-0284): every backticked `aiwf <verb> [<subverb>...]` mention
// inside a skill body resolves to a registered command path — walking
// as many tokens as the tree's namespace parents demand, not just the
// first.
func checkSkillBodyMentionsResolve(
	skills []embeddedSkillEntry,
	verbs map[string]verbEntry,
) []Violation {
	verbSet := verbSetWithBuiltins(verbs)
	var out []Violation
	for _, s := range skills {
		for _, m := range backtickedAiwfMentions(s.body, verbSet) {
			if !m.resolved {
				out = append(out, Violation{
					Policy: "skill-coverage",
					File:   s.relPath,
					Detail: fmt.Sprintf("skill body references `aiwf %s` but %q does not resolve to a registered command path (the inverse of the kernel principle that AI-discoverable functionality must resolve)", m.path, m.path),
				})
			}
		}
	}
	return out
}

// collectResolvedMentionPaths returns the set of full command paths
// that resolved successfully across every skill body — the "documented
// somewhere" evidence checkSubverbCoverage consumes.
func collectResolvedMentionPaths(skills []embeddedSkillEntry, verbs map[string]verbEntry) map[string]bool {
	verbSet := verbSetWithBuiltins(verbs)
	out := map[string]bool{}
	for _, s := range skills {
		for _, m := range backtickedAiwfMentions(s.body, verbSet) {
			if m.resolved {
				out[m.path] = true
			}
		}
	}
	return out
}

// verbSetWithBuiltins copies verbs and adds Cobra's auto-generated
// "help" and "completion" commands, which have no source-level
// AddCommand call for findAllVerbs to discover. Both are treated as
// runnable (leaf) so a mention stops cleanly after the first token
// instead of requiring their real Cobra-internal subcommands
// ("completion bash", etc.) to resolve too — out of scope here.
func verbSetWithBuiltins(verbs map[string]verbEntry) map[string]verbEntry {
	out := make(map[string]verbEntry, len(verbs)+2)
	for k, v := range verbs {
		out[k] = v
	}
	out["help"] = verbEntry{runnable: true}
	out["completion"] = verbEntry{runnable: true}
	return out
}

// skillCoverageAllowlist names every top-level Cobra verb, or runnable
// subverb (a full space-joined path, e.g. "milestone depends-on"),
// that ships without independent documentation — no same-named
// `aiwf-<verb>` embedded skill for a top-level verb, and no resolved
// `aiwf <full path>` mention in any skill body for a subverb. Each
// entry must carry a one-line rationale comment so a reviewer (human
// or AI) can see at a glance why this verb or subverb skips coverage.
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
	"cancel":      "convenience wrapper over promote (cancel = promote-to-terminal); aiwf-promote skill covers the lifecycle",
	"move":        "rare cross-epic milestone move; --help + the `aiwf-promote`/`aiwf-add` skills cover the surrounding flow",
	"rewidth":     "one-shot migration ritual per ADR-0008; --help is sufficient discovery surface (ADR-0006 'no skill when --help suffices' case)",
	"rename-area": "config-mutation verb; --help + area-member completion cover the surface; the orphan-trap warning lives in --help (E-0044, M-0177)",
	"set-area":    "single-entity area-tag mutation; --help + entity-id/area-member completion cover the surface; the orphan-trap framing lives in --help (E-0044, M-0183)",

	// Kind-namespace parent commands — non-Runnable; subverbs are documented elsewhere.
	"milestone": "kind-namespace parent (subverb `depends-on` for cross-milestone deps); the surrounding `aiwf-add` and `aiwf-promote` skills cover the flow",
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

// loadPluginSkillFixturesForPolicy reads every aiwfx-* SKILL.md under
// the embedded ritual snapshot at
// internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/. Per
// G-0182, the embedded snapshot is the canonical authoring location for
// ritual content (post-ADR-0014 distribution channel and pending
// ADR-0016 authoring-channel retirement); the per-AC content-assertion
// tests assert against the same bytes the binary embeds rather than a
// duplicated fixture under internal/policies/testdata/.
//
// Returns in directory-name-sorted order so violations surface
// deterministically. Filters strictly to `aiwfx-*` so the policy stays
// scoped to the plugin-skill authoring surface; the wf-* skills under
// the sibling wf-rituals plugin are a separate surface this policy
// does not check.
func loadPluginSkillFixturesForPolicy(root string) ([]embeddedSkillEntry, error) {
	ritualsRoot := filepath.Join(root, "internal", "skills", "embedded-rituals", "plugins", "aiwf-extensions", "skills")
	dirs, err := os.ReadDir(ritualsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []embeddedSkillEntry
	for _, d := range dirs {
		if !d.IsDir() || !strings.HasPrefix(d.Name(), "aiwfx-") {
			continue
		}
		path := filepath.Join(ritualsRoot, d.Name(), "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				// aiwfx-*/ without a SKILL.md isn't a plugin-skill
				// directory — skip silently.
				continue
			}
			return nil, err
		}
		entry := parseSkillMarkdown(data)
		entry.relPath = filepath.ToSlash(filepath.Join("internal", "skills", "embedded-rituals", "plugins", "aiwf-extensions", "skills", d.Name(), "SKILL.md"))
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

// verbEntry names a registered command's builder (for error messages)
// and whether the command is directly Runnable (carries its own
// Run/RunE) as opposed to a pure namespace parent that only groups
// children (e.g. "contract", "milestone", "contract recipe"). Some
// commands are both — Runnable AND have children (e.g. "add", whose
// bare form takes a `<kind>` positional and which also carries the
// "ac" subcommand; "render", whose bare form renders HTML and which
// also carries the "roadmap" subcommand). The runnable flag is what
// lets mention-resolution tell a genuine invalid subverb ("contract
// bogus-subverb") apart from a runnable command followed by an
// ordinary positional argument that happens to look like a subverb
// ("add milestone", "add gap") — see resolveAiwfMention.
type verbEntry struct {
	builder  string
	runnable bool
}

// findAllVerbs walks the full Cobra command tree reachable from
// NewRootCmd, recursing into every subcommand at every depth, and
// returns one entry per command keyed by its full space-joined path,
// e.g. "list", "milestone", "milestone depends-on", "contract recipe
// show". G-0284: earlier this walk (then named findTopLevelVerbs)
// stopped at the top level, so a namespace subverb had no coverage or
// resolution check at all.
//
// The walk: AST-parse internal/cli/root.go (the post-M-0118 home of
// NewRootCmd; cmd/aiwf/main.go is entry-only now), find NewRootCmd,
// collect every `cmd.AddCommand(...)` call, then recurse into each
// builder's own body for further `AddCommand` calls. Two builder
// shapes are recognized at the root:
//   - Ident form `newXCmd()` — a same-package helper, resolved against
//     root.go itself (e.g. `newVersionCmd`) and the legacy cmd/aiwf
//     dir (kept for symmetry; no verb uses that shape anymore).
//   - Selector form `pkg.NewCmd()` — the canonical per-verb subpackage
//     shape, where the builder lives in internal/cli/<pkg>/*.go.
//
// Below the root, every subcommand builder in this codebase is a
// same-package Ident call (contract's newBindCmd/newRecipeCmd and
// friends) — walkVerbSubtree only resolves that shape; see its doc
// comment for why the cross-package shape isn't handled there.
//
// For each builder, the relevant directory is walked for the FuncDecl
// definition; the `Use:` field's string literal in the
// `&cobra.Command{...}` is the verb name as the user types it, and the
// presence of a `Run:`/`RunE:` field marks the command Runnable.
func findAllVerbs(root string) (map[string]verbEntry, error) {
	fset := token.NewFileSet()
	rootCmdPath := filepath.Join(root, "internal", "cli", "root.go")
	rootAST, err := parser.ParseFile(fset, rootCmdPath, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	var newRootFn *ast.FuncDecl
	ast.Inspect(rootAST, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != "NewRootCmd" {
			return true
		}
		newRootFn = fn
		return false
	})
	if newRootFn == nil { //coverage:ignore internal/cli/root.go always defines NewRootCmd; this guards a malformed-repo edge the live tree can't exercise
		return nil, fmt.Errorf("NewRootCmd not found in %s", rootCmdPath)
	}

	cmdDir := filepath.Join(root, "cmd", "aiwf")
	cmdFiles, err := parseGoFilesInDir(fset, cmdDir)
	if err != nil { //coverage:ignore cmd/aiwf always exists in this repo; filepath.Glob only errors on a malformed pattern, not a missing dir
		return nil, err
	}
	// Root-level Ident builders may live in root.go itself (e.g.
	// newVersionCmd) or in the legacy cmd/aiwf dir; search both.
	rootIdentFiles := append([]*ast.File{rootAST}, cmdFiles...)

	out := map[string]verbEntry{}
	for _, fun := range addCommandArgs(newRootFn) {
		switch f := fun.(type) {
		case *ast.Ident:
			if child := findFuncDecl(rootIdentFiles, f.Name); child != nil {
				walkVerbSubtree(child, rootIdentFiles, "", f.Name, out)
			}
		case *ast.SelectorExpr:
			pkgIdent, ok := f.X.(*ast.Ident)
			if !ok { //coverage:ignore every `pkg.NewCmd(...)` call in root.go selects off an imported package Ident by construction
				continue
			}
			pkgFiles, ferr := parseGoFilesInDir(fset, filepath.Join(root, "internal", "cli", pkgIdent.Name))
			if ferr != nil { //coverage:ignore every internal/cli/<pkg> imported by root.go exists and its *.go glob can't error
				continue
			}
			if child := findFuncDecl(pkgFiles, f.Sel.Name); child != nil {
				walkVerbSubtree(child, pkgFiles, "", pkgIdent.Name+"."+f.Sel.Name, out)
			}
		}
	}
	return out, nil
}

// findTopLevelVerbs is the top-level-only view of findAllVerbs, kept
// for callers that only care about the root command set (the
// FSM-legality drift test in m0123_ac5_drift_test.go). Its output
// contract — keys are top-level verb names only, values are builder
// identity strings — is unchanged from before G-0284.
func findTopLevelVerbs(root string) (map[string]string, error) {
	all, err := findAllVerbs(root)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(all))
	for path, e := range all {
		if !strings.Contains(path, " ") {
			out[path] = e.builder
		}
	}
	return out, nil
}

// walkVerbSubtree records fn's own verb path (pathPrefix + fn's Use
// field's first token) into out, then recurses into every nested
// AddCommand call fn's body makes. files is the parsed set fn itself
// was found in, used to resolve children.
//
// Only the Ident (same-package) builder shape is handled below the
// root: every subcommand in this codebase's per-verb subpackages
// (contract's newBindCmd/newRecipeCmd, milestone's newDependsOnCmd,
// etc.) is wired that way — the cross-package `pkg.NewCmd()` shape
// only ever appears once, at the root (handled in findAllVerbs). If a
// subpackage ever nests a cross-package child, extend this the same
// way findAllVerbs' root loop does; until then that branch would be
// unexercised, untestable-except-synthetically dead code (YAGNI).
func walkVerbSubtree(fn *ast.FuncDecl, files []*ast.File, pathPrefix, builder string, out map[string]verbEntry) {
	fields := extractCobraCmdFields(fn)
	if fields.use == "" {
		return
	}
	seg := strings.SplitN(fields.use, " ", 2)[0]
	path := seg
	if pathPrefix != "" {
		path = pathPrefix + " " + seg
	}
	out[path] = verbEntry{builder: builder, runnable: fields.runnable}

	for _, fun := range addCommandArgs(fn) {
		ident, ok := fun.(*ast.Ident)
		if !ok { //coverage:ignore no subpackage below root nests a cross-package (*ast.SelectorExpr) child today; see the doc comment above
			continue
		}
		if child := findFuncDecl(files, ident.Name); child != nil {
			walkVerbSubtree(child, files, path, builder+"→"+ident.Name, out)
		}
	}
}

// addCommandArgs returns the callee expression (an *ast.Ident or
// *ast.SelectorExpr) of every `<recv>.AddCommand(builder(...))` call
// found anywhere inside fn's body. Shared by every recursion depth of
// findAllVerbs.
func addCommandArgs(fn *ast.FuncDecl) []ast.Expr {
	var out []ast.Expr
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
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
				// A raw composite literal (e.g. an inline hidden
				// compatibility alias) rather than a call to a named
				// builder — nothing to recurse into or discover.
				continue
			}
			out = append(out, inner.Fun)
		}
		return true
	})
	return out
}

// findFuncDecl returns the top-level FuncDecl named name across
// files, or nil if none matches.
func findFuncDecl(files []*ast.File, name string) *ast.FuncDecl {
	for _, f := range files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Name != nil && fn.Name.Name == name {
				return fn
			}
		}
	}
	return nil
}

// parseGoFilesInDir parses every *.go file in dir (including
// _test.go, matching the original single-level walk's leniency) and
// returns the successfully-parsed ASTs; a file that fails to parse is
// skipped rather than failing the whole walk.
func parseGoFilesInDir(fset *token.FileSet, dir string) ([]*ast.File, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil { //coverage:ignore the pattern is always the well-formed literal "*.go"; filepath.Glob only errors on a malformed pattern
		return nil, err
	}
	out := make([]*ast.File, 0, len(paths))
	for _, p := range paths {
		f, ferr := parser.ParseFile(fset, p, nil, parser.AllErrors)
		if ferr != nil {
			continue
		}
		out = append(out, f)
	}
	return out, nil
}

// cobraCmdFields is the subset of a `&cobra.Command{...}` composite
// literal findAllVerbs cares about.
type cobraCmdFields struct {
	use      string
	runnable bool
}

// extractCobraCmdFields walks fn's body for its own
// `&cobra.Command{...}` literal — the first one encountered, which by
// this codebase's `cmd := &cobra.Command{...}; ...; return cmd`
// convention is always the function's own command, never a nested
// literal passed to a child AddCommand call — and returns its `Use:`
// string plus whether it sets `Run:` or `RunE:`. Returns a zero value
// if no such literal is found.
func extractCobraCmdFields(fn *ast.FuncDecl) cobraCmdFields {
	var out cobraCmdFields
	var found bool
	ast.Inspect(fn, func(n ast.Node) bool {
		if found {
			return false
		}
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		sel, ok := comp.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Command" {
			return true
		}
		found = true
		for _, elt := range comp.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok { //coverage:ignore a struct composite literal's field key is always an *ast.Ident in valid, compiling Go
				continue
			}
			switch key.Name {
			case "Use":
				if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if v, err := strconv.Unquote(lit.Value); err == nil {
						out.use = v
					}
				}
			case "Run", "RunE":
				out.runnable = true
			}
		}
		return false
	})
	return out
}

// backtickedAiwfMention is one parsed “ `aiwf <path>` “ reference —
// either a fully-resolved command path, or the longest attempted
// prefix once resolution failed (the mismatched token included, for a
// readable violation message).
type backtickedAiwfMention struct {
	path     string
	resolved bool
}

// aiwfRunRE captures the whole run of consecutive lowercase-hyphen
// words immediately following `aiwf` (e.g. "aiwf contract recipe
// show" captures " contract recipe show"). Flag-shaped tokens
// (`--xxx`, `-v`), id-shaped tokens (`M-003`), and anything else that
// doesn't start with a lowercase letter end the run — those aren't
// verb/subverb names.
var aiwfRunRE = regexp.MustCompile(`aiwf((?:\s+[a-z][a-z-]*)+)`)

// resolveAiwfMention walks tokens against verbs, descending into
// namespace parents (non-runnable nodes) as far as the tokens allow.
// It stops cleanly — without flagging a violation — the moment it
// reaches a Runnable node, whether or not further tokens remain: a
// Runnable command may itself carry children (e.g. "add" also has
// "ac"; "render" also has "roadmap"), so the walk still tries the
// next token against those children, but if it doesn't match, that's
// just an ordinary positional argument ("add milestone", "add gap"),
// not an invalid subverb — the caller asked for `<kind>`, not for a
// subcommand. A mismatch is only reported as unresolved when the node
// it happened at is a non-runnable namespace parent, since there the
// grammar requires a real child (`contract bogus-subverb` is invalid;
// `contract` demands a subverb).
func resolveAiwfMention(tokens []string, verbs map[string]verbEntry) (path string, resolved bool) {
	var matched []string
	for _, tok := range tokens {
		next := make([]string, len(matched)+1)
		copy(next, matched)
		next[len(matched)] = tok
		candidate := strings.Join(next, " ")

		if _, found := verbs[candidate]; !found {
			current := strings.Join(matched, " ")
			if current == "" {
				return candidate, false
			}
			if !verbs[current].runnable {
				return candidate, false
			}
			return current, true
		}
		matched = next
	}
	return strings.Join(matched, " "), true
}

// backtickedAiwfMentions returns every “ `aiwf <path>` “ reference
// found in code regions of body — both inline-code spans
// (single-backtick fenced) and fenced code blocks (triple-backtick).
// Prose mentions outside code regions are skipped: skill bodies use
// phrases like "aiwf is the framework" or "## What aiwf does"
// legitimately, and treating them as verb references would fire false
// positives on the policy.
func backtickedAiwfMentions(body string, verbs map[string]verbEntry) []backtickedAiwfMention {
	var out []backtickedAiwfMention
	collect := func(text string) {
		for _, m := range aiwfRunRE.FindAllStringSubmatch(text, -1) {
			tokens := strings.Fields(m[1])
			path, resolved := resolveAiwfMention(tokens, verbs)
			out = append(out, backtickedAiwfMention{path: path, resolved: resolved})
		}
	}

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
			collect(line)
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
					collect(span.String())
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
