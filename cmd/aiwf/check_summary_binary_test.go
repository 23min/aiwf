package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// M-0089: binary-level integration tests for `aiwf check` per-code
// summary by default + --verbose fallback.
//
// These tests build the actual binary and run it against a frozen
// fixture so the whole pipeline (loader → checks → render) is
// exercised end-to-end. AC-3 ("--verbose reproduces the current
// behavior byte-for-byte") and AC-4 ("JSON envelope is unchanged")
// are byte-equal assertions against goldens captured from the
// pre-M-0089 binary; AC-7 ("kernel-tree integration test — default
// output is short") is a structural assertion against this repo's
// own planning tree.
//
// The fixture is a temp copy of internal/check/testdata/messy so the
// test runs outside any .git context (provenance audit and
// trunk-collision checks are skipped accordingly). Goldens live at
// cmd/aiwf/testdata/m0089/.

// fixtureCheckSource is the source tree that the M-0089 binary tests
// copy into a tempdir before invoking the binary. messy is rich
// enough — multiple distinct error codes with single instances each,
// and multiple distinct warning codes with multi-instance presence —
// to exercise both the per-instance-errors path (AC-2) and the
// summary-warnings path (AC-1) in one fixture.
const fixtureCheckSource = "../../internal/check/testdata/messy"

// TestBinary_CheckVerbose_ByteIdenticalToBaseline (M-0089 AC-3):
// `aiwf check --verbose` against the frozen fixture must reproduce
// the pre-M-0089 binary's output byte-for-byte. The golden was
// captured from the binary built at SHA 5523e99 (the milestone's
// base commit); a drift means the verbose path or one of its
// dependencies changed shape, which is exactly what AC-3 forbids.
func TestBinary_CheckVerbose_ByteIdenticalToBaseline(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp)
	fixture := copyFixture(t, fixtureCheckSource, tmp)

	out, err := runBinaryStdout(bin, "check", "--verbose", "--root", fixture)
	if err != nil && !exitedWithCode(err, 1) {
		t.Fatalf("aiwf check --verbose: %v\n%s", err, out)
	}

	want := readGolden(t, "testdata/m0089/verbose-text.golden")
	if out != want {
		writeActualForDiff(t, "verbose-text.actual", out)
		t.Errorf("`aiwf check --verbose` drifted from baseline (AC-3 violation).\n\nlen(got)=%d len(want)=%d\nfirst diff at line %d:\n%s",
			len(out), len(want), firstDiffLine(want, out), diffSnippet(want, out))
	}
}

// TestBinary_CheckJSON_ByteIdenticalToBaseline (M-0089 AC-4):
// `aiwf check --format=json` (with or without --verbose) must
// produce an envelope structurally identical to the pre-M-0089
// baseline. JSON consumers depend on the full per-finding shape;
// the summary collapse is a text-only concern.
//
// Strict byte-equality is impossible across runs because
// metadata.root is the absolute path of the resolved consumer
// repo, which legitimately varies (tempdir per test, /tmp/ in
// the baseline capture). The assertion is therefore "byte-identical
// modulo metadata.root" — every other field, including the entire
// findings array, must match exactly. Per CLAUDE.md "Substring
// assertions are not structural assertions", the comparison parses
// both envelopes and compares parsed structures, not raw bytes.
func TestBinary_CheckJSON_ByteIdenticalToBaseline(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp)
	fixture := copyFixture(t, fixtureCheckSource, tmp)

	want := parseEnvelope(t, readGolden(t, "testdata/m0089/json.golden"))
	for _, flags := range [][]string{
		{"check", "--format=json", "--root", fixture},
		{"check", "--format=json", "--verbose", "--root", fixture},
	} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			out, err := runBinaryStdout(bin, flags...)
			if err != nil && !exitedWithCode(err, 1) {
				t.Fatalf("aiwf %v: %v\n%s", flags, err, out)
			}
			got := parseEnvelope(t, out)
			assertEnvelopesEqualModuloRoot(t, want, got, out)
		})
	}
}

// TestBinary_CheckJSONPretty_ByteIdenticalToBaseline (M-0089 AC-4):
// the --pretty branch of JSON output also stays structurally
// identical. The renderer adds indentation, which has its own
// newline/whitespace shape; a structural compare via JSON parsing
// covers both compact and pretty paths uniformly. Bytes outside
// `metadata.root` are guaranteed identical because the underlying
// encoder is the stdlib's deterministic shape.
func TestBinary_CheckJSONPretty_ByteIdenticalToBaseline(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp)
	fixture := copyFixture(t, fixtureCheckSource, tmp)

	out, err := runBinaryStdout(bin, "check", "--format=json", "--pretty", "--root", fixture)
	if err != nil && !exitedWithCode(err, 1) {
		t.Fatalf("aiwf check --format=json --pretty: %v\n%s", err, out)
	}

	want := parseEnvelope(t, readGolden(t, "testdata/m0089/json-pretty.golden"))
	got := parseEnvelope(t, out)
	assertEnvelopesEqualModuloRoot(t, want, got, out)

	// Pretty mode adds indentation; verify the rendered bytes
	// actually contain the newlines + 2-space indent the renderer
	// claims to produce. (A regression that drops --pretty silently
	// would still parse to the same structure as compact mode.)
	if !strings.Contains(out, "\n  \"tool\":") {
		t.Errorf("--pretty output is not indented:\n%s", out)
	}
}

// TestBinary_CheckDefault_SummarizesWarnings (M-0089 AC-1, AC-2):
// `aiwf check` (no flags) against the same fixture produces:
//   - errors per-instance (one line per finding)
//   - warnings collapsed into per-code summary lines
//   - footer with raw instance totals
//
// Assertions are structural per AC-8: extract code tokens via the
// documented summary-line shape, do not grep flat.
func TestBinary_CheckDefault_SummarizesWarnings(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp)
	fixture := copyFixture(t, fixtureCheckSource, tmp)

	out, err := runBinaryStdout(bin, "check", "--root", fixture)
	if err != nil && !exitedWithCode(err, 1) {
		t.Fatalf("aiwf check: %v\n%s", err, out)
	}

	// Structural parse: collect summary lines (warning codes) and
	// per-instance error lines.
	summaries := parseSummaryLines(t, out)
	errorPerInstance := parseErrorPerInstanceLines(out)

	// AC-2: 10 per-instance error lines (the fixture's error count).
	if len(errorPerInstance) != 10 {
		t.Errorf("want 10 per-instance error lines, got %d", len(errorPerInstance))
	}

	// AC-1: warning-code summary count for the messy fixture is 6.
	// The kernel test for the fixture pins this: adr-supersession-mutual,
	// archive-sweep-pending, epic-active-no-drafted-milestones (added
	// by M-0094 alongside the E-02-no-drafts fixture entity),
	// gap-resolved-has-resolver, terminal-entity-not-archived,
	// titles-nonempty.
	wantCodes := map[string]int{
		"adr-supersession-mutual":           2,
		"archive-sweep-pending":             1,
		"epic-active-no-drafted-milestones": 1,
		"gap-resolved-has-resolver":         1,
		"terminal-entity-not-archived":      3,
		"titles-nonempty":                   1,
	}
	gotCodes := map[string]int{}
	for _, s := range summaries {
		if s.severity != "warning" {
			t.Errorf("default mode collapsed a non-warning into summary form: %+v", s)
			continue
		}
		gotCodes[s.code] = s.count
	}
	for code, want := range wantCodes {
		if gotCodes[code] != want {
			t.Errorf("code %q: want count %d, got %d", code, want, gotCodes[code])
		}
	}
	for code := range gotCodes {
		if _, ok := wantCodes[code]; !ok {
			t.Errorf("unexpected summary code in default output: %q", code)
		}
	}

	// AC-1 ordering pin: count desc, alphabetic tie-break.
	// Expected order: terminal-entity-not-archived (3),
	// adr-supersession-mutual (2), archive-sweep-pending (1),
	// epic-active-no-drafted-milestones (1), gap-resolved-has-resolver (1),
	// titles-nonempty (1).
	wantOrder := []string{
		"terminal-entity-not-archived",
		"adr-supersession-mutual",
		"archive-sweep-pending",
		"epic-active-no-drafted-milestones",
		"gap-resolved-has-resolver",
		"titles-nonempty",
	}
	if len(summaries) != len(wantOrder) {
		t.Fatalf("want %d summary lines, got %d", len(wantOrder), len(summaries))
	}
	for i, code := range wantOrder {
		if summaries[i].code != code {
			t.Errorf("summary[%d] code = %q, want %q (count-desc / alphabetic tie-break)", i, summaries[i].code, code)
		}
	}

	// Footer: instance counts shift by +1 warning with M-0094's
	// fixture addition (E-02-no-drafts) — 10 errors + 9 warnings = 19.
	if !strings.Contains(out, "19 findings (10 errors, 9 warnings)") {
		t.Errorf("default-mode footer missing or wrong:\n%s", out)
	}
}

// TestBinary_CheckDefault_KernelTreeShortOutput (M-0089 AC-7):
// run the binary against this repo's actual planning tree and assert
// the default-mode output is short (≤10 lines). The kernel tree
// currently has only a couple of distinct warning codes generating
// nearly all the noise, so the summary collapses ~180+ findings into
// a handful of lines. The exact count is environmental (depends on
// how many distinct codes are present), so the assertion is a bound,
// not a fixed value.
//
// Per CLAUDE.md "Render output must be human-verified before the
// iteration closes": this test pins the *bound*; the human pass at
// wrap also reads the actual output and confirms it scans cleanly.
// The two checks are complementary, not redundant.
func TestBinary_CheckDefault_KernelTreeShortOutput(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp)
	repo := repoRootForTest(t)

	out, err := runBinaryStdout(bin, "check", "--root", repo)
	if err != nil && !exitedWithCode(err, 1) {
		t.Fatalf("aiwf check (kernel tree): %v\n%s", err, out)
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) > 10 {
		t.Errorf("kernel-tree default output should be ≤10 lines, got %d (AC-7 violation):\n%s",
			len(lines), out)
	}

	// Sanity: at least one summary line is present and the footer line
	// names the total finding count. If neither holds, something
	// structural broke and the size bound is moot.
	summaries := parseSummaryLines(t, out)
	if len(summaries) == 0 {
		t.Errorf("expected at least one summary line; got none:\n%s", out)
	}
	if !regexp.MustCompile(`\d+ findings \(\d+ errors, \d+ warnings\)`).MatchString(out) {
		t.Errorf("footer line missing:\n%s", out)
	}
}

// TestBinary_CheckHelp_DocumentsVerbose (M-0089 AC-5):
// `aiwf check --help` names the --verbose flag with a one-line
// description, and the Example block shows the default vs.
// --verbose invocation contrast.
func TestBinary_CheckHelp_DocumentsVerbose(t *testing.T) {
	skipIfShortOrUnsupported(t)
	tmp := t.TempDir()
	bin := buildBinary(t, tmp)

	// Cobra writes --help output to stderr, not stdout, so use the
	// combined-output helper.
	out, err := runBinaryCombined(bin, "check", "--help")
	if err != nil {
		t.Fatalf("aiwf check --help: %v\n%s", err, out)
	}

	// Parse `--help` output structurally: the flags block is a series
	// of lines matching `^      --<name> ... <description>`. Find the
	// --verbose row and assert its description is non-empty. A flat
	// substring match for "verbose" would fire on any unrelated
	// mention; structural parsing pins the flag-row position.
	flagRow := regexp.MustCompile(`^\s+--verbose\b\s*(.*)$`)
	var verboseDesc string
	for _, ln := range strings.Split(out, "\n") {
		if m := flagRow.FindStringSubmatch(ln); m != nil {
			verboseDesc = strings.TrimSpace(m[1])
			break
		}
	}
	if verboseDesc == "" {
		t.Errorf("--verbose flag missing from `aiwf check --help` flags block:\n%s", out)
	}

	// The Example block must contrast default + --verbose invocations
	// so a reader sees both shapes named together. Parse the Examples
	// section structurally: lines starting with two spaces + "aiwf check"
	// that contain "--verbose" prove the example invocation is present.
	hasDefaultExample := false
	hasVerboseExample := false
	for _, ln := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(ln)
		if !strings.HasPrefix(trimmed, "aiwf check") {
			continue
		}
		if strings.Contains(trimmed, "--verbose") {
			hasVerboseExample = true
			continue
		}
		// "aiwf check" by itself (no --format=json) is the default-shape example.
		if !strings.Contains(trimmed, "--format") {
			hasDefaultExample = true
		}
	}
	if !hasDefaultExample {
		t.Errorf("Examples block missing a bare `aiwf check` invocation (default-mode shape):\n%s", out)
	}
	if !hasVerboseExample {
		t.Errorf("Examples block missing a `aiwf check --verbose` invocation:\n%s", out)
	}
}

// summaryLineParsed is the local twin of internal/render's summary
// line shape; the regex is mirrored here so the binary test doesn't
// import internal/render.
type summaryLineParsed struct {
	code     string
	severity string
	count    int
	sample   string
}

// summaryLineRE matches the per-code summary line format produced by
// render.TextSummary: `<code> (<severity>) × N — <message>`. Anchored
// at start-of-line so per-instance lines (which have a path: prefix)
// cannot accidentally match.
var summaryLineRE = regexp.MustCompile(`^(\S+) \((warning|error)\) × (\d+) — (.+)$`)

func parseSummaryLines(t *testing.T, out string) []summaryLineParsed {
	t.Helper()
	var lines []summaryLineParsed
	for _, ln := range strings.Split(out, "\n") {
		m := summaryLineRE.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[3])
		if err != nil {
			t.Fatalf("parsing count from %q: %v", ln, err)
		}
		lines = append(lines, summaryLineParsed{code: m[1], severity: m[2], count: n, sample: m[4]})
	}
	return lines
}

// errorLineRE matches per-instance error lines:
// `<path>:<line>: error <code>[/<subcode>]: <msg>`. The leading path
// segment is what distinguishes a per-instance line from a summary
// line (which has no path prefix).
var errorLineRE = regexp.MustCompile(`^(\S+):(\d+): error (\S+): (.+)$`)

func parseErrorPerInstanceLines(out string) []string {
	var lines []string
	for _, ln := range strings.Split(out, "\n") {
		if errorLineRE.MatchString(ln) {
			lines = append(lines, ln)
		}
	}
	return lines
}

// runBinaryStdout invokes bin with args and returns stdout only.
// stderr is silently dropped (the existing runBinary helper combines
// stdout+stderr, which corrupts byte-equal goldens with stray
// warnings like "--pretty has no effect" the binary writes to stderr).
func runBinaryStdout(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), err
}

// copyFixture copies srcRel (relative to the test's package dir)
// into dst as a sibling subdir named like the source's basename, and
// returns the destination path. Used to give the binary an isolated
// working tree per test run (so checks don't pick up the test
// harness's git context).
func copyFixture(t *testing.T, srcRel, dst string) string {
	t.Helper()
	src, err := filepath.Abs(srcRel)
	if err != nil {
		t.Fatalf("abs src: %v", err)
	}
	out := filepath.Join(dst, filepath.Base(src))
	if err := copyTree(src, out); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	return out
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

func readGolden(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(rel)
	if err != nil {
		t.Fatalf("read golden %s: %v", rel, err)
	}
	return string(b)
}

// writeActualForDiff dumps the actual output to a sibling file under
// the test's tempdir so a human can inspect it after a golden-diff
// failure (Go's default failure output truncates large strings).
func writeActualForDiff(t *testing.T, name, got string) {
	t.Helper()
	out := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(out, []byte(got), 0o644); err != nil {
		t.Logf("could not write %s: %v", out, err)
		return
	}
	t.Logf("actual output saved at %s", out)
}

// firstDiffLine returns the 1-based line number where want and got
// first diverge. Returns 0 if the strings are equal.
func firstDiffLine(want, got string) int {
	w := strings.Split(want, "\n")
	g := strings.Split(got, "\n")
	for i := 0; i < len(w) && i < len(g); i++ {
		if w[i] != g[i] {
			return i + 1
		}
	}
	if len(w) != len(g) {
		return min(len(w), len(g)) + 1
	}
	return 0
}

// diffSnippet returns a small window of lines around the first
// divergence — enough context for a human to see what changed
// without dumping the full file.
func diffSnippet(want, got string) string {
	line := firstDiffLine(want, got)
	if line == 0 {
		return "(no diff)"
	}
	w := strings.Split(want, "\n")
	g := strings.Split(got, "\n")
	start := line - 2
	if start < 1 {
		start = 1
	}
	var buf bytes.Buffer
	buf.WriteString("--- want\n")
	for i := start; i <= line+1 && i-1 < len(w); i++ {
		buf.WriteString("  ")
		buf.WriteString(w[i-1])
		buf.WriteString("\n")
	}
	buf.WriteString("+++ got\n")
	for i := start; i <= line+1 && i-1 < len(g); i++ {
		buf.WriteString("  ")
		buf.WriteString(g[i-1])
		buf.WriteString("\n")
	}
	return buf.String()
}

// parseEnvelope decodes one JSON envelope (the shape `aiwf check
// --format=json` writes). Used by AC-4 to compare structurally
// rather than byte-equal.
func parseEnvelope(t *testing.T, raw string) map[string]any {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatalf("decode envelope: %v\nraw:\n%s", err, raw)
	}
	return env
}

// assertEnvelopesEqualModuloRoot compares two envelopes for AC-4.
// Everything in `findings` and the top-level envelope keys must match
// exactly; `metadata.root` is permitted to vary (legitimately
// environmental — the absolute path of the resolved consumer repo).
//
// Per AC-4 in M-0089: "byte-identical to a saved pre-M-0089 baseline".
// The literal reading is impossible for the metadata.root slot, so
// the assertion compares everything else exactly and asserts that
// metadata.root is non-empty (the field is present and populated,
// just not pinnable in a portable golden).
func assertEnvelopesEqualModuloRoot(t *testing.T, want, got map[string]any, rawGot string) {
	t.Helper()
	wantClone := cloneEnvelope(want)
	gotClone := cloneEnvelope(got)

	// AC-4: metadata.root is environmental; pin presence + non-empty
	// but ignore the value when comparing structurally.
	if md, ok := gotClone["metadata"].(map[string]any); ok {
		root, hasRoot := md["root"].(string)
		if !hasRoot || root == "" {
			t.Errorf("metadata.root missing or empty:\n%s", rawGot)
		}
		md["root"] = "<env-specific>"
		gotClone["metadata"] = md
	}
	if md, ok := wantClone["metadata"].(map[string]any); ok {
		md["root"] = "<env-specific>"
		wantClone["metadata"] = md
	}

	if diff := cmp.Diff(wantClone, gotClone); diff != "" {
		writeActualForDiff(t, "json.actual", rawGot)
		t.Errorf("JSON envelope drifted from baseline (AC-4 violation):\n%s", diff)
	}
}

func cloneEnvelope(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch tv := v.(type) {
		case map[string]any:
			out[k] = cloneEnvelope(tv)
		default:
			out[k] = v
		}
	}
	return out
}

// runBinaryCombined invokes bin with args and returns both stdout
// and stderr merged. Used by tests that read text Cobra writes to
// stderr (notably --help output).
func runBinaryCombined(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}
