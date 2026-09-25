package policies

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/initrepo"
	"github.com/23min/aiwf/internal/pathutil"
	"github.com/23min/aiwf/internal/projectguidance"
)

// fenceHostFile renders a fictional host entry point: handwritten text,
// then the aiwf:guidance block carrying guidanceBody, then the routing
// block carrying routeBody. The markers come from their owning packages,
// so a marker change there moves these fixtures with it.
func fenceHostFile(handwritten, guidanceBody, routeBody string) string {
	gStart, gEnd, _ := initrepo.GuidanceMarkers()
	rStart, rEnd, _ := projectguidance.RouteMarkers()
	return handwritten + "\n" +
		gStart + "\n" + guidanceBody + "\n" + gEnd + "\n\n" +
		rStart + "\n" + routeBody + "\n" + rEnd + "\n"
}

// Fixture paths. docs/dev/testing.md is routed from .guidance/project.md
// and so is an on-demand document; docs/other.md is not routed and so is
// outside the guidance set.
const (
	fenceRoutedDoc   = "docs/dev/testing.md"
	fenceUnroutedDoc = "docs/other.md"
	fenceCodeFile    = "internal/app/app.go"
	fenceFragmentSrc = "internal/skills/embedded-guidance/aiwf-guidance.md"
	fencePack        = ".guidance/packs/go/guide.md"
	fenceProjectDoc  = ".guidance/project.md"
	fenceClaudeText  = "# Dev rules\n\nKeep it simple.\n"
	fenceAgentsText  = "# Codex rules\n\nRead CLAUDE.md in full.\n"
)

func fenceOwnedJSON(packDigest string) string {
	return "{\n" +
		`  ".guidance/index.md": "aa",` + "\n" +
		`  ".guidance/packs/go/guide.md": "` + packDigest + `",` + "\n" +
		`  "AGENTS.md": "cc",` + "\n" +
		`  "CLAUDE.md": "cc"` + "\n}\n"
}

// guidanceFenceFixture seeds a throwaway repo carrying the whole guidance
// set, its owned outputs, its source, configuration, an unrelated code
// file and a resolvable entity, and returns the root, a git runner, a file
// writer, and the seeded commit as the audit base.
func guidanceFenceFixture(t *testing.T) (root string, runGit func(...string) string, writeFile func(string, string), baseSHA string) {
	t.Helper()
	root, runGit, writeFile, _ = skillFixtureBase(t)
	writeFile(provFixtureEntityAt, provFixtureEntity)
	writeFile("CLAUDE.md", fenceHostFile(fenceClaudeText, "@.claude/aiwf-guidance.md", "route v1"))
	writeFile("AGENTS.md", fenceHostFile(fenceAgentsText, "fragment v1", "route v1"))
	writeFile(fenceProjectDoc, "# Project guidance\n\nFor tests read [testing](../"+fenceRoutedDoc+").\n")
	writeFile(fenceRoutedDoc, "# Testing\n\nRun the suite.\n")
	writeFile(fenceUnroutedDoc, "# Other\n\nNot routed.\n")
	writeFile(".guidance/.aiwf-owned", fenceOwnedJSON("bb"))
	writeFile(".guidance/index.md", "# Index\n")
	writeFile(fencePack, "# Go pack\n")
	writeFile(fenceFragmentSrc, "fragment v1\n")
	writeFile("aiwf.yaml", "hosts: [claude-code, codex]\n")
	writeFile(fenceCodeFile, "package app\n")
	runGit("add", "-A")
	runGit("commit", "-m", "seed the guidance fixture")
	return root, runGit, writeFile, trimLine(runGit("rev-parse", "HEAD"))
}

// commitOwned commits everything staged-or-not under a resolving entity
// trailer, so only the arm a case exercises can fire.
func commitOwned(runGit func(...string) string, subject string) {
	runGit("add", "-A")
	runGit("commit", "-m", subject, "--trailer", "aiwf-entity: "+provFixtureEntityID)
}

func fenceViolationFiles(t *testing.T, root, base string) []string {
	t.Helper()
	vs, err := guidanceFenceViolations(root, base)
	if err != nil {
		t.Fatalf("guidanceFenceViolations: %v", err)
	}
	for _, v := range vs {
		if v.Policy != "guidance-fence" {
			t.Errorf("violation Policy = %q, want guidance-fence", v.Policy)
		}
	}
	return violationFiles(vs)
}

// TestGuidanceFence_CommitScope is M-0333 AC-1: a commit changing
// handwritten guidance may carry related guidance sources, configuration
// and generated outputs, and nothing else. Every case adds text only and
// names a resolving entity, so the trailer and removal arms stay silent.
func TestGuidanceFence_CommitScope(t *testing.T) {
	t.Parallel()

	addToClaude := func(w func(string, string)) {
		w("CLAUDE.md", fenceHostFile(fenceClaudeText+"\nA new rule.\n", "@.claude/aiwf-guidance.md", "route v1"))
	}
	sourcePlusOutputs := func(w func(string, string)) {
		addToClaude(w)
		w(fenceFragmentSrc, "fragment v1\nfragment addition\n")
		w("AGENTS.md", fenceHostFile(fenceAgentsText, "fragment v1\nfragment addition", "route v1"))
		w("aiwf.yaml", "hosts: [claude-code, codex]\nguidance:\n  packs: [go]\n")
		w(fencePack, "# Go pack\n\nMore.\n")
		w(".guidance/.aiwf-owned", fenceOwnedJSON("dd"))
	}
	touchCode := func(w func(string, string)) { w(fenceCodeFile, "package app\n\nvar x = 1\n") }

	tests := []struct {
		name   string
		change func(w func(string, string), git func(...string) string)
		body   string
		want   []string
	}{
		{
			name:   "a handwritten CLAUDE.md edit beside a code edit refuses the code file",
			change: func(w func(string, string), _ func(...string) string) { addToClaude(w); touchCode(w) },
			want:   []string{fenceCodeFile},
		},
		{
			name:   "a guidance source with its rendered outputs and configuration passes",
			change: func(w func(string, string), _ func(...string) string) { sourcePlusOutputs(w) },
		},
		{
			name:   "an unrelated file hidden beside a source-plus-output update is refused alone",
			change: func(w func(string, string), _ func(...string) string) { sourcePlusOutputs(w); touchCode(w) },
			want:   []string{fenceCodeFile},
		},
		{
			name: "a handwritten AGENTS.md edit beside a code edit refuses the code file",
			change: func(w func(string, string), _ func(...string) string) {
				w("AGENTS.md", fenceHostFile(fenceAgentsText+"\nAnother rule.\n", "fragment v1", "route v1"))
				touchCode(w)
			},
			want: []string{fenceCodeFile},
		},
		{
			name: "a project router edit beside a code edit refuses the code file",
			change: func(w func(string, string), _ func(...string) string) {
				w(fenceProjectDoc, "# Project guidance\n\nFor tests read [testing](../"+fenceRoutedDoc+").\n\nMore.\n")
				touchCode(w)
			},
			want: []string{fenceCodeFile},
		},
		{
			name: "a routed on-demand document edit beside a code edit refuses the code file",
			change: func(w func(string, string), _ func(...string) string) {
				w(fenceRoutedDoc, "# Testing\n\nRun the suite.\n\nAnd the race detector.\n")
				touchCode(w)
			},
			want: []string{fenceCodeFile},
		},
		{
			name: "an unrouted document beside a code edit is not guidance",
			change: func(w func(string, string), _ func(...string) string) {
				w(fenceUnroutedDoc, "# Other\n\nNot routed.\n\nStill not.\n")
				touchCode(w)
			},
		},
		{
			name: "a change confined to a managed block is judged through its source",
			change: func(w func(string, string), _ func(...string) string) {
				w(fenceFragmentSrc, "fragment v2\n")
				w("AGENTS.md", fenceHostFile(fenceAgentsText, "fragment v2", "route v1"))
				touchCode(w)
			},
		},
		{
			name: "a file under .guidance/ outside the owned record is not exempt",
			change: func(w func(string, string), _ func(...string) string) {
				addToClaude(w)
				w(".guidance/notes.md", "# Notes\n")
			},
			want: []string{".guidance/notes.md"},
		},
		{
			name: "renaming a routed document with its route passes",
			change: func(w func(string, string), git func(...string) string) {
				git("mv", fenceRoutedDoc, "docs/dev/tests.md")
				w(fenceProjectDoc, "# Project guidance\n\nFor tests read [testing](../docs/dev/tests.md).\n")
			},
			// The route line is reworded, so the commit records where it went.
			body: "Removed: the old route\nDisposition: relocated to docs/dev/tests.md",
		},
		{
			name: "renaming a routed document beside a code edit refuses the code file",
			change: func(w func(string, string), git func(...string) string) {
				git("mv", fenceRoutedDoc, "docs/dev/tests.md")
				touchCode(w)
			},
			want: []string{fenceCodeFile},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, runGit, writeFile, base := guidanceFenceFixture(t)
			tt.change(writeFile, runGit)
			commitOwned(runGit, "docs(guidance): change\n\n"+tt.body)
			if got := fenceViolationFiles(t, root, base); !equalStrings(got, tt.want) {
				t.Errorf("violation files = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGuidanceFence_MergeCommits is M-0333 AC-1's merge edge: a merge
// adds no judgment of its own, so a merge of compliant commits is silent,
// and a merge cannot hide a non-compliant commit it brings in, since that
// commit is in the range on its own.
func TestGuidanceFence_MergeCommits(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		mixed bool
		want  []string
	}{
		{name: "a merge of separate guidance and code commits is silent"},
		{name: "a merge cannot hide a mixed commit it brings in", mixed: true, want: []string{fenceCodeFile}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, runGit, writeFile, base := guidanceFenceFixture(t)
			trunk := trimLine(runGit("rev-parse", "--abbrev-ref", "HEAD"))
			runGit("checkout", "-b", "side")
			writeFile("CLAUDE.md", fenceHostFile(fenceClaudeText+"\nA new rule.\n", "@.claude/aiwf-guidance.md", "route v1"))
			if !tc.mixed {
				commitOwned(runGit, "docs(guidance): rule")
			}
			writeFile(fenceCodeFile, "package app\n\nvar x = 1\n")
			commitOwned(runGit, "feat(app): code")
			runGit("checkout", trunk)
			writeFile("README.md", "base\ntrunk moved\n")
			commitOwned(runGit, "docs: trunk moves")
			runGit("merge", "--no-ff", "-m", "Merge side", "side")

			if got := fenceViolationFiles(t, root, base); !equalStrings(got, tc.want) {
				t.Errorf("violation files = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGuidanceFence_DeletionAndBase covers the paths a deletion takes and
// the gate's no-comparison-point arms: an empty or all-zero base no-ops,
// and a base naming no commit is an error rather than a silent pass.
func TestGuidanceFence_DeletionAndBase(t *testing.T) {
	t.Parallel()

	t.Run("deleting a routed document beside a code edit refuses the code file", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, base := guidanceFenceFixture(t)
		runGit("rm", "-q", fenceRoutedDoc)
		writeFile(fenceCodeFile, "package app\n\nvar x = 1\n")
		commitOwned(runGit, "docs(guidance): drop a document\n\nRemoved: the testing document\nDisposition: deleted")
		if got, want := fenceViolationFiles(t, root, base), []string{fenceCodeFile}; !equalStrings(got, want) {
			t.Errorf("violation files = %v, want %v", got, want)
		}
	})

	t.Run("guidance added with code, where the parent had none, refuses the code", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, base := skillFixtureBase(t)
		writeFile(provFixtureEntityAt, provFixtureEntity)
		writeFile(fenceProjectDoc, "# Project guidance\n\nFor tests read [testing](../"+fenceRoutedDoc+").\n")
		writeFile(fenceRoutedDoc, "# Testing\n")
		writeFile(fenceCodeFile, "package app\n")
		commitOwned(runGit, "docs(guidance): start the router")
		if got, want := fenceViolationFiles(t, root, base), []string{fenceCodeFile, provFixtureEntityAt}; !equalStrings(got, want) {
			t.Errorf("violation files = %v, want %v", got, want)
		}
	})

	t.Run("a root commit is judged against an empty tree", func(t *testing.T) {
		t.Parallel()
		// A base on an unrelated history leaves HEAD's root commit in the
		// range, and a root commit has no parent to read.
		root := t.TempDir()
		runGit, writeFile := repoGitRunner(t, root), repoFileWriter(t, root)
		runGit("init", "-q")
		// The caller's config must not decide whether a root commit shows
		// a diff.
		runGit("config", "log.showRoot", "false")
		writeFile(fenceProjectDoc, "# Project guidance\n")
		writeFile(fenceCodeFile, "package app\n")
		runGit("add", "-A")
		runGit("commit", "-q", "-m", "root")
		trunk := trimLine(runGit("rev-parse", "--abbrev-ref", "HEAD"))
		runGit("checkout", "-q", "--orphan", "unrelated")
		runGit("rm", "-rq", "--cached", ".")
		runGit("commit", "-q", "--allow-empty", "-m", "unrelated root")
		base := trimLine(runGit("rev-parse", "HEAD"))
		runGit("checkout", "-q", "-f", trunk)
		// The root commit also names no entity, so the trailer arm fires
		// on the guidance file beside the scope arm on the code file.
		if got, want := fenceViolationFiles(t, root, base), []string{fenceProjectDoc, fenceCodeFile}; !equalStrings(got, want) {
			t.Errorf("violation files = %v, want %v", got, want)
		}
	})

	t.Run("the entry point no-ops without a base in the environment", func(t *testing.T) {
		t.Parallel()
		if os.Getenv("AIWF_COVERAGE_BASE") != "" {
			t.Skip("a base is set; TestPolicy_GuidanceFence covers the entry point")
		}
		vs, err := PolicyGuidanceFence(t.TempDir())
		if err != nil || len(vs) != 0 {
			t.Errorf("PolicyGuidanceFence without a base = %v, %v; want none", vs, err)
		}
	})

	t.Run("no comparison point no-ops", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, _ := guidanceFenceFixture(t)
		writeFile("CLAUDE.md", "handwritten only\n")
		writeFile(fenceCodeFile, "package app\n\nvar x = 1\n")
		commitOwned(runGit, "mixed")
		for _, base := range []string{"", "  ", zeroSHA} {
			if got := fenceViolationFiles(t, root, base); len(got) != 0 {
				t.Errorf("base %q: violation files = %v, want none", base, got)
			}
		}
	})

	t.Run("an empty root is an error, not a silent pass", func(t *testing.T) {
		t.Parallel()
		if _, err := guidanceFenceViolations("", "HEAD"); err == nil {
			t.Fatal("want an error for an empty root, got nil")
		}
	})

	t.Run("a base naming no commit is an error", func(t *testing.T) {
		t.Parallel()
		root, _, _, _ := guidanceFenceFixture(t)
		if _, err := guidanceFenceViolations(root, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"); err == nil {
			t.Fatal("want an error for a base naming no commit, got nil")
		}
	})
}

// TestParseFenceLog pins how the NUL-framed log becomes commits: each
// field is read by position, so a rename keeps both paths, an addition
// has no old path, a deletion no new one, and a path or message shaped
// like a header or a status is never taken for one.
func TestParseFenceLog(t *testing.T) {
	t.Parallel()
	sha1, sha2, sha3 := strings.Repeat("a", 40), strings.Repeat("b", 40), strings.Repeat("c", 40)
	out := sha1 + " " + sha2 + "\x00E-0001\nE-0002\x00subject\n\nM\n" + sha3 + "\n\x00\x00\n" +
		"R087\x00docs/a.md\x00docs/b.md\x00A\x00" + sha3 + "\x00D\x00gone.md\x00M\x00M\x00" +
		sha2 + "\x00\x00root\n\x00\x00\nA\x00first.md\x00"
	got, err := parseFenceLog(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d records %+v, want 2", len(got), got)
	}
	first := got[0]
	if first.sha != sha1 || first.parent != sha2 || first.entity != "E-0001" || !strings.Contains(first.body, sha3) {
		t.Errorf("first header = %+v", first)
	}
	want := []fenceChange{{Old: "docs/a.md", New: "docs/b.md"}, {New: sha3}, {Old: "gone.md"}, {Old: "M", New: "M"}}
	if len(first.changes) != len(want) {
		t.Fatalf("first changes = %+v, want %+v", first.changes, want)
	}
	for i := range want {
		if first.changes[i] != want[i] {
			t.Errorf("change %d = %+v, want %+v", i, first.changes[i], want[i])
		}
	}
	if second := got[1]; second.sha != sha2 || second.parent != "" || len(second.changes) != 1 || second.changes[0].New != "first.md" {
		t.Errorf("root record = %+v", second)
	}
}

// TestGuidanceFence_FramingIsNotData drives messages and paths that carry
// what a byte-framed log would have taken for framing: control bytes in a
// message, and quote characters in a routed path.
func TestGuidanceFence_FramingIsNotData(t *testing.T) {
	t.Parallel()
	reworded := fenceHostFile("# Dev rules\n\nKeep it boring.\n", "@.claude/aiwf-guidance.md", "route v1")

	t.Run("control bytes in a message neither hide the commit nor its block", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, base := guidanceFenceFixture(t)
		writeFile("CLAUDE.md", reworded)
		writeFile(fenceCodeFile, "package app\n\nvar x = 1\n")
		runGit("add", "-A")
		runGit("commit", "-q", "-m", "docs: reword \x1e\x1d\x1f\n\nRemoved: keep it simple\nDisposition: deleted",
			"--trailer", "aiwf-entity: "+provFixtureEntityID)
		if got, want := fenceViolationFiles(t, root, base), []string{fenceCodeFile}; !equalStrings(got, want) {
			t.Errorf("violation files = %v, want %v", got, want)
		}
	})

	t.Run("a routed path with quote characters is judged and read", func(t *testing.T) {
		t.Parallel()
		const quoted = `docs/a"b.md`
		root, runGit, writeFile, _ := guidanceFenceFixture(t)
		writeFile(fenceProjectDoc, "# Project guidance\n\nRead [q](../"+quoted+").\n")
		writeFile(quoted, "# Quoted\n")
		commitOwned(runGit, "docs(guidance): route a quoted document")
		base := trimLine(runGit("rev-parse", "HEAD"))
		writeFile(quoted, "# Quoted\n\nMore.\n")
		writeFile(fenceCodeFile, "package app\n\nvar x = 1\n")
		commitOwned(runGit, "docs(guidance): edit it beside code")
		if got, want := fenceViolationFiles(t, root, base), []string{fenceCodeFile}; !equalStrings(got, want) {
			t.Errorf("violation files = %v, want %v", got, want)
		}
	})
}

// TestGuidanceFence_InsertedBlockIsNotHandwritten pins that aiwf's own
// insertion of a managed block, written by the real splice with the blank
// line it adds outside the block, is not a handwritten change.
func TestGuidanceFence_InsertedBlockIsNotHandwritten(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := guidanceFenceFixture(t)
	gStart, gEnd, _ := initrepo.GuidanceMarkers()
	bare := "# Dev rules\n\nKeep it simple.\n\n" + gStart + "\n@.claude/aiwf-guidance.md\n" + gEnd + "\n"
	writeFile("CLAUDE.md", bare)
	commitOwned(runGit, "docs(guidance): no routing block yet")
	base := trimLine(runGit("rev-parse", "HEAD"))
	rStart, rEnd, rPrefix := projectguidance.RouteMarkers()
	spliced, err := pathutil.SpliceManagedBlock(bare, "route v1", rStart, rEnd, rPrefix)
	if err != nil {
		t.Fatal(err)
	}
	writeFile("CLAUDE.md", spliced)
	writeFile(".guidance/.aiwf-owned", fenceOwnedJSON("ee"))
	writeFile(fenceCodeFile, "package app\n\nvar x = 1\n")
	runGit("add", "-A")
	runGit("commit", "-q", "-m", "chore: aiwf update")
	if got := fenceViolationFiles(t, root, base); len(got) != 0 {
		t.Errorf("an inserted block is generated output; got violations on %v", got)
	}
}

// TestGuidanceFence_OwnedRecordCannotExemptCode pins that the owned record
// relates only paths shaped like ones aiwf owns, and that a rename is
// related only when both its sides are.
func TestGuidanceFence_OwnedRecordCannotExemptCode(t *testing.T) {
	t.Parallel()
	addRule := func(w func(string, string)) {
		w("CLAUDE.md", fenceHostFile(fenceClaudeText+"\nA new rule.\n", "@.claude/aiwf-guidance.md", "route v1"))
	}
	t.Run("a code path listed in the record is still unrelated", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, base := guidanceFenceFixture(t)
		addRule(writeFile)
		writeFile(".guidance/.aiwf-owned", "{\n  \"internal/app/app.go\": \"aa\"\n}\n")
		writeFile(fenceCodeFile, "package app\n\nvar x = 1\n")
		commitOwned(runGit, "docs(guidance): rule")
		if got, want := fenceViolationFiles(t, root, base), []string{fenceCodeFile}; !equalStrings(got, want) {
			t.Errorf("violation files = %v, want %v", got, want)
		}
	})
	t.Run("an owned file renamed out of the owned set is unrelated", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, base := guidanceFenceFixture(t)
		addRule(writeFile)
		runGit("mv", fencePack, "internal/app/notes.md")
		commitOwned(runGit, "docs(guidance): rule")
		if got, want := fenceViolationFiles(t, root, base), []string{"internal/app/notes.md"}; !equalStrings(got, want) {
			t.Errorf("violation files = %v, want %v", got, want)
		}
	})
}

// TestRoutedDocuments pins what the router counts as a route: a markdown
// link in any CommonMark form — titled, angle-bracketed, reference-style,
// percent-escaped — resolved against the router's directory, or the
// repository root for a leading slash, anchor dropped. A link with a
// scheme, a link to something other than markdown, and a link leaving the
// repository are not routes.
func TestRoutedDocuments(t *testing.T) {
	t.Parallel()
	router := "Read [a](../docs/a.md#part), [b](b.md), [web](https://example.com/c.md),\n" +
		"[img](../x.png), [out](../../outside.md), [t](../docs/t.md \"titled\"),\n" +
		"[angle](<../docs/my doc.md>), [root](/docs/r.md), [esc](../docs/sp%20ace.md) and [ref][r].\n\n" +
		"[r]: ../docs/ref.md\n"
	got := routedDocuments(router)
	want := []string{"docs/a.md", ".guidance/b.md", "docs/t.md", "docs/my doc.md", "docs/r.md", "docs/sp ace.md", "docs/ref.md"}
	if !equalStrings(got, want) {
		t.Errorf("routedDocuments = %v, want %v", got, want)
	}
}

// TestHandwrittenText pins that a host file's managed blocks are removed
// and that malformed markers leave the whole file judged as handwritten.
func TestHandwrittenText(t *testing.T) {
	t.Parallel()
	if handwrittenText(fenceHostFile("mine\n", "generated v1", "route v1")) != handwrittenText(fenceHostFile("mine\n", "generated v2", "route v2")) {
		t.Error("a change inside the managed blocks must leave the handwritten text unchanged")
	}
	if handwrittenText(fenceHostFile("mine\n", "generated", "route")) == handwrittenText(fenceHostFile("yours\n", "generated", "route")) {
		t.Error("a change outside the managed blocks must change the handwritten text")
	}
	gStart, _, _ := initrepo.GuidanceMarkers()
	broken := "mine\n" + gStart + "\nno end marker\n"
	if got := handwrittenText(broken); got != broken {
		t.Errorf("malformed markers: handwrittenText = %q, want the whole file", got)
	}
}

// TestGuidanceFence_WiredIntoCoverageGate pins that the fence runs at the
// integration boundary: it is named in the coverage-gate run-pattern of
// both the CI workflow and the Makefile target.
func TestGuidanceFence_WiredIntoCoverageGate(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, f := range []string{".github/workflows/go.yml", "Makefile"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if line := coverageGateRunLine(t, f, string(data)); !strings.Contains(line, "|GuidanceFence)") && !strings.Contains(line, "|GuidanceFence|") {
			t.Errorf("%s: coverage-gate run-pattern does not include GuidanceFence:\n  %s", f, line)
		}
	}
}

// TestPolicy_GuidanceFence is the CI gate entry point, run with the base
// ref in AIWF_COVERAGE_BASE by the coverage-gate step and
// `make coverage-gate`; without one it skips.
func TestPolicy_GuidanceFence(t *testing.T) {
	t.Parallel()
	if os.Getenv("AIWF_COVERAGE_BASE") == "" {
		t.Skip("AIWF_COVERAGE_BASE unset; run via `make coverage-gate` or the CI coverage-gate step")
	}
	runPolicy(t, PolicyGuidanceFence)
}

// TestDetectGuidanceFence_Trailer is M-0333 AC-2's rule: a commit that
// changes handwritten guidance names the entity it belongs to, and the
// name resolves. Ids compare canonicalized and a composite id is owned by
// its milestone, so both resolve through the entity they name.
func TestDetectGuidanceFence_Trailer(t *testing.T) {
	t.Parallel()
	guidance := func(entity string) []fenceCommit {
		return []fenceCommit{{SHA: "aaaaaaaaaa", Entity: entity, Guidance: []string{"CLAUDE.md"}}}
	}
	tests := []struct {
		name   string
		in     []fenceCommit
		want   int
		detail string
	}{
		{name: "no trailer fires once", in: guidance(""), want: 1, detail: "no aiwf-entity"},
		{name: "an unresolvable value fires once and names it", in: guidance("M-9999"), want: 1, detail: "M-9999"},
		{name: "a resolving value is silent", in: guidance("M-0312")},
		{name: "a narrow legacy id resolves after canonicalization", in: guidance("M-312")},
		{name: "a composite id resolves to its milestone", in: guidance("M-0312/AC-2")},
		{name: "a commit changing no handwritten guidance needs no trailer", in: []fenceCommit{{SHA: "bbbbbbbbbb", Unrelated: []string{"x.go"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := detectGuidanceFence(tt.in, resolvesOnly("M-0312"))
			if len(got) != tt.want {
				t.Fatalf("got %d violations %+v, want %d", len(got), got, tt.want)
			}
			if tt.want > 0 && (!strings.Contains(got[0].Detail, "aaaaaaa") || !strings.Contains(got[0].Detail, tt.detail)) {
				t.Errorf("Detail must name the commit and %q; got %q", tt.detail, got[0].Detail)
			}
		})
	}
}

// TestGuidanceFence_TrailerSeam drives M-0333 AC-2 through git and the
// loader: the missing and unresolvable cases each fire once, a live
// entity and an archived one both resolve.
func TestGuidanceFence_TrailerSeam(t *testing.T) {
	t.Parallel()
	const archived = "work/epics/archive/E-0002-fictional-archived-epic/epic.md"
	tests := []struct {
		name    string
		trailer string
		want    []string
	}{
		{name: "no trailer fires on the guidance file", want: []string{"CLAUDE.md"}},
		{name: "an unresolvable trailer fires on the guidance file", trailer: "M-9999", want: []string{"CLAUDE.md"}},
		{name: "a live entity resolves", trailer: provFixtureEntityID},
		{name: "an archived entity resolves", trailer: "E-0002"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, runGit, writeFile, _ := guidanceFenceFixture(t)
			writeFile(archived, "---\nid: E-0002\ntitle: Fictional archived epic\nstatus: done\n---\n## Goal\n\nFixture.\n")
			runGit("add", "-A")
			runGit("commit", "-q", "-m", "archive fixture")
			base := trimLine(runGit("rev-parse", "HEAD"))
			writeFile("CLAUDE.md", fenceHostFile(fenceClaudeText+"\nA new rule.\n", "@.claude/aiwf-guidance.md", "route v1"))
			runGit("add", "-A")
			args := []string{"commit", "-q", "-m", "docs(guidance): rule"}
			if tt.trailer != "" {
				args = append(args, "--trailer", "aiwf-entity: "+tt.trailer)
			}
			runGit(args...)
			if got := fenceViolationFiles(t, root, base); !equalStrings(got, tt.want) {
				t.Errorf("violation files = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGuidanceFence_TreeLoadFailure pins that a planning tree the loader
// cannot read is an error rather than a silent pass: the trailer arm
// cannot answer without it.
func TestGuidanceFence_TreeLoadFailure(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits do not deny the walk")
	}
	root, runGit, writeFile, base := guidanceFenceFixture(t)
	writeFile("CLAUDE.md", fenceHostFile(fenceClaudeText+"\nA new rule.\n", "@.claude/aiwf-guidance.md", "route v1"))
	commitOwned(runGit, "docs(guidance): rule")
	denied := filepath.Join(root, "work", "epics")
	if err := os.Chmod(denied, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(denied, 0o755) })
	if _, err := guidanceFenceViolations(root, base); err == nil {
		t.Fatal("want an error when the planning tree cannot be read, got nil")
	}
}

// TestDispositionBlocks pins M-0333 AC-3's block grammar: a Removed: line
// immediately followed by a Disposition: line, whose value is one of the
// closed set. The check is shape only; whether the named path or id
// exists is held at review.
func TestDispositionBlocks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		body            string
		wellFormed, bad int
	}{
		{name: "copy of", body: "Removed: the rule\nDisposition: copy of internal/skills/embedded-guidance/aiwf-guidance.md\n", wellFormed: 1},
		{name: "relocated to", body: "Removed: the rule\nDisposition: relocated to docs/dev/testing.md\n", wellFormed: 1},
		{name: "pointer to", body: "Removed: the rule\nDisposition: pointer to shipped-prose-assertion\n", wellFormed: 1},
		{name: "deleted", body: "Removed: the rule\nDisposition: deleted\n", wellFormed: 1},
		{name: "a value outside the set is malformed", body: "Removed: the rule\nDisposition: moved somewhere\n", bad: 1},
		{name: "a form missing its target is malformed", body: "Removed: the rule\nDisposition: copy of\n", bad: 1},
		{name: "Removed without a following Disposition is malformed", body: "Removed: the rule\n\nDisposition: deleted\n", bad: 1},
		{name: "Removed on the message's last line is malformed", body: "Removed: the rule", bad: 1},
		{name: "a Disposition with no Removed is not a block", body: "Disposition: deleted\n"},
		{name: "two blocks both count", body: "Removed: a\nDisposition: deleted\nRemoved: b\nDisposition: relocated to x.md\n", wellFormed: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			good, bad := dispositionBlocks(tt.body)
			if good != tt.wellFormed || len(bad) != tt.bad {
				t.Errorf("dispositionBlocks = %d well-formed, %v malformed; want %d, %d", good, bad, tt.wellFormed, tt.bad)
			}
		})
	}
}

// TestDetectGuidanceFence_Disposition is M-0333 AC-3's rule: a guidance
// commit that removes text records at least one well-formed disposition
// block, and a malformed block is refused.
func TestDetectGuidanceFence_Disposition(t *testing.T) {
	t.Parallel()
	commit := func(removes bool, body string) []fenceCommit {
		return []fenceCommit{{SHA: "cccccccccc", Entity: "M-0312", Guidance: []string{"CLAUDE.md"}, Removes: removes, Body: body}}
	}
	tests := []struct {
		name string
		in   []fenceCommit
		want int
	}{
		{name: "a removal with no block fires once", in: commit(true, "docs: cut\n"), want: 1},
		{name: "a removal with a malformed block fires once", in: commit(true, "Removed: x\nDisposition: gone\n"), want: 1},
		{name: "a removal with a well-formed block is silent", in: commit(true, "Removed: x\nDisposition: deleted\n")},
		{name: "a pure addition needs no block", in: commit(false, "docs: add\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := detectGuidanceFence(tt.in, resolvesOnly("M-0312")); len(got) != tt.want {
				t.Errorf("got %d violations %+v, want %d", len(got), got, tt.want)
			}
		})
	}
}

// TestGuidanceFence_DispositionSeam drives M-0333 AC-3 through git: a
// rewording is a removal plus an addition, a block written as trailers is
// still a block, and a removal inside a managed block is generated output
// the fence does not judge.
func TestGuidanceFence_DispositionSeam(t *testing.T) {
	t.Parallel()
	reworded := fenceHostFile("# Dev rules\n\nKeep it boring.\n", "@.claude/aiwf-guidance.md", "route v1")
	tests := []struct {
		name  string
		write func(w func(string, string))
		extra []string
		want  []string
	}{
		{
			name:  "a rewording without a block fires",
			write: func(w func(string, string)) { w("CLAUDE.md", reworded) },
			want:  []string{"CLAUDE.md"},
		},
		{
			name:  "a block in trailer position is a block",
			write: func(w func(string, string)) { w("CLAUDE.md", reworded) },
			extra: []string{"--trailer", "Removed: keep it simple", "--trailer", "Disposition: deleted"},
		},
		{
			name:  "removing a routed document without a block fires",
			write: func(w func(string, string)) { w(fenceRoutedDoc, "# Testing\n") },
			want:  []string{fenceRoutedDoc},
		},
		{
			name: "a removal inside a managed block is not judged",
			write: func(w func(string, string)) {
				w("AGENTS.md", fenceHostFile(fenceAgentsText, "", "route v1"))
				w(fenceFragmentSrc, "")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, runGit, writeFile, base := guidanceFenceFixture(t)
			tt.write(writeFile)
			runGit("add", "-A")
			runGit(append([]string{"commit", "-q", "-m", "docs(guidance): change", "--trailer", "aiwf-entity: " + provFixtureEntityID}, tt.extra...)...)
			if got := fenceViolationFiles(t, root, base); !equalStrings(got, tt.want) {
				t.Errorf("violation files = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseFenceLog_Malformed pins that a field where a header belongs
// that is not one is an error, and that a header cut off before its fields
// reads as a commit with none.
func TestParseFenceLog_Malformed(t *testing.T) {
	t.Parallel()
	if got, err := parseFenceLog(""); err != nil || len(got) != 0 {
		t.Errorf("an empty log = %+v, %v; want no commits and no error", got, err)
	}
	if _, err := parseFenceLog("Good signature for someone\x00"); err == nil {
		t.Error("a stream opening with something other than a header: want an error")
	}
	sha := strings.Repeat("d", 40)
	got, err := parseFenceLog(sha)
	if err != nil || len(got) != 1 || got[0].sha != sha || got[0].entity != "" || got[0].body != "" || len(got[0].changes) != 0 {
		t.Errorf("parseFenceLog = %+v, %v; want one commit with no fields", got, err)
	}
}

// TestStderrOf pins that only a command's exit error carries stderr.
func TestStderrOf(t *testing.T) {
	t.Parallel()
	if got := stderrOf(errors.New("not an exit")); got != nil {
		t.Errorf("stderrOf(non-exit error) = %q, want nil", got)
	}
	_, err := exec.Command("git", "not-a-git-command").Output()
	if got := stderrOf(err); len(got) == 0 {
		t.Error("stderrOf(exit error) must return what git wrote to stderr")
	}
}

// TestGuidanceFence_Deletions pins the deletion side of the related-file
// rule: deleting an owned output with a guidance change is related, and
// deleting an unrelated file is not.
func TestGuidanceFence_Deletions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, gone string
		want       []string
	}{
		{name: "an owned output", gone: fencePack},
		{name: "an unrelated file", gone: fenceCodeFile, want: []string{fenceCodeFile}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, runGit, writeFile, base := guidanceFenceFixture(t)
			writeFile("CLAUDE.md", fenceHostFile(fenceClaudeText+"\nA new rule.\n", "@.claude/aiwf-guidance.md", "route v1"))
			runGit("rm", "-q", tc.gone)
			commitOwned(runGit, "docs(guidance): rule")
			if got := fenceViolationFiles(t, root, base); !equalStrings(got, tc.want) {
				t.Errorf("violation files = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestGuidanceFence_SignedCommitIsJudged pins that a signature check git
// prints under log.showSignature does not take the place of a commit's
// header: a signed mixed commit is still refused.
func TestGuidanceFence_SignedCommitIsJudged(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen unavailable; cannot sign a fixture commit")
	}
	root, runGit, writeFile, base := guidanceFenceFixture(t)
	key := filepath.Join(t.TempDir(), "key")
	if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v\n%s", err, out)
	}
	runGit("config", "gpg.format", "ssh")
	runGit("config", "user.signingkey", key)
	runGit("config", "log.showSignature", "true")
	writeFile("CLAUDE.md", fenceHostFile(fenceClaudeText+"\nA new rule.\n", "@.claude/aiwf-guidance.md", "route v1"))
	writeFile(fenceCodeFile, "package app\n\nvar x = 1\n")
	runGit("add", "-A")
	runGit("commit", "-q", "-S", "-m", "docs(guidance): signed", "--trailer", "aiwf-entity: "+provFixtureEntityID)
	if got, want := fenceViolationFiles(t, root, base), []string{fenceCodeFile}; !equalStrings(got, want) {
		t.Errorf("violation files = %v, want %v", got, want)
	}
}

// TestGuidanceFence_RelatedRules pins two rules of what may ride with a
// guidance change: an owned output the router links to is an output, not
// guidance; and a host file changed only inside its managed blocks rides
// along as generated output, whatever the owned record lists.
func TestGuidanceFence_RelatedRules(t *testing.T) {
	t.Parallel()
	t.Run("an owned output the router links to is related", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, _ := guidanceFenceFixture(t)
		writeFile(fenceProjectDoc, "# Project guidance\n\nFor tests read [testing](../"+fenceRoutedDoc+"). Go: [pack](packs/go/guide.md).\n")
		commitOwned(runGit, "docs(guidance): route the pack\n\nRemoved: old route line\nDisposition: deleted")
		base := trimLine(runGit("rev-parse", "HEAD"))
		writeFile(fencePack, "# Go pack\n\nRegenerated.\n")
		writeFile(".guidance/.aiwf-owned", fenceOwnedJSON("ff"))
		writeFile(fenceCodeFile, "package app\n\nvar x = 1\n")
		runGit("add", "-A")
		runGit("commit", "-q", "-m", "chore: aiwf update beside code")
		if got := fenceViolationFiles(t, root, base); len(got) != 0 {
			t.Errorf("a regenerated owned output is not a guidance change; got %v", got)
		}
	})
	t.Run("a host file changed only in its blocks rides along", func(t *testing.T) {
		t.Parallel()
		root, runGit, writeFile, _ := guidanceFenceFixture(t)
		writeFile(".guidance/.aiwf-owned", "{\n  \".guidance/index.md\": \"aa\"\n}\n")
		commitOwned(runGit, "chore: owned record without host files")
		base := trimLine(runGit("rev-parse", "HEAD"))
		writeFile("AGENTS.md", fenceHostFile(fenceAgentsText+"\nAnother rule.\n", "fragment v1", "route v1"))
		writeFile("CLAUDE.md", fenceHostFile(fenceClaudeText, "@.claude/aiwf-guidance.md", "route v2"))
		commitOwned(runGit, "docs(guidance): rule")
		if got := fenceViolationFiles(t, root, base); len(got) != 0 {
			t.Errorf("a block-only host change is related; got %v", got)
		}
	})
}

// TestRemovesLine pins what counts as removing a line: a line inserted
// anywhere removes nothing, a reworded line removes the old one, and a
// repeated line removed once is a removal.
func TestRemovesLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, before, after string
		want                bool
	}{
		{"an insertion mid-file", "a\nb\nc\n", "a\nb\nnew\nc\n", false},
		{"a line moved", "a\nb\n", "b\na\n", false},
		{"a rewording", "a\nb\n", "a\nB\n", true},
		{"one of two repeats removed", "x\nx\ny\n", "x\ny\n", true},
		{"blank lines removed", "a\n\n\nb\n", "a\nb\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := removesLine(tt.before, tt.after); got != tt.want {
				t.Errorf("removesLine = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGuidanceFence_CreatedHostFileHoldingOnlyBlocks pins that a host file
// aiwf creates holding only its managed blocks is generated output, not a
// handwritten change, so it may ride with any commit.
func TestGuidanceFence_CreatedHostFileHoldingOnlyBlocks(t *testing.T) {
	t.Parallel()
	root, runGit, writeFile, _ := guidanceFenceFixture(t)
	runGit("rm", "-q", "AGENTS.md")
	commitOwned(runGit, "chore: no AGENTS.md yet\n\nRemoved: the Codex entry point\nDisposition: deleted")
	base := trimLine(runGit("rev-parse", "HEAD"))
	gStart, gEnd, _ := initrepo.GuidanceMarkers()
	writeFile("AGENTS.md", gStart+"\nfragment v1\n"+gEnd+"\n")
	writeFile(fenceCodeFile, "package app\n\nvar x = 1\n")
	runGit("add", "-A")
	runGit("commit", "-q", "-m", "chore: aiwf init beside code")
	if got := fenceViolationFiles(t, root, base); len(got) != 0 {
		t.Errorf("a created host file holding only blocks is generated output; got %v", got)
	}
}
