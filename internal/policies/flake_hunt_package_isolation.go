package policies

import (
	"os"
	"path/filepath"
	"strings"
)

// PolicyFlakeHuntPackageIsolation asserts that the only `...` package
// pattern in the flake-hunt workflow is the `go list` that builds its
// matrix.
//
// flake-hunt is the gate that decides whether a tag is safe to push, so
// its red has to mean "a regression", not "the runner was busy". What
// fails this repo's subprocess-bearing packages is co-tenancy rather
// than machine size: alone they pass every repeat, and sharing cores
// with a broad sweep they report scheduling delay as a defect. The
// workflow therefore fans out over a `go list`-derived matrix, one
// package per runner, and a wildcard anywhere else puts those packages
// back together on one runner — whether it lands on the test
// invocation or in a hand-written matrix (G-0438).
//
// Scanning is over logical lines, so a wildcard on a backslash
// continuation belongs to the command it continues. The enumerator's
// exemption is withdrawn on a line that also runs `go test`, since the
// shipped enumerator is itself a continued block and appending a sweep
// to it is the cheapest way to reach the shape this bans.
//
// That a race line exists in this file at all, capped at `-parallel 8`,
// is PolicyRaceParallelCap's claim and is not restated here.
func PolicyFlakeHuntPackageIsolation(root string) ([]Violation, error) {
	rel := filepath.Join(".github", "workflows", "flake-hunt.yml")
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return nil, err
	}

	var vs []Violation
	for _, ln := range logicalLines(string(b)) {
		// Comments describe the workflow rather than running it, so a
		// `./...` in prose is not a sweep.
		if isComment(ln.Text) || !carriesEllipsis(ln.Text) {
			continue
		}
		// The enumerator is the one command whose job is to name every
		// package; the matrix it feeds is what splits them up again.
		if strings.Contains(ln.Text, "go list") && !strings.Contains(ln.Text, "go test") {
			continue
		}
		vs = append(vs, Violation{
			Policy: "flake-hunt-package-isolation",
			File:   rel,
			Line:   ln.Num,
			Detail: "only the `go list` matrix enumerator may carry a `...` pattern — flake-hunt runs one package per runner so contention can't read as a defect (G-0438)",
		})
	}
	return vs, nil
}

// carriesEllipsis reports whether the line contains `...`. It is a
// substring test, not a parse: in this workflow every `...` is a
// package pattern (`./...`, `./internal/...`,
// `github.com/23min/aiwf/...`), which is what makes the cheap test
// sufficient. Anything else carrying the token fails loudly and says
// why, which is the trade a ban is allowed to make.
func carriesEllipsis(line string) bool {
	return strings.Contains(line, "...")
}

// sourceLine is one logical line of a shell-bearing file, numbered from
// 1 by the line it starts on.
type sourceLine struct {
	Num  int
	Text string
}

// logicalLines joins backslash-continued lines, so an argument written
// on a continuation belongs to the command it continues rather than
// reading as a line of its own. A comment neither starts nor extends a
// join: `make` would continue one, but a policy that scans for real
// invocations is better off exposing the line after a commented
// backslash than swallowing it.
func logicalLines(content string) []sourceLine {
	var (
		out   []sourceLine
		buf   strings.Builder
		start int
	)
	flush := func(num int, text string) {
		out = append(out, sourceLine{Num: num, Text: text})
		buf.Reset()
	}
	for i, raw := range strings.Split(content, "\n") {
		if buf.Len() == 0 {
			start = i + 1
		}
		if isComment(raw) {
			// A comment mid-join terminates what it interrupted, so
			// neither line is lost.
			if buf.Len() > 0 {
				flush(start, buf.String())
			}
			flush(i+1, raw)
			continue
		}
		if trimmed := strings.TrimRight(raw, " \t"); strings.HasSuffix(trimmed, `\`) {
			buf.WriteString(strings.TrimRight(strings.TrimSuffix(trimmed, `\`), " \t"))
			buf.WriteString(" ")
			continue
		}
		buf.WriteString(raw)
		flush(start, buf.String())
	}
	// A file whose last line ends in a continuation has no terminator
	// to flush on.
	if buf.Len() > 0 {
		flush(start, buf.String())
	}
	return out
}

// isComment reports whether the line is Makefile / YAML comment prose
// rather than an invocation.
func isComment(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "#")
}
