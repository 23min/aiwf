package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// commit_msg_unparseable_test.go pins the two composition-time refusals that
// depend on what git's trailer parser can actually see.

// TestRunCommitMsg_RefusesATrailerBlockGitWillNotRead pins the split-block
// refusal. Git reads trailers only from a message's last paragraph, so a blank
// line between the aiwf block and a trailing `Co-Authored-By:` line hides the
// whole block: `aiwf history` never sees it, and the verb-value check has
// nothing to judge. The message looks correct to its author, which is why it
// has to be refused where it is written rather than reported later.
func TestRunCommitMsg_RefusesATrailerBlockGitWillNotRead(t *testing.T) {
	t.Parallel()

	verbs := map[string]struct{}{"promote": {}, "add": {}}

	cases := []struct {
		name string
		msg  string
		want int
	}{
		{
			"aiwf block split from a trailing Co-Authored-By",
			"chore(x): a subject\n\naiwf-verb: promote\naiwf-entity: M-0001\naiwf-actor: human/peter\n\nCo-Authored-By: A <a@example.com>\n",
			cliutil.ExitFindings,
		},
		{
			// The same content in one paragraph is what git reads, so it is
			// judged normally — and here that means passing.
			"the same trailers joined into one paragraph",
			"chore(x): a subject\n\naiwf-verb: promote\naiwf-entity: M-0001\naiwf-actor: human/peter\nCo-Authored-By: A <a@example.com>\n",
			cliutil.ExitOK,
		},
		{
			// Prose that mentions a trailer inline is not a trailer block.
			// Refusing it would break the case the parser indirection exists
			// for: a message discussing trailer syntax.
			"prose discussing a trailer key",
			"docs(x): explain trailers\n\nWe write aiwf-verb: promote on the last line so git reads it.\n\naiwf-verb: promote\naiwf-entity: M-0001\n",
			cliutil.ExitOK,
		},
		{
			// A subject with no trailing newline has no body at all, so
			// there are no paragraphs to inspect.
			"a bare subject with no newline",
			"chore(x): a subject",
			cliutil.ExitOK,
		},
		{
			// One trailer-shaped line is prose, not a block. It cannot be
			// told from a hidden single trailer by text or by what git
			// parsed, and refusing it would reject ordinary English that
			// opens with a key and a colon.
			"a single trailer-shaped prose line",
			"docs(x): explain\n\naiwf-entity: is the key we now require here.\n\nCo-Authored-By: A <a@example.com>\n",
			cliutil.ExitOK,
		},
		{
			// Two aiwf keys is a block wherever it sits, including where the
			// final paragraph carries an aiwf key of its own — git read that
			// one, not this, so these trailers are still lost.
			"a hidden block below a final paragraph that also carries an aiwf key",
			"feat(x): a thing\n\naiwf-entity: M-0001\naiwf-verb: frobnicate\n\nCo-Authored-By: A <a@example.com>\naiwf-entity: M-0001\n",
			cliutil.ExitFindings,
		},
		{
			// The shape a shipped-surface edit carries: the entity trailer
			// alone, because no aiwf verb commits source. One line, but its
			// value names an entity, so it is a block and it is hidden.
			"a lone entity trailer whose value names an entity",
			"docs(rituals): reword a step\n\naiwf-entity: M-0001\n\nCo-Authored-By: A <a@example.com>\n",
			cliutil.ExitFindings,
		},
		{
			"a lone composite entity trailer, likewise",
			"feat(x): a thing\n\naiwf-entity: M-0001/AC-1\n\nCo-Authored-By: A <a@example.com>\n",
			cliutil.ExitFindings,
		},
		{
			// One aiwf line whose value is not an id is prose, whatever key
			// it opens with.
			"a lone entity trailer whose value is a sentence",
			"docs(x): explain\n\naiwf-entity: is the key we now require here.\n\nCo-Authored-By: A <a@example.com>\n",
			cliutil.ExitOK,
		},
		{
			// A lone non-entity aiwf line stays prose: no value grammar
			// separates a verb name from an ordinary word.
			"a lone verb trailer is not enough to be a block",
			"docs(x): explain\n\naiwf-verb: promote\n\nCo-Authored-By: A <a@example.com>\n",
			cliutil.ExitOK,
		},
		{
			// Indentation is what separates a prose line from a trailer, so
			// the paragraph's leading whitespace must survive the trim.
			"an indented first line keeps the paragraph out of block shape",
			"docs(x): explain\n\n    aiwf-entity: M-0001\naiwf-verb: promote\n\nCo-Authored-By: A <a@example.com>\n",
			cliutil.ExitOK,
		},
		{
			"no aiwf trailers anywhere",
			"chore(x): a subject\n\nSome rationale.\n\nCo-Authored-By: A <a@example.com>\n",
			cliutil.ExitOK,
		},
		{
			// A non-final paragraph of purely non-aiwf trailers is somebody
			// else's convention, not a hidden aiwf block.
			"a non-final block carrying no aiwf key",
			"chore(x): a subject\n\nSigned-off-by: A <a@example.com>\n\naiwf-verb: promote\naiwf-entity: M-0001\n",
			cliutil.ExitOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			code := runCommitMsg(writeMsg(t, tc.msg), t.TempDir(), verbs, &buf)
			if code != tc.want {
				t.Errorf("code = %d, want %d; stderr = %q", code, tc.want, buf.String())
			}
			if tc.want == cliutil.ExitFindings && !strings.Contains(buf.String(), "last paragraph") {
				t.Errorf("refusal does not explain what git reads; stderr = %q", buf.String())
			}
		})
	}
}

// stagedRepo makes a repo with paths staged, returning its root.
func stagedRepo(t *testing.T, paths ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, a := range [][]string{
		{"init", "-q", "--initial-branch=main"},
		{"config", "user.email", "p@example.com"},
		{"config", "user.name", "P"},
	} {
		if out, err := testutil.RunGit(root, a...); err != nil {
			t.Fatalf("git %v: %v\n%s", a, err, out)
		}
	}
	for _, rel := range paths {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte("# a shipped surface\n"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		if out, err := testutil.RunGit(root, "add", rel); err != nil {
			t.Fatalf("git add %s: %v\n%s", rel, err, out)
		}
	}
	return root
}

// TestRunCommitMsg_RefusesAnUnownedShippedSurfaceEdit pins the composition-time
// half of the shipped-surface rule. The CI backstop judges commits, so a
// forgotten trailer is found once the commit exists and the repair is an amend
// or a rebase; refusing at composition costs a second instead.
//
// The rule needs no repo detection: it fires on a staged path under the ritual
// authoring tree, and a consumer repo has none, so the path predicate is the
// scope. Presence is all it can ask — resolving the id against the tree needs a
// load the hook does not do, and the CI backstop keeps that half.
func TestRunCommitMsg_RefusesAnUnownedShippedSurfaceEdit(t *testing.T) {
	t.Parallel()

	const ritual = "internal/skills/embedded-rituals/plugins/p/skills/s/SKILL.md"
	verbs := map[string]struct{}{"promote": {}}

	cases := []struct {
		name  string
		paths []string
		msg   string
		want  int
	}{
		{
			"ritual edit with no entity trailer",
			[]string{ritual},
			"docs(rituals): reword a step\n",
			cliutil.ExitFindings,
		},
		{
			"ritual edit naming its entity",
			[]string{ritual},
			"docs(rituals): reword a step\n\naiwf-entity: M-0001\n",
			cliutil.ExitOK,
		},
		{
			// The entity trailer is what is asked for; a verb is not, since no
			// aiwf verb commits source.
			"ritual edit naming an entity via a composite id",
			[]string{ritual},
			"docs(rituals): reword a step\n\naiwf-entity: M-0001/AC-2\n",
			cliutil.ExitOK,
		},
		{
			// A key with no value names nothing, so it is not an owner.
			// Accepting it would make the guard satisfiable by typing the
			// trailer key and nothing else.
			"ritual edit with an empty entity value",
			[]string{ritual},
			"docs(rituals): reword a step\n\naiwf-entity:\n",
			cliutil.ExitFindings,
		},
		{
			"an untouched ritual tree leaves an ordinary commit alone",
			[]string{"internal/cli/check/check.go"},
			"refactor(check): tidy\n",
			cliutil.ExitOK,
		},
		{
			// A consumer repo has no such tree, so nothing it stages matches.
			"nothing staged at all",
			nil,
			"chore: an empty commit\n",
			cliutil.ExitOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := stagedRepo(t, tc.paths...)
			var buf bytes.Buffer
			code := runCommitMsg(writeMsg(t, tc.msg), root, verbs, &buf)
			if code != tc.want {
				t.Errorf("code = %d, want %d; stderr = %q", code, tc.want, buf.String())
			}
			if tc.want == cliutil.ExitFindings && !strings.Contains(buf.String(), "SKILL.md") {
				t.Errorf("refusal does not name the staged surface; stderr = %q", buf.String())
			}
		})
	}
}

// TestRunCommitMsg_ShippedSurfaceGuardIgnoresAnUnreadableIndex pins the
// contract when the guard cannot read the index at all: it states nothing.
// A hook that refused every commit it could not inspect would be worse than
// the omission it looks for, and the CI backstop still judges the commit.
func TestRunCommitMsg_ShippedSurfaceGuardIgnoresAnUnreadableIndex(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	// A bare temp dir is not a git repository, so `git diff --cached` fails.
	code := runCommitMsg(writeMsg(t, "docs(rituals): reword a step\n"), t.TempDir(),
		map[string]struct{}{"promote": {}}, &buf)
	if code != cliutil.ExitOK {
		t.Errorf("code = %d, want %d; stderr = %q", code, cliutil.ExitOK, buf.String())
	}
}

// TestRunCommitMsg_LeavesGitComposedAutosquashSubjectsAlone pins that the AC
// guard does not judge a subject git wrote. `git commit --fixup` copies the
// target commit's subject verbatim, so the scope in it is that commit's claim;
// refusing here would break `rebase --autosquash`, whose pairing depends on the
// subject matching exactly.
func TestRunCommitMsg_LeavesGitComposedAutosquashSubjectsAlone(t *testing.T) {
	t.Parallel()
	for _, prefix := range []string{"fixup! ", "squash! ", "amend! "} {
		t.Run(strings.TrimSpace(prefix), func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			msg := prefix + "feat(x): the thing (M-0001/AC-1)\n"
			if code := runCommitMsg(writeMsg(t, msg), t.TempDir(), map[string]struct{}{"promote": {}}, &buf); code != cliutil.ExitOK {
				t.Errorf("%q: code = %d, want %d; stderr = %q", prefix, code, cliutil.ExitOK, buf.String())
			}
		})
	}
}

// TestRunCommitMsg_ReportsTheHiddenBlockBeforeItsSymptoms pins the guard order.
// A message carrying both faults has one cause: the block git cannot read. The
// AC guard sees no entity trailer and would report that the subject's claim is
// unbacked — untrue of the message the operator is looking at, and its
// remediation produces a second refusal.
func TestRunCommitMsg_ReportsTheHiddenBlockBeforeItsSymptoms(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	msg := "feat(x): the thing (M-0001/AC-1)\n\naiwf-entity: M-0001/AC-1\naiwf-actor: human/peter\n\nCo-Authored-By: A <a@example.com>\n"
	code := runCommitMsg(writeMsg(t, msg), t.TempDir(), map[string]struct{}{"promote": {}}, &buf)
	if code != cliutil.ExitFindings {
		t.Fatalf("code = %d, want %d", code, cliutil.ExitFindings)
	}
	if !strings.Contains(buf.String(), "last paragraph") {
		t.Errorf("reported a symptom, not the cause; stderr = %q", buf.String())
	}
	if strings.Contains(buf.String(), "carrying no aiwf-entity trailer") {
		t.Errorf("reported the subject as unbacked when its trailer is present but hidden; stderr = %q", buf.String())
	}
}
