package policies

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/entity"
)

// A kind with no draft phase is created together with its body: for exactly
// the kinds entity.IsBornComplete names, `aiwf add` refuses a create whose
// required sections are empty, which every create without a body flag is. A
// shipped surface that spells such a create hands its reader a command that
// exits 2, so none may.
//
// Only code is read — inline code spans, and fenced lines with their backslash
// continuations joined — because code is what a reader runs. A bare
// `aiwf add gap`, with nothing after the kind, names the verb rather than
// spelling a command. `--body` and `--body-file` share the prefix matched here.

// addSpelling is one create as a shipped file spells it.
type addSpelling struct {
	file    string
	command string
}

// refusedAddSpellingExemptions holds the spellings that show the refused create
// on purpose, each with the reason it stays.
var refusedAddSpellingExemptions = map[addSpelling]string{
	{"internal/skills/embedded/aiwf-add/SKILL.md", `aiwf add gap --title "Retry loop spins forever"`}: "a transcript of the refusal itself, printed with its error",
	{"internal/skills/embedded/aiwf-add/SKILL.md", `aiwf add gap --title ""`}:                         "an example acceptance criterion whose pass criterion is the refusal",
}

var (
	addInvocation = regexp.MustCompile("aiwf add ([^\\s`]+)")
	inlineCode    = regexp.MustCompile("`[^`\n]+`")
)

// codeFragments returns the parts of a markdown document a reader would run:
// each inline code span outside a fence, and each fenced line with any
// backslash continuation joined onto it.
func codeFragments(doc string) []string {
	var out []string
	lines := strings.Split(doc, "\n")
	inFence := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if !inFence {
			for _, span := range inlineCode.FindAllString(line, -1) {
				out = append(out, strings.Trim(span, "`"))
			}
			continue
		}
		for strings.HasSuffix(strings.TrimRight(line, " "), `\`) && i+1 < len(lines) {
			i++
			line = strings.TrimRight(strings.TrimSuffix(strings.TrimRight(line, " "), `\`), " ") + " " + strings.TrimSpace(lines[i])
		}
		out = append(out, line)
	}
	return out
}

// bodylessBornCompleteAdds returns, as spelled, each create a document spells
// for a born-complete kind without a body flag. A command ends at the next
// backtick, which closes a span quoted inside a fenced markdown example. A kind
// list such as `gap|decision` counts when any kind in it is born complete, and
// a request for `--help` creates nothing.
func bodylessBornCompleteAdds(doc string) []string {
	var out []string
	for _, frag := range codeFragments(doc) {
		for _, m := range addInvocation.FindAllStringSubmatchIndex(frag, -1) {
			rest := frag[m[1]:]
			if end := strings.IndexByte(rest, '`'); end >= 0 {
				rest = rest[:end]
			}
			if next := addInvocation.FindStringIndex(rest); next != nil {
				rest = rest[:next[0]]
			}
			if strings.TrimSpace(rest) == "" || strings.Contains(rest, "--body") || strings.Contains(rest, "--help") {
				continue
			}
			for kind := range strings.SplitSeq(frag[m[2]:m[3]], "|") {
				if entity.IsBornComplete(entity.Kind(kind)) {
					out = append(out, strings.TrimSpace(frag[m[0]:m[1]]+rest))
					break
				}
			}
		}
	}
	return out
}

func TestBodylessBornCompleteAdds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		doc  string
		want []string
	}{
		{
			name: "a bare mention names the verb",
			doc:  "Reach for `aiwf add gap` when a defect surfaces.",
		},
		{
			name: "a create spelled without a body is reported",
			doc:  "Run `aiwf add adr --title \"Pick a cache\"` first.",
			want: []string{`aiwf add adr --title "Pick a cache"`},
		},
		{
			name: "continuation lines join into one command",
			doc:  "```bash\naiwf add contract \\\n  --title \"Op spec\" \\\n  --linked-adr x\n```\n",
			want: []string{`aiwf add contract --title "Op spec" --linked-adr x`},
		},
		{
			name: "an indented fence is still a fence",
			doc:  "- Step one:\n\n   ```bash\n   aiwf add gap --title \"Indented\"\n   ```\n",
			want: []string{`aiwf add gap --title "Indented"`},
		},
		{
			name: "a body flag on a continuation line satisfies it",
			doc:  "```bash\naiwf add contract \\\n  --title \"Op spec\" \\\n  --body-file op.md\n```\n",
		},
		{
			name: "a kind with a draft phase is never reported",
			doc:  "`aiwf add epic --title \"Billing rework\"`",
		},
		{
			name: "prose outside code is not read",
			doc:  "```bash\naiwf add epic --title x\n```\nthen aiwf add gap --title x in running prose",
		},
		{
			name: "every code span on a line is read",
			doc:  "Invoke `aiwfx-record-gap` (`aiwf add gap --title \"second\"`).",
			want: []string{`aiwf add gap --title "second"`},
		},
		{
			name: "a request for help creates nothing",
			doc:  "Flags are listed by `aiwf add gap --help`.",
		},
		{
			name: "a command quoted inside a fenced example ends at its backtick",
			doc:  "```markdown\n**Pass criterion**: `aiwf add gap --title \"\"` exits 2.\n```\n",
			want: []string{`aiwf add gap --title ""`},
		},
		{
			name: "a fenced line is read whole",
			doc:  "```bash\naiwf add gap --title \"Retry spins\"\n```\n",
			want: []string{`aiwf add gap --title "Retry spins"`},
		},
		{
			name: "each aiwf add on a line is judged by its own flags",
			doc:  "```bash\naiwf add gap --title \"x\" && aiwf add epic --title \"y\" --body-file e.md\n```\n",
			want: []string{`aiwf add gap --title "x" &&`},
		},
		{
			name: "a kind list is reported once when any kind in it is born complete",
			doc:  "`aiwf add epic|gap|decision|milestone --priority high`",
			want: []string{"aiwf add epic|gap|decision|milestone --priority high"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if diff := cmp.Diff(tc.want, bodylessBornCompleteAdds(tc.doc)); diff != "" {
				t.Errorf("bodylessBornCompleteAdds mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// bodylessAddReports walks the shipped markdown under root and reports each
// bodyless born-complete create that no exemption covers, and each exemption
// whose spelling its file no longer carries, so the list cannot outlive what it
// excuses.
func bodylessAddReports(root string, exemptions map[addSpelling]string) ([]string, error) {
	seen := map[addSpelling]bool{}
	var reports []string
	err := walkShippedMarkdown(root, shippedSurfaceRoots, func(rel, content string) {
		for _, command := range bodylessBornCompleteAdds(content) {
			spelling := addSpelling{rel, command}
			if _, exempt := exemptions[spelling]; exempt {
				seen[spelling] = true
				continue
			}
			reports = append(reports, bodylessAddReport(rel, command))
		}
	})
	if err != nil {
		return nil, err
	}
	for spelling := range exemptions {
		if !seen[spelling] {
			reports = append(reports, fmt.Sprintf("refusedAddSpellingExemptions lists %q in %s, which no longer "+
				"spells it; drop the entry", spelling.command, spelling.file))
		}
	}
	sort.Strings(reports)
	return reports, nil
}

func bodylessAddReport(rel, command string) string {
	return fmt.Sprintf("%s spells %q, which `aiwf add` refuses: the kind is created with its body, so the command "+
		"carries --body-file or --body. If showing the refusal is the point, add the spelling to "+
		"refusedAddSpellingExemptions with the reason", rel, command)
}

func TestShippedSurfaces_SpellBornCompleteCreatesWithABody(t *testing.T) {
	t.Parallel()
	reports, err := bodylessAddReports(repoRoot(t), refusedAddSpellingExemptions)
	if err != nil {
		t.Fatal(err)
	}
	for _, report := range reports {
		t.Error(report)
	}
}

func TestBodylessAddReports_WalksEveryShippedMarkdownFileAgainstItsExemptions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	var want []string
	for i, dir := range shippedSurfaceRoots {
		rel := filepath.ToSlash(filepath.Join(dir, "nested", "bare.md"))
		command := fmt.Sprintf(`aiwf add decision --title "bare %d"`, i)
		writeAt(t, root, rel, "Run `"+command+"`.\n")
		want = append(want, bodylessAddReport(rel, command))
	}
	kept := filepath.ToSlash(filepath.Join(shippedSurfaceRoots[0], "kept.md"))
	writeAt(t, root, kept, "Run `aiwf add gap --title \"kept\"`, not `aiwf add gap --title \"other\"`.\n")
	want = append(want, bodylessAddReport(kept, `aiwf add gap --title "other"`))
	elsewhere := filepath.ToSlash(filepath.Join(shippedSurfaceRoots[1], "elsewhere.md"))
	writeAt(t, root, elsewhere, "Run `aiwf add gap --title \"kept\"`.\n")
	want = append(want, bodylessAddReport(elsewhere, `aiwf add gap --title "kept"`))
	writeAt(t, root, filepath.ToSlash(filepath.Join(shippedSurfaceRoots[0], "notes.txt")), "Run `aiwf add adr --title \"text\"`.\n")
	exemptions := map[addSpelling]string{
		{kept, `aiwf add gap --title "kept"`}: "still spelled",
		{kept, `aiwf add adr --title "gone"`}: "no longer spelled",
	}
	want = append(want, fmt.Sprintf("refusedAddSpellingExemptions lists %q in %s, which no longer spells it; drop the entry",
		`aiwf add adr --title "gone"`, kept))
	sort.Strings(want)
	reports, err := bodylessAddReports(root, exemptions)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, reports); diff != "" {
		t.Errorf("bodylessAddReports mismatch (-want +got):\n%s", diff)
	}
}

// bodylessExampleReports holds each command's `--help` Example to the same
// rule, across the whole tree under root: an Example is a command a reader runs
// as printed, so every line of it is read.
func bodylessExampleReports(root *cobra.Command) []string {
	var reports []string
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, command := range bodylessBornCompleteAdds("```\n" + c.Example + "\n```\n") {
			reports = append(reports, fmt.Sprintf("`%s --help` shows %q, which `aiwf add` refuses: the kind is "+
				"created with its body, so the example carries --body-file or --body", c.CommandPath(), command))
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
	return reports
}

func TestCommandExamples_SpellBornCompleteCreatesWithABody(t *testing.T) {
	t.Parallel()
	for _, report := range bodylessExampleReports(cli.NewRootCmd("")) {
		t.Error(report)
	}
}

func TestBodylessExampleReports_ReadsEveryExampleInTheTree(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "tool", Example: "  aiwf add adr --title \"root\""}
	parent := &cobra.Command{Use: "group", Example: "  aiwf add decision --title \"group\""}
	parent.AddCommand(&cobra.Command{
		Use:     "leaf",
		Long:    "Prose that mentions aiwf add contract --title unread.",
		Example: "  aiwf add gap --title \"leaf\"",
		Run:     func(*cobra.Command, []string) {},
	})
	root.AddCommand(parent)
	report := func(path, command string) string {
		return fmt.Sprintf("`%s --help` shows %q, which `aiwf add` refuses: the kind is created with its body, so the "+
			"example carries --body-file or --body", path, command)
	}
	want := []string{
		report("tool", `aiwf add adr --title "root"`),
		report("tool group", `aiwf add decision --title "group"`),
		report("tool group leaf", `aiwf add gap --title "leaf"`),
	}
	if diff := cmp.Diff(want, bodylessExampleReports(root)); diff != "" {
		t.Errorf("bodylessExampleReports mismatch (-want +got):\n%s", diff)
	}
}
