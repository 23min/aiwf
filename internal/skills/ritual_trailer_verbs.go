package skills

import (
	"fmt"
	"regexp"
	"sync"
)

// ritualTrailerVerbStampRE matches the literal
// `--trailer "aiwf-verb: <verb>"` shape that ritual skill bodies use
// to instruct an aiwfx-* / wf-* ritual to stamp a non-kernel verb on a
// commit. Anything that is not a literal --trailer stamp (prose
// mentions like `commits with aiwf-verb: promote`) is intentionally
// excluded — the allowlist's purpose is to recognize *what the rituals
// actually stamp*, not what they describe.
var ritualTrailerVerbStampRE = regexp.MustCompile(`--trailer "aiwf-verb:\s*([a-z][a-z0-9-]*)"`)

var (
	ritualTrailerVerbsOnce sync.Once
	ritualTrailerVerbsSet  map[string]struct{}
	ritualTrailerVerbsErr  error
)

// RitualTrailerVerbs returns the closed set of non-kernel ritual verbs
// that embedded ritual skills stamp via `--trailer "aiwf-verb: <verb>"`.
// The set is derived once at first call from the embedded skill
// snapshot under `embedded-rituals/` and cached for the binary's
// lifetime — it changes only when the embedded snapshot changes (i.e.
// when `make sync-rituals` lands a new upstream pin).
//
// G-0190 introduced this derivation to eliminate the drift class where
// the upstream stamp set changes and a kernel allowlist silently goes
// stale (or starts flagging legitimate ritual stamps). Callers (e.g.
// `internal/check`'s trailer-verb-unknown rule) consume this set
// directly instead of holding a parallel hand-maintained map.
func RitualTrailerVerbs() (map[string]struct{}, error) {
	ritualTrailerVerbsOnce.Do(func() {
		ritualTrailerVerbsSet, ritualTrailerVerbsErr = extractRitualTrailerVerbs()
	})
	return ritualTrailerVerbsSet, ritualTrailerVerbsErr
}

func extractRitualTrailerVerbs() (map[string]struct{}, error) {
	rits, err := ListRituals()
	if err != nil {
		return nil, fmt.Errorf("listing embedded rituals: %w", err)
	}
	out := make(map[string]struct{})
	for _, s := range rits {
		matches := ritualTrailerVerbStampRE.FindAllSubmatch(s.Content, -1)
		for _, m := range matches {
			if len(m) >= 2 {
				out[string(m[1])] = struct{}{}
			}
		}
	}
	return out, nil
}
