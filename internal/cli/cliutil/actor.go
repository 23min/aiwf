package cliutil

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
// return value of ResolveActorWithSource. Stable strings; do not change
// without updating tests and documentation.
//
// The pre-I2.5 `aiwf.yaml` source is gone — identity is now runtime-
// derived per `provenance-model.md`, with `--actor` overriding the
// git-config default. The aiwf.yaml `actor:` key (if still present)
// is ignored for resolution; `aiwf doctor` surfaces a deprecation note.
const (
	ActorSourceFlag      = "--actor flag"
	ActorSourceGitConfig = "git config user.email"
)

// ResolveActor picks the actor string for a verb's commit trailer.
// Precedence: explicit `--actor` > git config user.email derivation.
// Returns an error when neither yields a valid value or when the
// explicit value is malformed.
//
// Git config is read in root, including Git's usual global fallback. An empty
// root uses the current directory. Explicit actors do not require a repository.
func ResolveActor(explicit, root string) (string, error) {
	actor, _, err := ResolveActorWithSource(explicit, root)
	return actor, err
}

// ResolveActorWithSource is ResolveActor plus the human-readable label
// of which source produced the value. Used by `aiwf whoami` to explain
// the precedence outcome to the user.
func ResolveActorWithSource(explicit, root string) (actor, source string, err error) {
	if explicit != "" {
		if !actorPattern.MatchString(explicit) {
			return "", "", fmt.Errorf("--actor %q must match <role>/<identifier> (single '/', no whitespace)", explicit)
		}
		return explicit, ActorSourceFlag, nil
	}
	cmd := exec.Command("git", "config", "user.email")
	cmd.Dir = root
	out, gitErr := cmd.Output()
	if gitErr == nil {
		email := strings.TrimSpace(string(out))
		if at := strings.IndexByte(email, '@'); at > 0 {
			candidate := "human/" + email[:at]
			if actorPattern.MatchString(candidate) {
				return candidate, ActorSourceGitConfig, nil
			}
		}
	}
	return "", "", fmt.Errorf("no actor: pass --actor <role>/<identifier> or set git config user.email")
}
