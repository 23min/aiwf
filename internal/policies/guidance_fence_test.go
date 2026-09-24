package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/initrepo"
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
			commitOwned(runGit, "docs(guidance): change")
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
		commitOwned(runGit, "docs(guidance): drop a document")
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
		vs, err := PolicyGuidanceFence(repoRoot(t))
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

	t.Run("a base naming no commit is an error", func(t *testing.T) {
		t.Parallel()
		root, _, _, _ := guidanceFenceFixture(t)
		if _, err := guidanceFenceViolations(root, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"); err == nil {
			t.Fatal("want an error for a base naming no commit, got nil")
		}
	})
}

// TestParseFenceChanges pins how each --name-status shape becomes a path
// pair: a rename keeps both paths, an addition has no old path, a
// deletion no new one, and a line without a path is skipped.
func TestParseFenceChanges(t *testing.T) {
	t.Parallel()
	got := parseFenceChanges([]string{
		"R087\tdocs/a.md\tdocs/b.md",
		"A\tnew.md",
		"D\tgone.md",
		"M\tsame.md",
		"",
	})
	want := []fenceChange{{Old: "docs/a.md", New: "docs/b.md"}, {New: "new.md"}, {Old: "gone.md"}, {Old: "same.md", New: "same.md"}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("change %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestRoutedDocuments pins what the router counts as a route: a relative
// markdown link resolved against the router's directory, anchor dropped.
// A link with a scheme, a link to something other than markdown, and a
// link leaving the repository are not routes.
func TestRoutedDocuments(t *testing.T) {
	t.Parallel()
	router := "Read [a](../docs/a.md#part), [b](b.md), [web](https://example.com/c.md),\n" +
		"[img](../x.png), and [out](../../outside.md)."
	got := routedDocuments(router)
	want := []string{"docs/a.md", ".guidance/b.md"}
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
