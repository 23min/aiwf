package aiwfyaml

import "unicode/utf8"

// yamlLines retains separators and follows yaml.v3's is_break/skip_line rules,
// so byte ranges agree with the parser's one-based node line numbers.
func yamlLines(raw []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, r := range string(raw) {
		switch r {
		case '\r':
			if i+1 < len(raw) && raw[i+1] == '\n' {
				continue
			}
		case '\n', '\u0085', '\u2028', '\u2029':
		default:
			continue
		}
		end := i + utf8.RuneLen(r)
		lines = append(lines, raw[start:end])
		start = end
	}
	return append(lines, raw[start:])
}
