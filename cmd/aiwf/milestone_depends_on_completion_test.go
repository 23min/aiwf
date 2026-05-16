package main

import (
	"testing"

	"github.com/spf13/cobra"
)

// M-076/AC-5: closed-set completion for the new `--depends-on` flag
// on `aiwf add` and the `--on` flag on `aiwf milestone depends-on`,
// plus the positional milestone-id arg on the new verb.
//
// The generic drift tests (TestPolicy_FlagsHaveCompletion,
// TestPolicy_PositionalsHaveCompletion) catch the absence of any
// completion wiring; this file pins the *specific* surfaces M-076
// adds, so a future refactor that drops the wiring fails with a
// named, M-076-specific message.

// findCommand locates a sub-command by command-path. Returns nil if no
// command at that path is found.
func findCommand(root *cobra.Command, path string) *cobra.Command {
	var found *cobra.Command
	walkCommands(root, func(c *cobra.Command) {
		if c.CommandPath() == path {
			found = c
		}
	})
	return found
}

// TestMilestoneDependsOn_AddFlagCompletion pins the --depends-on flag
// completion on `aiwf add`.
func TestMilestoneDependsOn_AddFlagCompletion(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	addCmd := findCommand(root, "aiwf add")
	if addCmd == nil {
		t.Fatal("aiwf add command not found")
	}
	if !hasFlagCompletion(addCmd, "depends-on") {
		t.Error("aiwf add: --depends-on flag is missing a completion function (M-076/AC-5)")
	}
}

// TestMilestoneDependsOn_VerbFlagCompletion pins the --on flag
// completion on `aiwf milestone depends-on`.
func TestMilestoneDependsOn_VerbFlagCompletion(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	verbCmd := findCommand(root, "aiwf milestone depends-on")
	if verbCmd == nil {
		t.Fatal("aiwf milestone depends-on command not found")
	}
	if !hasFlagCompletion(verbCmd, "on") {
		t.Error("aiwf milestone depends-on: --on flag is missing a completion function (M-076/AC-5)")
	}
}

// TestMilestoneDependsOn_PositionalCompletion pins the positional
// milestone-id completion on `aiwf milestone depends-on`.
func TestMilestoneDependsOn_PositionalCompletion(t *testing.T) {
	t.Parallel()
	root := newRootCmd()
	verbCmd := findCommand(root, "aiwf milestone depends-on")
	if verbCmd == nil {
		t.Fatal("aiwf milestone depends-on command not found")
	}
	if verbCmd.ValidArgsFunction == nil {
		t.Error("aiwf milestone depends-on: positional <milestone-id> has no ValidArgsFunction (M-076/AC-5)")
	}
}
