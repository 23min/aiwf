package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/check"
	"github.com/23min/aiwf/internal/skills"
)

// stampedVerbTrailerRE matches a static `aiwf-verb:` trailer value the
// verb layer assembles as a `gitops.Trailer{Key: gitops.TrailerVerb,
// Value: "<literal>"}` struct literal. Dynamic values (`Value:
// verbName`, an identifier) carry no quotes and are intentionally not
// matched — their runtime value is pinned behaviorally by the
// per-verb integration tests (e.g. TestTrailerShape) rather than
// statically here.
var stampedVerbTrailerRE = regexp.MustCompile(`gitops\.TrailerVerb,\s*Value:\s*"([^"]+)"`)

type stampOffender struct{ where, value string }

// scanStampedVerbTrailers walks each source (display path -> content)
// for static aiwf-verb trailer stamps and returns those whose value is
// a member of neither registered nor ritual, plus whether any stamp
// literal was seen at all. Pure and injectable so the positive-control
// test below can drive the offender, ritual-match, and no-stamp paths
// without depending on the live verb tree.
func scanStampedVerbTrailers(sources map[string]string, registered, ritual map[string]struct{}) (offenders []stampOffender, sawAny bool) {
	paths := make([]string, 0, len(sources))
	for p := range sources {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, path := range paths {
		for i, line := range strings.Split(sources[path], "\n") {
			m := stampedVerbTrailerRE.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			sawAny = true
			value := m[1]
			if _, ok := registered[value]; ok {
				continue
			}
			if _, ok := ritual[value]; ok {
				continue
			}
			offenders = append(offenders, stampOffender{
				where: path + ":" + strconv.Itoa(i+1),
				value: value,
			})
		}
	}
	return offenders, sawAny
}

// TestStampedVerbTrailersAreRegistered pins the invariant that broke in
// G-0339: every `aiwf-verb:` trailer value a kernel verb stamps must be
// a member of the closed set the trailer-verb check derives from the
// live Cobra command tree (∪ the ritual-verb allowlist). When the two
// disagree — as they did for the `contract` subverbs, whose stamps
// dropped the `contract-` path segment — the verb's own single commit
// is refused by the `commit-msg` hook `aiwf init`/`update` installs,
// making the verb non-functional in any initialized consumer repo.
//
// The check enumerates the allowed set by hyphen-joining each command's
// full path (`contract recipe install` -> `contract-recipe-install`),
// so a stamp of `recipe-install` is absent from it. This test scans the
// verb-layer source for every statically-stamped trailer value and
// asserts membership, catching the whole class rather than the four
// verbs that first exhibited it.
func TestStampedVerbTrailersAreRegistered(t *testing.T) {
	t.Parallel()

	root := cli.NewRootCmd()
	registered := check.EnumerateRegisteredVerbs(root)
	if len(registered) == 0 {
		t.Fatal("EnumerateRegisteredVerbs returned an empty set; command tree wiring is broken")
	}
	ritual, err := skills.RitualTrailerVerbs()
	if err != nil {
		t.Fatalf("RitualTrailerVerbs: %v", err)
	}

	sources := readVerbSources(t)
	offenders, sawAny := scanStampedVerbTrailers(sources, registered, ritual)

	if !sawAny {
		t.Fatal("scanned internal/verb/*.go but matched no stamped aiwf-verb trailer literals; the scan regex or the verb layer changed shape")
	}
	if len(offenders) > 0 {
		t.Fatal(formatStampOffenders(offenders))
	}
}

// TestScanStampedVerbTrailers_positiveControl drives the scanner with
// synthetic sources so the offender-collection, ritual-allowlist, and
// no-stamp branches are exercised regardless of the (clean) live tree.
func TestScanStampedVerbTrailers_positiveControl(t *testing.T) {
	t.Parallel()

	registered := map[string]struct{}{"add": {}, "contract-bind": {}}
	ritual := map[string]struct{}{"wrap-milestone": {}}

	t.Run("flags unregistered stamp, passes registered and ritual", func(t *testing.T) {
		t.Parallel()
		sources := map[string]string{
			"internal/verb/good.go":   "\t\t{Key: gitops.TrailerVerb, Value: \"add\"},",
			"internal/verb/ritual.go": "\t\t{Key: gitops.TrailerVerb, Value: \"wrap-milestone\"},",
			"internal/verb/bad.go":    "line one\n\t\t{Key: gitops.TrailerVerb, Value: \"bind\"},",
			"internal/verb/dyn.go":    "\t\t{Key: gitops.TrailerVerb, Value: verbName},",
		}
		offenders, sawAny := scanStampedVerbTrailers(sources, registered, ritual)
		if !sawAny {
			t.Fatal("sawAny = false; expected the literal stamps to be seen")
		}
		want := []stampOffender{{where: "internal/verb/bad.go:2", value: "bind"}}
		if len(offenders) != 1 || offenders[0] != want[0] {
			t.Fatalf("offenders = %+v, want %+v", offenders, want)
		}
		// The rendered message names the offending path and value.
		msg := formatStampOffenders(offenders)
		if !strings.Contains(msg, "internal/verb/bad.go:2") || !strings.Contains(msg, "bind") {
			t.Fatalf("message %q missing offender path/value", msg)
		}
	})

	t.Run("no stamps yields sawAny false and no offenders", func(t *testing.T) {
		t.Parallel()
		offenders, sawAny := scanStampedVerbTrailers(
			map[string]string{"internal/verb/empty.go": "package verb\n// nothing to see"},
			registered, ritual)
		if sawAny {
			t.Fatal("sawAny = true; expected false for stamp-free source")
		}
		if len(offenders) != 0 {
			t.Fatalf("offenders = %+v, want none", offenders)
		}
	})
}

// readVerbSources returns internal/verb/*.go (excluding _test.go) keyed
// by a repo-relative display path.
func readVerbSources(t *testing.T) map[string]string {
	t.Helper()
	verbDir := filepath.Join(findKernelRoot(t), "internal", "verb")
	entries, err := os.ReadDir(verbDir)
	if err != nil {
		t.Fatalf("read %s: %v", verbDir, err)
	}
	out := make(map[string]string)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(verbDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		out[filepath.Join("internal", "verb", name)] = string(data)
	}
	return out
}

// formatStampOffenders renders the failure message: what broke, why it
// bites (the commit-msg hook refuses the verb's own commit), and the
// fix, followed by one line per offender.
func formatStampOffenders(offenders []stampOffender) string {
	sort.Slice(offenders, func(a, b int) bool { return offenders[a].where < offenders[b].where })
	var b strings.Builder
	b.WriteString("verb(s) stamp an aiwf-verb trailer value absent from the registered-verb set ∪ ritualVerbs;\n")
	b.WriteString("the commit-msg hook (aiwf init/update) will refuse the verb's own commit. Fixes: stamp the\n")
	b.WriteString("full hyphen-joined Cobra path (e.g. `contract-recipe-install`), or register the verb accordingly.\n")
	for _, o := range offenders {
		b.WriteString("  " + o.where + " stamps aiwf-verb: " + o.value + "\n")
	}
	return b.String()
}
