package policies

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// lycheeConfigPath is the link checker's config, relative to the repo
// root. `.github/workflows/link-check.yml` passes it to lychee verbatim.
const lycheeConfigPath = ".lychee.toml"

// archivalExemptPrefix is the one docs subtree that may sit in
// exclude_path while still linking into work/. CLAUDE.md's documentation
// hierarchy puts docs/archive/ in the Archival tier, whose links are
// forget-by-default: a cross-reference there is a frozen snapshot and is
// not held to keep resolving.
const archivalExemptPrefix = "docs/archive"

// TestM0317_AC1_DocsLinkingIntoWorkAreNotShadowedByLycheeExcludes is the
// mechanical evidence for M-0317/AC-1.
//
// The measurement recorded under that AC found that link-check — lychee
// over `./**/*.md` — does report a docs-to-work link broken by a mover,
// and that `exclude_path` filters lychee's *input files* rather than its
// link targets, so `work` appearing in that list does not blind it. The
// finding therefore rests on a coverage claim: every docs file that links
// into work/ is a file lychee actually reads.
//
// That claim is what rots silently. Adding a docs subtree to
// exclude_path takes its links out of the checked set with no other
// symptom — link-check keeps passing, because the links stop being read
// rather than starting to resolve. So the check runs the other way round:
// walk what exclude_path already shadows and fail if any of it links into
// work/.
//
// A relationship check rather than a phrase assertion: the expectation is
// derived by parsing the real `.lychee.toml`, so a config edit moves the
// expectation with it, and no wording in either file is pinned.
//
// Retires when G-0478's detection half lands a link-integrity rule inside
// `aiwf check`. At that point the guarantee is a kernel finding rather
// than a CI-only gate, and this test is what that change deletes.
func TestM0317_AC1_DocsLinkingIntoWorkAreNotShadowedByLycheeExcludes(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	excludes := lycheeExcludePaths(t, filepath.Join(root, lycheeConfigPath))

	// A parse that silently found nothing would make every assertion below
	// vacuous, so an empty result is a failure rather than a clean run.
	if len(excludes) == 0 {
		t.Fatalf("%s: parsed no exclude_path entries — the config's shape changed and this check has stopped measuring anything", lycheeConfigPath)
	}

	var shadowed []string
	for _, ex := range excludes {
		if !strings.HasPrefix(ex, "docs/") && ex != "docs" {
			// Only a docs prefix can shadow the delegated class. `work` is
			// in this list too, and deliberately: it stops lychee reading
			// entity bodies, whose links the movers already repair.
			continue
		}
		if ex == archivalExemptPrefix || strings.HasPrefix(ex, archivalExemptPrefix+"/") {
			continue
		}
		shadowed = append(shadowed, ex)
	}

	for _, ex := range shadowed {
		for _, f := range docsFilesLinkingIntoWork(t, root, ex) {
			t.Errorf("%s links into work/ but sits under exclude_path %q, so lychee never reads it — the delegated class is no longer fully covered", f, ex)
		}
	}
}

// residualOwningGaps are the gaps that own the residual ADR-0033 declines
// to cover mechanically — links from non-entity files into the entity
// tree. M-0317/AC-2 routes the delegation measurement to them.
var residualOwningGaps = []string{"G-0478", "G-0439"}

// TestM0317_AC2_ADR0033NamesTheGapsOwningItsResidual is the mechanical
// evidence for M-0317/AC-2.
//
// The measurement's consequence for ADR-0033 is that its second bullet
// named a delegate carrying no mechanical trigger, while the check that
// does cover the class went unmentioned. Correcting the prose is not what
// makes the finding survive: what does is that a reader of the decision
// can reach the gaps that own the residual, so the next person to ask
// "what covers non-entity narrative?" lands on the measurement instead of
// re-deriving it.
//
// A relationship check rather than a phrase assertion: it compares the
// ADR against the tree, so deleting either gap or dropping its citation
// turns it red, while rewording any of the three does not. It is also not
// redundant with `body-prose-id`, which fires on a *dangling* id and
// never on a *missing* one — the citation going absent is exactly the
// failure that rule cannot see.
//
// Retires when both gaps reach a terminal status: the residual then has
// an answer rather than an owner, and the citation becomes history.
func TestM0317_AC2_ADR0033NamesTheGapsOwningItsResidual(t *testing.T) {
	t.Parallel()

	root, tr := sharedRepoTree(t)

	adr := tr.ByID("ADR-0033")
	if adr == nil {
		t.Fatal("ADR-0033 does not resolve via tr.ByID")
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(adr.Path)))
	if err != nil {
		t.Fatalf("reading ADR-0033 at %s: %v", adr.Path, err)
	}
	refs := sectionBody(string(raw), "\n## References")
	if refs == "" {
		t.Fatal("ADR-0033 has no `## References` section to carry the citations")
	}

	for _, id := range residualOwningGaps {
		if tr.ByID(id) == nil {
			t.Errorf("%s does not resolve via tr.ByID — ADR-0033's residual has no owning gap to reach", id)
			continue
		}
		if !strings.Contains(refs, id) {
			t.Errorf("ADR-0033's `## References` does not name %s, so the gap owning its non-entity residual is not reachable from the decision:\n%s", id, refs)
		}
	}
}

// lycheeExcludePaths reads lychee's config and returns its exclude_path
// entries.
func lycheeExcludePaths(t *testing.T, configPath string) []string {
	t.Helper()

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading %s: %v", configPath, err)
	}
	return parseLycheeExcludePaths(string(raw))
}

// lycheeExcludeEntry matches one exclude_path element on a line of its
// own: a double-quoted literal, optionally comma-terminated. Anything
// else on the line — a single-quoted literal, a trailing comment, two
// entries sharing a line — fails to match, which is what makes the parser
// refuse rather than mangle.
var lycheeExcludeEntry = regexp.MustCompile(`^"([^"]*)",?$`)

// parseLycheeExcludePaths extracts the `exclude_path` array from lychee's
// TOML config. Hand-parsed rather than via a TOML dependency: the array
// is a flat list of quoted literals, and the repo carries no TOML
// decoder.
//
// It reads exactly one TOML spelling — a bracket on the key's line, then
// one quoted literal per line — and returns nil for every other shape,
// including ones TOML accepts and lychee honours. That asymmetry is
// deliberate and is the whole reason the parser can be trusted: a
// permissive parser that recovers a partial list from an unfamiliar
// spelling would hand the caller a short list indistinguishable from a
// genuinely short one, and the caller would pass. Returning nil routes
// every unreadable shape to the caller's emptiness check, which fails.
//
// The failure mode this closes is not hypothetical: reformatting the
// committed array onto one line is a semantic no-op that lychee honours
// and that a lenient parser reads as a single bogus entry.
func parseLycheeExcludePaths(raw string) []string {
	_, after, found := strings.Cut(raw, "exclude_path = [")
	if !found {
		return nil
	}
	body, _, found := strings.Cut(after, "]")
	if !found {
		return nil
	}

	// An inline array carries its entries on the key's own line, where
	// the per-line scan below would never see them.
	head, rest, found := strings.Cut(body, "\n")
	if !found || strings.TrimSpace(head) != "" {
		return nil
	}

	var out []string
	for _, line := range strings.Split(rest, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := lycheeExcludeEntry.FindStringSubmatch(line)
		if m == nil {
			return nil
		}
		out = append(out, m[1])
	}
	return out
}

// docsFilesLinkingIntoWork returns the repo-relative paths of markdown
// files under prefix carrying at least one markdown link whose
// destination resolves under work/.
func docsFilesLinkingIntoWork(t *testing.T, root, prefix string) []string {
	t.Helper()

	dir := filepath.Join(root, filepath.FromSlash(prefix))
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// An exclude_path entry naming a directory that does not exist
		// shadows nothing.
		return nil
	}

	var out []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if linksIntoWork(string(body), rel) {
			out = append(out, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", prefix, err)
	}
	return out
}

// linksIntoWork reports whether body — the markdown of the file at
// repo-relative linkingFile — carries a link destination resolving under
// work/. Fenced blocks and inline-code spans are skipped (the latter via
// the shared stripInlineCode): a link shape inside one is prose about
// links, and lychee does not check it either.
//
// A destination is resolved against the linking file's own directory,
// except one already rooted at work/, which names a path from the repo
// root. Nothing else needs discriminating here: a scheme-bearing
// destination cannot resolve under work/ once joined to a docs directory,
// and an `#anchor` or `?query` suffix rides along without changing which
// prefix the result carries. The predicate is a prefix test rather than a
// file lookup, so neither shape needs its own arm.
func linksIntoWork(body, linkingFile string) bool {
	dir := path.Dir(linkingFile)
	fenced := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		for _, m := range markdownLinkRegex.FindAllStringSubmatch(stripInlineCode(line), -1) {
			dest := m[1]
			resolved := path.Clean(path.Join(dir, dest))
			if strings.HasPrefix(dest, "work/") {
				resolved = path.Clean(dest)
			}
			if resolved == "work" || strings.HasPrefix(resolved, "work/") {
				return true
			}
		}
	}
	return false
}

// TestM0317_LinksIntoWork covers each arm of the destination scan
// directly. Driving it only through the live docs tree would leave the
// discriminating arms — the code-span skips above all — exercised by
// whatever those files happen to contain rather than on purpose.
func TestM0317_LinksIntoWork(t *testing.T) {
	t.Parallel()

	const from = "docs/initiatives/example.md"

	tests := []struct {
		name        string
		linkingFile string
		body        string
		want        bool
	}{
		{
			name: "relative destination into work",
			body: "see [G-0311](../../work/gaps/G-0311-slug.md) for detail",
			want: true,
		},
		{
			name: "root-relative destination into work",
			body: "see [G-0311](work/gaps/G-0311-slug.md) for detail",
			want: true,
		},
		{
			name: "anchor suffix is split before resolving",
			body: "see [G-0311](../../work/gaps/G-0311-slug.md#why-it-matters)",
			want: true,
		},
		{
			name: "query suffix is split before resolving",
			body: "see [G-0311](../../work/gaps/G-0311-slug.md?plain=1)",
			want: true,
		},
		{
			name: "destination staying inside docs is not a work link",
			body: "see [the design](../design/design-decisions.md)",
			want: false,
		},
		{
			name: "url carrying work/ in its path is not a repo link",
			body: "see [upstream](https://example.com/work/gaps/G-0311-slug.md)",
			want: false,
		},
		{
			name: "empty destination resolves to nothing",
			body: "see [nothing]() here",
			want: false,
		},
		{
			name: "link shape inside an inline-code span is prose about links",
			body: "write it as `[G-0311](../../work/gaps/G-0311-slug.md)` in the body",
			want: false,
		},
		{
			name: "link shape inside a fenced block is an example",
			body: "before\n```markdown\n[G-0311](../../work/gaps/G-0311-slug.md)\n```\nafter",
			want: false,
		},
		{
			name: "a live link after a closed fence still counts",
			body: "```\ncode\n```\nsee [G-0311](../../work/gaps/G-0311-slug.md)",
			want: true,
		},
		{
			name:        "depth is resolved against the linking file's own directory",
			linkingFile: "docs/example.md",
			body:        "see [G-0311](../../work/gaps/G-0311-slug.md)",
			want:        false,
		},
		{
			name: "destination naming the work directory itself",
			body: "see [the tree](../../work)",
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			linking := tc.linkingFile
			if linking == "" {
				linking = from
			}
			if got := linksIntoWork(tc.body, linking); got != tc.want {
				t.Errorf("linksIntoWork(_, %q) = %v, want %v\nbody:\n%s", linking, got, tc.want, tc.body)
			}
		})
	}
}

// TestM0317_DocsFilesLinkingIntoWork drives the walk over a fixture
// tree.
//
// Against the committed repo the walk returns nothing — every prefix
// `exclude_path` shadows today is either work-link-free or the exempt
// archival one — so the arm that reports a violation is not reachable
// from the live-tree test above. A guard whose firing path only ever runs
// under a hand-applied mutation is pinned by nobody once the mutation is
// reverted, which is precisely how a check comes to pass for a reason
// unrelated to what it claims.
func TestM0317_DocsFilesLinkingIntoWork(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", rel, err)
		}
	}

	// Each destination carries the `../` depth its own file needs, so a
	// hit depends on the walk resolving against the linking file rather
	// than against the prefix.
	write("sub/withlink.md", "see [G-0311](../work/gaps/G-0311-slug.md)\n")
	write("sub/nested/deep.md", "see [G-0311](../../work/gaps/G-0311-slug.md)\n")
	write("sub/nolink.md", "see [the design](../design/design-decisions.md)\n")
	write("sub/notmarkdown.txt", "see [G-0311](../work/gaps/G-0311-slug.md)\n")
	write("other/withlink.md", "see [G-0311](../work/gaps/G-0311-slug.md)\n")

	got := docsFilesLinkingIntoWork(t, root, "sub")
	want := []string{"sub/nested/deep.md", "sub/withlink.md"}
	sort.Strings(got)
	if !slices.Equal(got, want) {
		t.Errorf("docsFilesLinkingIntoWork(_, %q) = %q, want %q", "sub", got, want)
	}

	// A prefix naming no directory shadows nothing, which is a clean
	// result rather than an error: `.lychee.toml` carries an entry for a
	// tree that has since moved.
	if got := docsFilesLinkingIntoWork(t, root, "absent"); got != nil {
		t.Errorf("docsFilesLinkingIntoWork(_, %q) = %q, want nil for a missing directory", "absent", got)
	}
}

// TestM0317_ParseLycheeExcludePaths covers the parser's arms, including
// the two that yield an empty result. Those matter because the caller
// reads empty as "the config's shape changed", so a parser that silently
// returned nothing for a readable config would disarm the check.
func TestM0317_ParseLycheeExcludePaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "entries, comments and blank lines",
			raw:  "exclude_path = [\n  \"node_modules\",\n\n  # a comment inside the block\n  \"work\",\n]\n",
			want: []string{"node_modules", "work"},
		},
		{
			name: "no exclude_path key at all",
			raw:  "accept = [200]\n",
			want: nil,
		},
		{
			name: "unterminated array",
			raw:  "exclude_path = [\n  \"work\",\n",
			want: nil,
		},
		{
			name: "empty array",
			raw:  "exclude_path = [\n]\n",
			want: nil,
		},
		// The three shapes below are valid TOML that lychee honours. Each
		// one defeated a lenient earlier parser: the inline array read as
		// a single bogus entry, and the other two kept punctuation that
		// made the docs/ prefix test miss. Refusing beats half-reading,
		// because the caller can only detect the refusal.
		{
			name: "inline array on the key's own line",
			raw:  "exclude_path = [\"node_modules\", \"docs/research\", \"work\"]\n",
			want: nil,
		},
		{
			name: "single-quoted literal",
			raw:  "exclude_path = [\n  'docs/research',\n]\n",
			want: nil,
		},
		{
			name: "trailing comment sharing an entry's line",
			raw:  "exclude_path = [\n  \"docs/research\", # narrative\n]\n",
			want: nil,
		},
		{
			name: "two entries sharing one line",
			raw:  "exclude_path = [\n  \"node_modules\", \"work\",\n]\n",
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseLycheeExcludePaths(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("parseLycheeExcludePaths() = %q, want %q", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("entry %d = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
