package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// actorPattern enforces the Q10 format: <role>/<identifier> with
// exactly one forward slash and no whitespace. Both sides must be
// non-empty and themselves slash-free.
var actorPattern = regexp.MustCompile(`^[^\s/]+/[^\s/]+$`)

// Actor source labels surfaced by `aiwf whoami` and used as the second
// return value of resolveActorWithSource. Stable strings; do not change
// without updating tests and documentation.
//
// The pre-I2.5 `aiwf.yaml` source is gone — identity is now runtime-
// derived per `provenance-model.md`, with `--actor` overriding the
// git-config default. The aiwf.yaml `actor:` key (if still present)
// is ignored for resolution; `aiwf doctor` surfaces a deprecation note.
const (
	actorSourceFlag      = "--actor flag"
	actorSourceGitConfig = "git config user.email"
)

// resolveActor picks the actor string for a verb's commit trailer.
// Precedence: explicit `--actor` > git config user.email derivation.
// Returns an error when neither yields a valid value or when the
// explicit value is malformed.
//
// The root parameter is unused but kept for call-site compatibility;
// future per-repo identity policy (if it ever lands) would consult it.
func resolveActor(explicit, root string) (string, error) {
	actor, _, err := resolveActorWithSource(explicit, root)
	return actor, err
}

// resolveActorWithSource is resolveActor plus the human-readable label
// of which source produced the value. Used by `aiwf whoami` to explain
// the precedence outcome to the user.
func resolveActorWithSource(explicit, root string) (actor, source string, err error) {
	_ = root // reserved for future per-repo identity policy
	if explicit != "" {
		if !actorPattern.MatchString(explicit) {
			return "", "", fmt.Errorf("--actor %q must match <role>/<identifier> (single '/', no whitespace)", explicit)
		}
		return explicit, actorSourceFlag, nil
	}
	out, gitErr := exec.Command("git", "config", "user.email").Output()
	if gitErr == nil {
		email := strings.TrimSpace(string(out))
		if at := strings.IndexByte(email, '@'); at > 0 {
			candidate := "human/" + email[:at]
			if actorPattern.MatchString(candidate) {
				return candidate, actorSourceGitConfig, nil
			}
		}
	}
	return "", "", fmt.Errorf("no actor: pass --actor <role>/<identifier> or set git config user.email")
}
