package gitops

import (
	"strconv"
	"strings"
)

// TestMetrics is the parsed payload of an aiwf-tests trailer. The four
// integer counts (Pass / Fail / Skip / Total) are the recognized keys
// in the I3 plan §4. Total is optional in the on-wire format and is
// derivable from Pass+Fail+Skip; readers that need it should call
// TotalOrDerive rather than reading Total directly.
type TestMetrics struct {
	Pass  int `json:"pass"`
	Fail  int `json:"fail"`
	Skip  int `json:"skip"`
	Total int `json:"total,omitempty"`
}

// TotalOrDerive returns the on-wire Total when present (>0), otherwise
// Pass+Fail+Skip. Useful for renderers that want a denominator without
// caring whether the writer recorded it.
func (m TestMetrics) TotalOrDerive() int {
	if m.Total > 0 {
		return m.Total
	}
	return m.Pass + m.Fail + m.Skip
}

// ParseTestMetrics parses an aiwf-tests trailer value of the form
// `pass=12 fail=0 skip=0 total=12`. Tokens are whitespace-separated.
// Recognized keys: pass, fail, skip, total. Unknown keys are silently
// ignored (forward compat — future trailer extensions don't break old
// readers). Malformed tokens (non-`key=value` shape, non-integer
// values, negative values) are skipped without erroring; callers get
// the metrics that did parse.
//
// Returns ok=true when at least one recognized key produced a value;
// ok=false when the input was empty after trim or contained no
// recognized keys. Read-side parser by design — write-time validation
// (which rejects malformed input) lives on the verb boundary that
// constructs the trailer.
func ParseTestMetrics(value string) (TestMetrics, bool) {
	v := strings.TrimSpace(value)
	if v == "" {
		return TestMetrics{}, false
	}
	var m TestMetrics
	var seen bool
	for _, tok := range strings.Fields(v) {
		eq := strings.IndexByte(tok, '=')
		if eq <= 0 || eq == len(tok)-1 {
			continue
		}
		key, raw := tok[:eq], tok[eq+1:]
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			continue
		}
		switch key {
		case "pass":
			m.Pass = n
			seen = true
		case "fail":
			m.Fail = n
			seen = true
		case "skip":
			m.Skip = n
			seen = true
		case "total":
			m.Total = n
			seen = true
		}
	}
	return m, seen
}
