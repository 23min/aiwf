package main

import "testing"

func TestNewRootCmd_HasRunAndComposeSubcommands(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	names := map[string]bool{}
	for _, c := range root.Commands() {
		names[c.Name()] = true
	}
	if !names["run"] {
		t.Fatal("expected root command to register a 'run' subcommand")
	}
	if !names["compose"] {
		t.Fatal("expected root command to register a 'compose' subcommand")
	}
}
