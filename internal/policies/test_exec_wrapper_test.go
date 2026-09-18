package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// test_exec_wrapper_test.go — G-0693.
//
// `go test -exec=<prog>` sits outside the flag set `go help test` defines
// as cacheable, so naming a program costs every run the full suite. The
// program named here is the ad-hoc signing wrapper, and an unsigned
// Mach-O test binary only crashes syspolicyd on macOS (G-0128/G-0133) —
// off Darwin `scripts/sign-and-run.sh` is a bare `exec "$@"`. So every
// go-test recipe passes the flag, and TEST_EXEC resolves it to the
// wrapper on Darwin and to empty elsewhere.
//
// Both arms are judged on whatever host runs this, by putting a `uname`
// on PATH that reports the platform under test: `$(shell uname)` is the
// only thing the Makefile consults. A pin that judged the running host
// alone would assert nothing about Darwin on the Linux this project
// gates on — leaving the signing half, which is why the wrapper exists,
// resting on a branch no run reaches.
//
// A candidate target may be named anywhere; `make -n` decides which ones
// actually run `go test`, so a recipe holding the command in a variable
// or a canned block is judged on what make expands rather than on how
// the Makefile spells it. Asking make rather than reading the recipe has
// consequences worth knowing: a target that merely echoes the words `go
// test` is selected and then fails for passing no flag, loudly and by
// name; the `.PHONY` names are the recovery path for a target the rule
// pattern cannot spell, such as a variable-named one, rather than
// redundancy to trim; and a target whose `make -n` errors fails this
// test rather than one about itself.
var (
	execFlagValue = regexp.MustCompile(`-exec[= ](\S*)`)
	goTestCall    = regexp.MustCompile(`\bgo\s+test\b`)
)

// unamePATH returns a PATH entry whose `uname` reports goos, so a recipe
// can be expanded for a platform other than the one running the test.
func unamePATH(t *testing.T, goos string) string {
	t.Helper()
	dir := t.TempDir()
	if err := testsupport.WriteExecutable(filepath.Join(dir, "uname"), []byte("#!/bin/sh\necho "+goos+"\n")); err != nil {
		t.Fatalf("writing uname shim: %v", err)
	}
	return "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")
}

// makefileTargets returns every name that could be a target: the ones
// declared .PHONY and the ones written as a rule. Over-broad on purpose
// — a name whose recipe runs no `go test` is dropped by the caller.
func makefileTargets(t *testing.T, root string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("reading Makefile: %v", err)
	}
	// A rule line, not a variable assignment: `foo:` and `foo: dep`
	// qualify, `FOO := bar` does not.
	ruleLine := regexp.MustCompile(`^([A-Za-z0-9_./-]+)\s*:(?:[^=]|$)`)

	seen := map[string]bool{}
	var names []string
	add := func(name string) {
		if name == "" || strings.HasPrefix(name, ".") || seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "\t") {
			continue
		}
		if rest, ok := strings.CutPrefix(line, ".PHONY:"); ok {
			for _, name := range strings.Fields(rest) {
				add(name)
			}
			continue
		}
		if m := ruleLine.FindStringSubmatch(line); m != nil {
			add(m[1])
		}
	}
	if len(names) == 0 {
		t.Fatal("no Makefile target names were found; the scan is broken")
	}
	return names
}

func TestMakefile_SigningWrapperNamedOnlyOnDarwin(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	darwinPATH, elsewherePATH := unamePATH(t, "Darwin"), unamePATH(t, "Linux")

	// `make -n` is the oracle for which targets run `go test`: it
	// expands variables and canned recipes, so the answer does not
	// depend on how the command is spelled in the Makefile.
	type target struct{ name, darwin, elsewhere string }
	var targets []target
	for _, name := range makefileTargets(t, root) {
		recipe := makeDryRun(t, root, name, darwinPATH)
		if !goTestCall.MatchString(recipe) {
			continue
		}
		targets = append(targets, target{name, recipe, makeDryRun(t, root, name, elsewherePATH)})
	}
	if len(targets) == 0 {
		t.Fatal("no Makefile target runs `go test`")
	}

	for _, tgt := range targets {
		t.Run(tgt.name, func(t *testing.T) {
			t.Parallel()
			assertExecValues(t, tgt.name, "Darwin", tgt.darwin, func(v string) string {
				if !strings.HasSuffix(v, "/scripts/sign-and-run.sh") {
					return "must name the signing wrapper: an unsigned Mach-O test binary crashes syspolicyd (G-0133)"
				}
				return ""
			})
			assertExecValues(t, tgt.name, "Linux", tgt.elsewhere, func(v string) string {
				if v != "" {
					return "must pass an empty value: the wrapper is a no-op off Darwin, and naming a program defeats the test cache (G-0693)"
				}
				return ""
			})
		})
	}
}

// assertExecValues checks every -exec value the recipe passes, and that
// it passes one at all — a recipe that dropped the flag would run a
// Darwin test binary unsigned, which no value check would notice.
func assertExecValues(t *testing.T, target, goos, recipe string, reject func(string) string) {
	t.Helper()
	values := execFlagValue.FindAllStringSubmatch(recipe, -1)
	if len(values) == 0 {
		t.Fatalf("`make %s` on %s passes no -exec; every go-test recipe routes through TEST_EXEC:\n%s", target, goos, recipe)
	}
	for _, v := range values {
		if why := reject(v[1]); why != "" {
			t.Errorf("`make %s` on %s passes -exec=%q; it %s", target, goos, v[1], why)
		}
	}
}
