package check

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/pathutil"
	"github.com/23min/aiwf/internal/projectguidance"
)

// Fixture paths. docs/dev/testing.md is routed from the project router and
// so is handwritten instruction content; docs/other.md is not routed and is
// outside the set.
const (
	fenceRoutedDoc   = "docs/dev/testing.md"
	fenceUnroutedDoc = "docs/other.md"
	fenceCodeFile    = "internal/app/app.go"
	fenceFragmentSrc = "internal/skills/embedded-guidance/aiwf-guidance.md"
	fencePack        = ".guidance/packs/go/guide.md"
	fenceIndex       = ".guidance/index.md"
	fenceClaudeText  = "# Dev rules\n\nKeep it simple.\n"
	fenceAgentsText  = "# Codex rules\n\nRead the router first.\n"
	fenceRouterText  = "# Project guidance\n\nFor tests read [testing](../" + fenceRoutedDoc + ").\n"
)

// fenceHostFile renders a fictional host instruction file: handwritten
// text, then the aiwf:guidance block carrying guidanceBody, then the
// routing block carrying routeBody.
func fenceHostFile(handwritten, guidanceBody, routeBody string) string {
	gStart, gEnd, _ := projectguidance.GuidanceMarkers()
	rStart, rEnd, _ := projectguidance.RouteMarkers()
	return handwritten + "\n" +
		gStart + "\n" + guidanceBody + "\n" + gEnd + "\n\n" +
		rStart + "\n" + routeBody + "\n" + rEnd + "\n"
}

// fenceOwnedJSON is an owned-output record listing the pack index, one pack
// and both host files, the pack with the given digest.
func fenceOwnedJSON(packDigest string) string {
	return "{\n" +
		`  ".guidance/index.md": "aa",` + "\n" +
		`  ".guidance/packs/go/guide.md": "` + packDigest + `",` + "\n" +
		`  "AGENTS.md": "cc",` + "\n" +
		`  "CLAUDE.md": "cc"` + "\n}\n"
}

// fenceRepo is a throwaway repository carrying a project's whole
// instruction set, its owned outputs and their record, configuration, the
// aiwf fragment source, and an unrelated code file.
type fenceRepo struct {
	t    *testing.T
	root string
	base string
}

func newFenceRepo(t *testing.T) *fenceRepo {
	t.Helper()
	r := &fenceRepo{t: t, root: t.TempDir()}
	r.git("init", "-q", "-b", "main")
	r.write(projectguidance.ClaudeFile, fenceHostFile(fenceClaudeText, "@.claude/aiwf-guidance.md", "route v1"))
	r.write(projectguidance.AgentsFile, fenceHostFile(fenceAgentsText, "fragment v1", "route v1"))
	r.write(projectguidance.RouterFile, fenceRouterText)
	r.write(fenceRoutedDoc, "# Testing\n\nRun the suite.\n")
	r.write(fenceUnroutedDoc, "# Other\n\nNot routed.\n")
	r.write(projectguidance.OwnedRecordFile, fenceOwnedJSON("bb"))
	r.write(fenceIndex, "# Index\n")
	r.write(fencePack, "# Go pack\n")
	r.write(fenceFragmentSrc, "fragment v1\n")
	r.write("aiwf.yaml", "hosts: [claude-code, codex]\n")
	r.write(fenceCodeFile, "package app\n")
	r.commit("seed the fixture")
	r.base = strings.TrimSpace(r.git("rev-parse", "HEAD"))
	return r
}

func (r *fenceRepo) git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func (r *fenceRepo) write(rel, content string) {
	r.t.Helper()
	abs := filepath.Join(r.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// commit stages everything and commits it under an entity trailer, so only
// the arm a case exercises can fire.
func (r *fenceRepo) commit(message string) {
	r.t.Helper()
	r.git("add", "-A")
	r.git("commit", "-q", "-m", message, "--trailer", "aiwf-entity: E-0001")
}

// fenceFindings runs the fence over the commits since the fixture's seed, with
// no trunk ref.
func (r *fenceRepo) fenceFindings() []Finding {
	r.t.Helper()
	return r.fenceFindingsBeside("")
}

// fenceFindingsBeside runs the fence over the commits since the fixture's seed,
// skipping those already on trunk.
func (r *fenceRepo) fenceFindingsBeside(trunk string) []Finding {
	r.t.Helper()
	commits, err := WalkInstructionCommits(context.Background(), r.root, r.base+"..HEAD", trunk)
	if err != nil {
		r.t.Fatalf("WalkInstructionCommits: %v", err)
	}
	return RunInstructionFence(commits, func(string) bool { return true })
}

// unrelatedPaths returns the paths the unrelated-file code names, sorted.
func unrelatedPaths(fs []Finding) []string {
	var out []string
	for i := range fs {
		if fs[i].Code == CodeInstructionCommitUnrelatedFile.ID {
			out = append(out, fs[i].Path)
		}
	}
	slices.Sort(out)
	return out
}

// TestInstructionFence_UnrelatedFiles is M-0350 AC-1: a commit changing
// handwritten instruction content may carry the other instruction files,
// the owned outputs and their record, and aiwf.yaml, and nothing else;
// each other file is one finding naming the commit.
func TestInstructionFence_UnrelatedFiles(t *testing.T) {
	t.Parallel()

	addToClaude := func(r *fenceRepo) {
		r.write(projectguidance.ClaudeFile, fenceHostFile(fenceClaudeText+"\nA new rule.\n", "@.claude/aiwf-guidance.md", "route v1"))
	}
	touchCode := func(r *fenceRepo) { r.write(fenceCodeFile, "package app\n\nvar x = 1\n") }

	tests := []struct {
		name   string
		change func(r *fenceRepo)
		want   []string
	}{
		{
			name:   "a CLAUDE.md edit beside a code edit refuses the code file",
			change: func(r *fenceRepo) { addToClaude(r); touchCode(r) },
			want:   []string{fenceCodeFile},
		},
		{
			name: "an AGENTS.md edit beside a code edit refuses the code file",
			change: func(r *fenceRepo) {
				r.write(projectguidance.AgentsFile, fenceHostFile(fenceAgentsText+"\nAnother rule.\n", "fragment v1", "route v1"))
				touchCode(r)
			},
			want: []string{fenceCodeFile},
		},
		{
			name: "a router edit beside a code edit refuses the code file",
			change: func(r *fenceRepo) {
				r.write(projectguidance.RouterFile, fenceRouterText+"\nMore.\n")
				touchCode(r)
			},
			want: []string{fenceCodeFile},
		},
		{
			name: "a routed document edit beside a code edit refuses the code file",
			change: func(r *fenceRepo) {
				r.write(fenceRoutedDoc, "# Testing\n\nRun the suite.\n\nAnd the race detector.\n")
				touchCode(r)
			},
			want: []string{fenceCodeFile},
		},
		{
			name: "an update carrying owned outputs, their record and aiwf.yaml passes",
			change: func(r *fenceRepo) {
				addToClaude(r)
				r.write(projectguidance.AgentsFile, fenceHostFile(fenceAgentsText+"\nAnother rule.\n", "fragment v2", "route v2"))
				r.write(fencePack, "# Go pack\n\nMore.\n")
				r.write(fenceIndex, "# Index\n\nGo.\n")
				r.write(projectguidance.OwnedRecordFile, fenceOwnedJSON("dd"))
				r.write("aiwf.yaml", "hosts: [claude-code, codex]\nguidance:\n  packs: [go]\n")
			},
		},
		{
			name: "a file under .guidance/ outside the owned record is refused",
			change: func(r *fenceRepo) {
				addToClaude(r)
				r.write(".guidance/notes.md", "# Notes\n")
			},
			want: []string{".guidance/notes.md"},
		},
		{
			name: "a code file listed in the owned record is still refused",
			change: func(r *fenceRepo) {
				r.write(projectguidance.OwnedRecordFile, strings.Replace(fenceOwnedJSON("bb"), "{\n", "{\n  \""+fenceCodeFile+"\": \"ee\",\n", 1))
				r.commit("chore: the record lists a code file")
				addToClaude(r)
				touchCode(r)
			},
			want: []string{fenceCodeFile},
		},
		{
			name: "an owned file renamed out of the owned set is refused",
			change: func(r *fenceRepo) {
				addToClaude(r)
				r.git("mv", fencePack, "internal/app/notes.md")
			},
			want: []string{"internal/app/notes.md"},
		},
		{
			name: "the aiwf fragment source is not a companion",
			change: func(r *fenceRepo) {
				addToClaude(r)
				r.write(fenceFragmentSrc, "fragment v1\nfragment addition\n")
			},
			want: []string{fenceFragmentSrc},
		},
		{
			name: "a change confined to a managed block is not judged",
			change: func(r *fenceRepo) {
				r.write(projectguidance.AgentsFile, fenceHostFile(fenceAgentsText, "fragment v2", "route v1"))
				touchCode(r)
			},
		},
		{
			name: "an unrouted document is not instruction content",
			change: func(r *fenceRepo) {
				r.write(fenceUnroutedDoc, "# Other\n\nNot routed.\n\nStill not.\n")
				touchCode(r)
			},
		},
		{
			name: "renaming a routed document with its route passes",
			change: func(r *fenceRepo) {
				r.git("mv", fenceRoutedDoc, "docs/dev/tests.md")
				r.write(projectguidance.RouterFile, "# Project guidance\n\nFor tests read [testing](../docs/dev/tests.md).\n")
			},
		},
		{
			name: "renaming a routed document beside a code edit refuses the code file",
			change: func(r *fenceRepo) {
				r.git("mv", fenceRoutedDoc, "docs/dev/tests.md")
				touchCode(r)
			},
			want: []string{fenceCodeFile},
		},
		{
			name: "deleting an owned output beside an instruction edit passes",
			change: func(r *fenceRepo) {
				addToClaude(r)
				r.git("rm", "-q", fencePack)
			},
		},
		{
			name: "an owned output the router links to is generated, not instruction content",
			change: func(r *fenceRepo) {
				r.write(projectguidance.RouterFile, fenceRouterText+"Go: [pack](packs/go/guide.md).\n")
				r.commit("docs(guidance): route the pack")
				r.write(fencePack, "# Go pack\n\nRegenerated.\n")
				touchCode(r)
			},
		},
		{
			name: "aiwf inserting a block into a host file beside code is not a handwritten change",
			change: func(r *fenceRepo) {
				gStart, gEnd, _ := projectguidance.GuidanceMarkers()
				bare := fenceClaudeText + "\n" + gStart + "\n@.claude/aiwf-guidance.md\n" + gEnd + "\n"
				r.write(projectguidance.ClaudeFile, bare)
				r.commit("docs(guidance): no routing block yet")
				rStart, rEnd, rPrefix := projectguidance.RouteMarkers()
				spliced, err := pathutil.SpliceManagedBlock(bare, "route v1", rStart, rEnd, rPrefix)
				if err != nil {
					r.t.Fatal(err)
				}
				r.write(projectguidance.ClaudeFile, spliced)
				touchCode(r)
			},
		},
		{
			name: "aiwf creating a host file holding only blocks beside code is not a handwritten change",
			change: func(r *fenceRepo) {
				r.git("rm", "-q", projectguidance.AgentsFile)
				r.commit("chore: no AGENTS.md yet\n\nRemoved: the Codex entry point\nDisposition: deleted")
				gStart, gEnd, _ := projectguidance.GuidanceMarkers()
				r.write(projectguidance.AgentsFile, gStart+"\nfragment v1\n"+gEnd+"\n")
				touchCode(r)
			},
		},
		{
			name: "a routed path with quote characters is judged and read",
			change: func(r *fenceRepo) {
				const quoted = `docs/a"b.md`
				r.write(projectguidance.RouterFile, fenceRouterText+"Read [q](<../"+quoted+">).\n")
				r.write(quoted, "# Quoted\n")
				r.commit("docs(guidance): route a quoted document")
				r.write(quoted, "# Quoted\n\nMore.\n")
				touchCode(r)
			},
			want: []string{fenceCodeFile},
		},
		{
			name: "a host file changed only in its blocks rides along",
			change: func(r *fenceRepo) {
				r.write(projectguidance.OwnedRecordFile, "{\n  \".guidance/index.md\": \"aa\"\n}\n")
				r.commit("chore: a record listing no host file")
				r.write(projectguidance.AgentsFile, fenceHostFile(fenceAgentsText+"\nAnother rule.\n", "fragment v1", "route v1"))
				r.write(projectguidance.ClaudeFile, fenceHostFile(fenceClaudeText, "@.claude/aiwf-guidance.md", "route v2"))
			},
		},
		{
			name: "deleting an unrelated file beside an instruction edit refuses it",
			change: func(r *fenceRepo) {
				addToClaude(r)
				r.git("rm", "-q", fenceCodeFile)
			},
			want: []string{fenceCodeFile},
		},
		{
			name: "adding an owned output the record lists beside an instruction edit passes",
			change: func(r *fenceRepo) {
				addToClaude(r)
				r.write(".guidance/packs/py/guide.md", "# Python pack\n")
				r.write(projectguidance.OwnedRecordFile, strings.Replace(fenceOwnedJSON("bb"), "{\n", "{\n  \".guidance/packs/py/guide.md\": \"ff\",\n", 1))
			},
		},
		{
			name: "aiwf deleting a host file holding only blocks beside code is not a handwritten change",
			change: func(r *fenceRepo) {
				gStart, gEnd, _ := projectguidance.GuidanceMarkers()
				r.write(projectguidance.AgentsFile, gStart+"\nfragment v1\n"+gEnd+"\n")
				r.commit("chore: AGENTS.md holds only its block\n\nRemoved: the Codex rules\nDisposition: deleted")
				r.git("rm", "-q", projectguidance.AgentsFile)
				touchCode(r)
			},
		},
		{
			name: "deleting a routed document beside a code edit refuses the code file",
			change: func(r *fenceRepo) {
				r.git("rm", "-q", fenceRoutedDoc)
				touchCode(r)
			},
			want: []string{fenceCodeFile},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := newFenceRepo(t)
			tt.change(r)
			r.commit("docs(guidance): change")
			fs := r.fenceFindings()
			if diff := cmp.Diff(tt.want, unrelatedPaths(fs)); diff != "" {
				t.Errorf("unrelated files (-want +got):\n%s", diff)
			}
			head := strings.TrimSpace(r.git("rev-parse", "--short", "HEAD"))
			for _, f := range fs {
				if f.Severity != SeverityError || !strings.Contains(f.Message, head) {
					t.Errorf("finding must be an error naming commit %s; got %s %q", head, f.Severity, f.Message)
				}
			}
		})
	}
}

// TestInstructionFence_MergeCommits is M-0350 AC-1's merge edge: a merge
// is not judged, and the commits it brings in are, so a merge of separate
// commits is silent and a mixed commit it brings in is refused.
func TestInstructionFence_MergeCommits(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		mixed bool
		want  []string
	}{
		{name: "a merge of separate instruction and code commits is silent"},
		{name: "a merge cannot hide a mixed commit it brings in", mixed: true, want: []string{fenceCodeFile}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newFenceRepo(t)
			r.git("checkout", "-q", "-b", "side")
			r.write(projectguidance.ClaudeFile, fenceHostFile(fenceClaudeText+"\nA new rule.\n", "@.claude/aiwf-guidance.md", "route v1"))
			if !tc.mixed {
				r.commit("docs(guidance): rule")
			}
			r.write(fenceCodeFile, "package app\n\nvar x = 1\n")
			r.commit("feat(app): code")
			r.git("checkout", "-q", "main")
			r.write("README.md", "trunk moved\n")
			r.commit("docs: trunk moves")
			r.git("merge", "-q", "--no-ff", "-m", "Merge side", "side")
			if diff := cmp.Diff(tc.want, unrelatedPaths(r.fenceFindings())); diff != "" {
				t.Errorf("unrelated files (-want +got):\n%s", diff)
			}
		})
	}
}

// TestInstructionFence_TrunkCommitsAreNotJudged is M-0350 AC-1's range: a
// commit already on a remote-tracking trunk has been pushed, so a branch
// that merges trunk is not held to it, while the branch's own commits still
// are. A local branch named as the trunk can carry unpushed commits, so it
// excludes nothing.
func TestInstructionFence_TrunkCommitsAreNotJudged(t *testing.T) {
	t.Parallel()
	r := newFenceRepo(t)
	r.git("checkout", "-q", "-b", "feature")
	r.write(fenceRoutedDoc, "# Testing\n\nRun the suite.\n\nOn the branch.\n")
	r.write("feature.go", "package app\n")
	r.commit("feat: the branch's own mixed commit")
	r.git("checkout", "-q", "main")
	r.write(projectguidance.ClaudeFile, fenceHostFile(fenceClaudeText+"\nA new rule.\n", "@.claude/aiwf-guidance.md", "route v1"))
	r.write(fenceCodeFile, "package app\n\nvar x = 1\n")
	r.commit("feat: trunk's mixed commit")
	r.git("update-ref", "refs/remotes/origin/main", "main")
	r.git("checkout", "-q", "feature")
	r.git("merge", "-q", "--no-ff", "-m", "Merge main", "main")

	if diff := cmp.Diff([]string{"feature.go", fenceCodeFile}, unrelatedPaths(r.fenceFindings())); diff != "" {
		t.Errorf("with no trunk ref (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"feature.go"}, unrelatedPaths(r.fenceFindingsBeside("refs/remotes/origin/main"))); diff != "" {
		t.Errorf("beside a remote-tracking trunk (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"feature.go", fenceCodeFile}, unrelatedPaths(r.fenceFindingsBeside("refs/heads/main"))); diff != "" {
		t.Errorf("beside a local trunk branch (-want +got):\n%s", diff)
	}
}

// TestInstructionFence_CommitFraming pins that what a byte-framed log would
// take for framing is data: control bytes in a message, and a signature
// git prints when log.showSignature is set, hide neither the commit nor
// what it changed.
func TestInstructionFence_CommitFraming(t *testing.T) {
	t.Parallel()
	addRuleBesideCode := func(r *fenceRepo) {
		r.write(projectguidance.ClaudeFile, fenceHostFile(fenceClaudeText+"\nA new rule.\n", "@.claude/aiwf-guidance.md", "route v1"))
		r.write(fenceCodeFile, "package app\n\nvar x = 1\n")
		r.git("add", "-A")
	}
	t.Run("control bytes in a message", func(t *testing.T) {
		t.Parallel()
		r := newFenceRepo(t)
		addRuleBesideCode(r)
		r.git("commit", "-q", "-m", "docs: rule \x1e\x1d\x1f\n\nbody \x1e", "--trailer", "aiwf-entity: E-0001")
		if diff := cmp.Diff([]string{fenceCodeFile}, unrelatedPaths(r.fenceFindings())); diff != "" {
			t.Errorf("unrelated files (-want +got):\n%s", diff)
		}
	})
	t.Run("a signed commit under log.showSignature", func(t *testing.T) {
		t.Parallel()
		r := newFenceRepo(t)
		key := filepath.Join(t.TempDir(), "key")
		if out, err := exec.Command("ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key).CombinedOutput(); err != nil {
			t.Skipf("ssh-keygen unavailable: %v\n%s", err, out)
		}
		r.git("config", "gpg.format", "ssh")
		r.git("config", "user.signingkey", key)
		r.git("config", "log.showSignature", "true")
		addRuleBesideCode(r)
		r.git("commit", "-q", "-S", "-m", "docs: signed", "--trailer", "aiwf-entity: E-0001")
		if diff := cmp.Diff([]string{fenceCodeFile}, unrelatedPaths(r.fenceFindings())); diff != "" {
			t.Errorf("unrelated files (-want +got):\n%s", diff)
		}
	})
}

// TestWalkInstructionCommits_RootAndRange pins the walk's two ends: a root
// commit is judged against an empty tree, and a range naming no commit is
// an error rather than a silent pass.
func TestWalkInstructionCommits_RootAndRange(t *testing.T) {
	t.Parallel()
	r := &fenceRepo{t: t, root: t.TempDir()}
	r.git("init", "-q", "-b", "main")
	r.write(projectguidance.ClaudeFile, fenceClaudeText)
	r.write(fenceCodeFile, "package app\n")
	r.commit("root: rules beside code")
	commits, err := WalkInstructionCommits(context.Background(), r.root, "HEAD", "")
	if err != nil {
		t.Fatalf("WalkInstructionCommits: %v", err)
	}
	if diff := cmp.Diff([]string{fenceCodeFile}, unrelatedPaths(RunInstructionFence(commits, func(string) bool { return true }))); diff != "" {
		t.Errorf("root commit unrelated files (-want +got):\n%s", diff)
	}
	if _, err := WalkInstructionCommits(context.Background(), r.root, "no-such-ref..HEAD", ""); err == nil {
		t.Error("a range naming no commit: want an error")
	}
	// git log runs in the process's directory when root is empty, so it is
	// the content reader that refuses it.
	if _, err := WalkInstructionCommits(context.Background(), "", "HEAD..HEAD", ""); err == nil {
		t.Error("an empty root: want an error, not a silent pass")
	}
}

// TestParseInstructionLog pins the NUL-framed log grammar: a header of hash
// and parents, the entity trailer values, the message, then status fields
// with one path, or two for a rename or copy. A field that looks like a
// hash or a status inside a message or a path is data.
func TestParseInstructionLog(t *testing.T) {
	t.Parallel()
	sha1, sha2, sha3 := strings.Repeat("a", 40), strings.Repeat("b", 40), strings.Repeat("c", 40)
	out := sha1 + " " + sha2 + "\x00E-0001\nE-0002\x00subject\n\nM\n" + sha3 + "\n\x00\x00\n" +
		"R087\x00docs/a.md\x00docs/b.md\x00A\x00" + sha3 + "\x00D\x00gone.md\x00M\x00M\x00" +
		sha2 + "\x00\x00root\n\x00\x00\nA\x00first.md\x00"
	got, err := parseInstructionLog(out)
	if err != nil {
		t.Fatal(err)
	}
	want := []instructionLogRecord{
		{sha: sha1, parent: sha2, entity: "E-0001", body: "subject\n\nM\n" + sha3 + "\n", changes: []instructionChange{
			{Old: "docs/a.md", New: "docs/b.md"}, {New: sha3}, {Old: "gone.md"}, {Old: "M", New: "M"},
		}},
		{sha: sha2, body: "root\n", changes: []instructionChange{{New: "first.md"}}},
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(instructionLogRecord{})); diff != "" {
		t.Errorf("records (-want +got):\n%s", diff)
	}
	if _, err := parseInstructionLog("not-a-header\x00"); err == nil {
		t.Error("a stream not opening with a header: want an error")
	}
	// A record cut short after its header reads its missing fields as empty.
	sha := strings.Repeat("d", 40)
	if got, err := parseInstructionLog(sha); err != nil || !cmp.Equal(got, []instructionLogRecord{{sha: sha}}, cmp.AllowUnexported(instructionLogRecord{})) {
		t.Errorf("a header alone = %+v, %v; want one record with no fields", got, err)
	}
}

// TestRunInstructionFence_EntityTrailer is M-0350 AC-2: a commit changing
// handwritten instructions names, in its first aiwf-entity trailer, an
// entity that resolves. A missing trailer and an unresolvable value are one
// finding each, naming the commit and the value; a narrow legacy id and a
// composite acceptance-criterion id are resolved as the entity they name.
func TestRunInstructionFence_EntityTrailer(t *testing.T) {
	t.Parallel()
	known := map[string]bool{"E-0001": true, "M-0002": true}
	resolves := func(id string) bool { return known[id] }
	sha := strings.Repeat("a", 40)
	tests := []struct {
		name, entity string
		changed      []string
		want         []string // substrings the one finding's message carries
	}{
		{name: "no trailer", changed: []string{"CLAUDE.md"}, want: []string{"aaaaaaaa", "no aiwf-entity trailer"}},
		{name: "a value naming no entity", entity: "E-0099", changed: []string{"CLAUDE.md"}, want: []string{"aaaaaaaa", "E-0099"}},
		{name: "a live entity", entity: "E-0001", changed: []string{"CLAUDE.md"}},
		{name: "a narrow legacy id", entity: "E-01", changed: []string{"CLAUDE.md"}},
		{name: "a composite acceptance-criterion id", entity: "M-0002/AC-3", changed: []string{"CLAUDE.md"}},
		{name: "no instruction change needs no trailer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fs := RunInstructionFence([]InstructionCommit{{SHA: sha, Entity: tt.entity, Changed: tt.changed}}, resolves)
			var got []Finding
			for i := range fs {
				if fs[i].Code == CodeInstructionCommitEntityMissing.ID {
					got = append(got, fs[i])
				}
			}
			if len(tt.want) == 0 {
				if len(got) != 0 {
					t.Errorf("findings = %+v, want none", got)
				}
				return
			}
			if len(got) != 1 || got[0].Severity != SeverityError || got[0].Path != "CLAUDE.md" {
				t.Fatalf("findings = %+v, want one error on CLAUDE.md", got)
			}
			for _, w := range tt.want {
				if !strings.Contains(got[0].Message, w) {
					t.Errorf("message %q does not name %q", got[0].Message, w)
				}
			}
		})
	}
}

// TestRemovesText pins what counts as removing handwritten text (M-0350
// AC-3): some old line is missing from the new text and the old words do
// not survive in order. A moved line, an insertion, a re-wrap and an
// insertion re-flowed into a paragraph remove nothing; a deletion, a
// rewording and a reordering of words do.
func TestRemovesText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, before, after string
		want                bool
	}{
		{"an insertion mid-file", "a\nb\nc\n", "a\nb\nnew\nc\n", false},
		{"a line moved", "a\nb\n", "b\na\n", false},
		{"blank lines removed", "a\n\n\nb\n", "a\nb\n", false},
		{"a paragraph re-wrapped", "keep it\nsimple and\nboring\n", "keep it simple\nand boring\n", false},
		{"a sentence added and the paragraph re-flowed", "keep it\nsimple.\n", "keep it simple,\nreally. simple.\n", false},
		{"a line deleted", "a\nb\n", "a\n", true},
		{"a rewording", "a\nb\n", "a\nB\n", true},
		{"one of two repeats removed", "x\nx\ny\n", "x\ny\n", true},
		{"words reordered in a line", "always lint never push\n", "always push never lint\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := removesText(tt.before, tt.after); got != tt.want {
				t.Errorf("removesText = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDispositionBlocks pins the block grammar (M-0350 AC-3): a Removed:
// line immediately followed by a Disposition: line whose value is one of
// copy of <path>, relocated to <path>, pointer to <id>, reworded or
// deleted. Every Removed: line opens a block, so a half-written block or a
// prose line starting that way is malformed even beside a good block.
func TestDispositionBlocks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		body            string
		wellFormed, bad int
	}{
		{name: "copy of", body: "Removed: the rule\nDisposition: copy of docs/rules.md\n", wellFormed: 1},
		{name: "relocated to", body: "Removed: the rule\nDisposition: relocated to docs/dev/testing.md\n", wellFormed: 1},
		{name: "pointer to", body: "Removed: the rule\nDisposition: pointer to some-finding-code\n", wellFormed: 1},
		{name: "reworded", body: "Removed: the rule\nDisposition: reworded\n", wellFormed: 1},
		{name: "deleted", body: "Removed: the rule\nDisposition: deleted\n", wellFormed: 1},
		{name: "a value outside the set is malformed", body: "Removed: the rule\nDisposition: moved somewhere\n", bad: 1},
		{name: "a form missing its target is malformed", body: "Removed: the rule\nDisposition: copy of\n", bad: 1},
		{name: "Removed without a following Disposition is malformed", body: "Removed: the rule\n\nDisposition: deleted\n", bad: 1},
		{name: "Removed on the message's last line is malformed", body: "Removed: the rule", bad: 1},
		{name: "a prose line starting Removed: beside a good block is malformed", body: "Removed: the rule\nDisposition: deleted\n\nRemoved: nothing else, as it happens.\n", wellFormed: 1, bad: 1},
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

// TestInstructionFence_Disposition is M-0350 AC-3 end to end: a commit
// removing handwritten instruction text records where it went in a
// well-formed disposition block, in the body or written as trailers; one
// with no block, or a malformed one, is one finding; a pure addition and a
// re-wrap need none.
func TestInstructionFence_Disposition(t *testing.T) {
	t.Parallel()
	reworded := fenceHostFile("# Dev rules\n\nKeep it boring.\n", "@.claude/aiwf-guidance.md", "route v1")
	tests := []struct {
		name, file, content, body string
		want                      int
	}{
		{name: "a rewording with no block", file: projectguidance.ClaudeFile, content: reworded, want: 1},
		{name: "a rewording with a reworded block", file: projectguidance.ClaudeFile, content: reworded, body: "Removed: keep it simple\nDisposition: reworded\n\nWhy it changed."},
		{name: "a block written as trailers", file: projectguidance.ClaudeFile, content: reworded, body: "Removed: keep it simple\nDisposition: reworded"},
		{name: "a value outside the set", file: projectguidance.ClaudeFile, content: reworded, body: "Removed: keep it simple\nDisposition: tidied", want: 1},
		{name: "a stray Removed: line beside a good block", file: projectguidance.ClaudeFile, content: reworded, body: "Removed: keep it simple\nDisposition: reworded\n\nRemoved: nothing else.", want: 1},
		{name: "a routed document deleted with no block", file: fenceRoutedDoc, want: 1},
		{name: "a pure addition needs no block", file: fenceRoutedDoc, content: "# Testing\n\nRun the suite.\n\nAnd the race detector.\n"},
		{name: "a re-wrap needs no block", file: fenceRoutedDoc, content: "# Testing\n\nRun\nthe suite.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := newFenceRepo(t)
			if tt.content == "" {
				r.git("rm", "-q", tt.file)
			} else {
				r.write(tt.file, tt.content)
			}
			message := "docs(guidance): change"
			if tt.body != "" {
				message += "\n\n" + tt.body
			}
			r.commit(message)
			var got []Finding
			for _, f := range r.fenceFindings() {
				if f.Code == CodeInstructionCommitDispositionMissing.ID {
					got = append(got, f)
				}
			}
			if len(got) != tt.want {
				t.Fatalf("disposition findings = %+v, want %d", got, tt.want)
			}
			if tt.want == 1 && (got[0].Severity != SeverityError || got[0].Path != tt.file) {
				t.Errorf("finding = %+v, want an error on %s", got[0], tt.file)
			}
		})
	}
}
