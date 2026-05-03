package gitops

import (
	"fmt"
	"sort"
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

// recognizedTestKeys is the closed set of keys the kernel accepts on
// the write boundary. ParseStrictTestMetrics rejects anything else.
// Keep in sync with the table in I3 plan §4.
var recognizedTestKeys = map[string]struct{}{
	"pass":  {},
	"fail":  {},
	"skip":  {},
	"total": {},
}

// ParseStrictTestMetrics parses an aiwf-tests trailer value with
// write-strict semantics. Unknown keys, malformed `key=value` shapes,
// non-integer values, and negative values all return errors with a
// usage-shaped message.
//
// This is the validator the CLI uses on the `--tests` flag boundary.
// Read-side parsing is separately tolerant (see ParseTestMetrics) so
// future format extensions don't break old binaries reading new
// commits.
//
// Empty input returns a zero TestMetrics with no error — callers that
// require at least one recognized key should check the result for
// emptiness.
func ParseStrictTestMetrics(value string) (TestMetrics, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return TestMetrics{}, nil
	}
	var m TestMetrics
	for _, tok := range strings.Fields(v) {
		eq := strings.IndexByte(tok, '=')
		if eq <= 0 || eq == len(tok)-1 {
			return TestMetrics{}, fmt.Errorf("--tests: token %q is not key=value", tok)
		}
		key, raw := tok[:eq], tok[eq+1:]
		if _, ok := recognizedTestKeys[key]; !ok {
			sorted := sortedKeys(recognizedTestKeys)
			return TestMetrics{}, fmt.Errorf("--tests: unknown key %q (allowed: %s)", key, strings.Join(sorted, ", "))
		}
		n, err := strconv.Atoi(raw)
		if err != nil {
			return TestMetrics{}, fmt.Errorf("--tests: %s=%q must be a non-negative integer", key, raw)
		}
		if n < 0 {
			return TestMetrics{}, fmt.Errorf("--tests: %s=%d must be non-negative", key, n)
		}
		switch key {
		case "pass":
			m.Pass = n
		case "fail":
			m.Fail = n
		case "skip":
			m.Skip = n
		case "total":
			m.Total = n
		}
	}
	return m, nil
}

// FormatTestMetrics writes the canonical on-wire form of m. Order is
// pass / fail / skip / total; total is omitted when zero. Always emits
// `key=value` pairs separated by single spaces, matching the recipe in
// the I3 plan §4. Returns the empty string for a zero-value
// TestMetrics — callers should not write the trailer at all in that
// case.
func FormatTestMetrics(m TestMetrics) string {
	if m == (TestMetrics{}) {
		return ""
	}
	parts := []string{
		fmt.Sprintf("pass=%d", m.Pass),
		fmt.Sprintf("fail=%d", m.Fail),
		fmt.Sprintf("skip=%d", m.Skip),
	}
	if m.Total > 0 {
		parts = append(parts, fmt.Sprintf("total=%d", m.Total))
	}
	return strings.Join(parts, " ")
}

// sortedKeys returns the keys of a string-keyed set sorted
// alphabetically. Used to format deterministic error messages.
func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
