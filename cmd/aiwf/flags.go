package main

import "strings"

// reorderFlagsFirst hoists known flags (and their values) to the front
// of args so Go's stdlib `flag` package — which stops parsing at the
// first non-flag token — accepts the natural CLI shape:
//
//	aiwf cancel M-001 --reason "..."     (flag after positional)
//
// Without this, the user is forced to write:
//
//	aiwf cancel --reason "..." M-001     (flags first)
//
// which is technically correct but goes against everyone's habits.
//
// knownFlags lists value-taking flags (`--name value`); knownBoolFlags
// lists boolean flags that do NOT consume a following token. A bool
// flag wrapped in `--name=value` form is recognized either way. When
// the same name appears in both lists, the value-taking interpretation
// wins (defensive — callers should not duplicate).
//
// The function is conservative: a token is treated as a flag only when
// it starts with `--` or `-` AND its name is in one of the known sets.
// Unknown flags fall through to the original position so flag.Parse
// can produce its usual error.
func reorderFlagsFirst(args, knownFlags, knownBoolFlags []string) []string {
	known := make(map[string]bool, len(knownFlags))
	for _, k := range knownFlags {
		known[k] = true
	}
	knownBool := make(map[string]bool, len(knownBoolFlags))
	for _, k := range knownBoolFlags {
		if !known[k] {
			knownBool[k] = true
		}
	}
	var hoisted, rest []string
	i := 0
	for i < len(args) {
		a := args[i]
		name, hasValue := flagName(a)
		switch {
		case name != "" && known[name]:
			if hasValue {
				hoisted = append(hoisted, a)
				i++
				continue
			}
			// `--name value` form: take the next token as the value.
			if i+1 < len(args) {
				hoisted = append(hoisted, a, args[i+1])
				i += 2
				continue
			}
			// Trailing flag without a value — let flag.Parse complain.
			hoisted = append(hoisted, a)
			i++
		case name != "" && knownBool[name]:
			// Bool flags never consume a following token.
			hoisted = append(hoisted, a)
			i++
		default:
			rest = append(rest, a)
			i++
		}
	}
	return append(hoisted, rest...)
}

// repeatedString implements flag.Value for a flag that may appear
// multiple times on the command line, accumulating each value into
// a slice. Used by `aiwf add ac --title "..." --title "..."` so a
// single invocation can create N acceptance criteria atomically
// (M-057). The Set method returns nil for empty input so the
// `--title ""` corner case is caught downstream by the verb's
// title-shape validation, not silently dropped here.
type repeatedString []string

// Set is the flag.Value contract — called once per flag occurrence.
func (r *repeatedString) Set(v string) error {
	*r = append(*r, v)
	return nil
}

// String renders the accumulated values for `--help` and error
// output. Comma-separated keeps the diagnostic readable when a
// batch is large.
func (r *repeatedString) String() string {
	if r == nil {
		return ""
	}
	return strings.Join(*r, ", ")
}

// flagName extracts the name from a CLI arg that looks like a flag.
// Returns ("", false) when the arg isn't flag-shaped. Returns
// (name, true) when the arg is `--name=value` (so the caller knows
// the value is bundled). Returns (name, false) for `--name` or `-name`.
func flagName(arg string) (name string, hasValue bool) {
	switch {
	case strings.HasPrefix(arg, "--"):
		arg = arg[2:]
	case strings.HasPrefix(arg, "-") && len(arg) > 1:
		arg = arg[1:]
	default:
		return "", false
	}
	if eq := strings.IndexByte(arg, '='); eq >= 0 {
		return arg[:eq], true
	}
	return arg, false
}
