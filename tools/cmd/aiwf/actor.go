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

// resolveActor picks the actor string for a verb's commit trailer.
// Precedence: explicit > git config user.email derivation. Returns an
// error when neither is available or when the explicit value is
// malformed.
//
// aiwf.yaml's `actor` field is *not* consulted here in Session 2 — the
// config loader lands in Session 3 alongside `aiwf init`. Until then,
// pass --actor explicitly or have git config set.
func resolveActor(explicit string) (string, error) {
	if explicit != "" {
		if !actorPattern.MatchString(explicit) {
			return "", fmt.Errorf("--actor %q must match <role>/<identifier> (single '/', no whitespace)", explicit)
		}
		return explicit, nil
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
	return "", fmt.Errorf("no actor: pass --actor <role>/<identifier> or set git config user.email")
}
