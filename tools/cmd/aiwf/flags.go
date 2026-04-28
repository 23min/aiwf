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
// The function is conservative: a token is treated as a flag only when
// it starts with `--` or `-` AND its name is in knownFlags. The
// `--name=value` form is hoisted as one token; the `--name value` form
// is hoisted as two. Unknown flags fall through to the original
// position so flag.Parse can produce its usual error.
func reorderFlagsFirst(args, knownFlags []string) []string {
	known := make(map[string]bool, len(knownFlags))
	for _, k := range knownFlags {
		known[k] = true
	}
	var hoisted, rest []string
	i := 0
	for i < len(args) {
		a := args[i]
		name, hasValue := flagName(a)
		if name != "" && known[name] {
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
			continue
		}
		rest = append(rest, a)
		i++
	}
	return append(hoisted, rest...)
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
