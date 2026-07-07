package cliutil

import (
	"fmt"
	"os"
)

// Errorf, Errorln, Printf, Println, and Print are the sanctioned raw-
// stdio writers for operator-facing CLI text outside OutputFormat's
// text-mode branch (outputformat.go) — the forbidigo chokepoint (ADR-0017
// AC-3) bans every other bare fmt.Print*/Fprint* call so a stray print
// can't reintroduce an unaccounted-for output path. Each wrapper is a
// pure pass-through to the same stdlib call it replaces: no behavior
// change, same stream, same bytes.

// Errorf writes a formatted line to stderr.
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}

// Errorln writes args to stderr, space-joined with a trailing newline.
func Errorln(args ...any) {
	fmt.Fprintln(os.Stderr, args...)
}

// Printf writes a formatted line to stdout.
func Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stdout, format, args...)
}

// Println writes args to stdout, space-joined with a trailing newline.
func Println(args ...any) {
	fmt.Println(args...)
}

// Print writes args to stdout with no added newline.
func Print(args ...any) {
	fmt.Print(args...)
}
