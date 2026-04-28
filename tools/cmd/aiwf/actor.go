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

// resolveActor picks the actor string for a verb's commit trailer.
// Precedence: explicit > aiwf.yaml > git config user.email derivation.
// Returns an error when none yields a valid value or when the explicit
// value is malformed.
func resolveActor(explicit, root string) (string, error) {
	if explicit != "" {
		if !actorPattern.MatchString(explicit) {
			return "", fmt.Errorf("--actor %q must match <role>/<identifier> (single '/', no whitespace)", explicit)
		}
		return explicit, nil
	}
	if root != "" {
		cfg, err := config.Load(root)
		switch {
		case err == nil:
			if cfg.Actor != "" {
				return cfg.Actor, nil
			}
		case errors.Is(err, config.ErrNotFound):
			// fall through to git config derivation
		default:
			return "", err
		}
	}
	out, err := exec.Command("git", "config", "user.email").Output()
	if err == nil {
		email := strings.TrimSpace(string(out))
		if at := strings.IndexByte(email, '@'); at > 0 {
			candidate := "human/" + email[:at]
			if actorPattern.MatchString(candidate) {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("no actor: pass --actor <role>/<identifier>, run `aiwf init`, or set git config user.email")
}
