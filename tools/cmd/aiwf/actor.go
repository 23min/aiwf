package main

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/config"
)

// actorPattern enforces the Q10 format: <role>/<identifier> with
// exactly one forward slash and no whitespace. Both sides must be
// non-empty and themselves slash-free.
var actorPattern = regexp.MustCompile(`^[^\s/]+/[^\s/]+$`)

// Actor source labels surfaced by `aiwf whoami` and used as the second
// return value of resolveActorWithSource. Stable strings; do not change
// without updating tests and documentation.
const (
	actorSourceFlag      = "--actor flag"
	actorSourceConfig    = "aiwf.yaml"
	actorSourceGitConfig = "git config user.email"
)

// resolveActor picks the actor string for a verb's commit trailer.
// Precedence: explicit > aiwf.yaml > git config user.email derivation.
// Returns an error when none yields a valid value or when the explicit
// value is malformed.
func resolveActor(explicit, root string) (string, error) {
	actor, _, err := resolveActorWithSource(explicit, root)
	return actor, err
}

// resolveActorWithSource is resolveActor plus the human-readable label
// of which source produced the value. Used by `aiwf whoami` to explain
// the precedence outcome to the user.
func resolveActorWithSource(explicit, root string) (actor, source string, err error) {
	if explicit != "" {
		if !actorPattern.MatchString(explicit) {
			return "", "", fmt.Errorf("--actor %q must match <role>/<identifier> (single '/', no whitespace)", explicit)
		}
		return explicit, actorSourceFlag, nil
	}
	if root != "" {
		cfg, cfgErr := config.Load(root)
		switch {
		case cfgErr == nil:
			if cfg.Actor != "" {
				return cfg.Actor, actorSourceConfig, nil
			}
		case errors.Is(cfgErr, config.ErrNotFound):
			// fall through to git config derivation
		default:
			return "", "", cfgErr
		}
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
	return "", "", fmt.Errorf("no actor: pass --actor <role>/<identifier>, run `aiwf init`, or set git config user.email")
}
