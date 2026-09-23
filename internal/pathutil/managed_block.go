package pathutil

import (
	"errors"
	"strings"
)

// ManagedBlockSpan locates one ordered pair of standalone markers. It returns
// (-1, -1) for an absent block, and excludes the end line's newline from the span.
func ManagedBlockSpan(content, startMarker, endMarker, prefix string) (startOffset, endOffset int, err error) {
	start, end, offset := -1, -1, 0
	for _, line := range strings.SplitAfter(content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case startMarker:
			if start >= 0 {
				return 0, 0, errors.New("duplicate START")
			}
			start = offset
		case endMarker:
			if end >= 0 {
				return 0, 0, errors.New("duplicate END")
			}
			end = offset + len(strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"))
		default:
			if strings.HasPrefix(trimmed, prefix) && strings.HasSuffix(trimmed, "-->") {
				return 0, 0, errors.New("unrecognized marker")
			}
		}
		offset += len(line)
	}
	if (start < 0) != (end < 0) || (start >= 0 && start >= end) {
		return 0, 0, errors.New("missing or reversed START/END")
	}
	return start, end, nil
}

// SpliceManagedBlock replaces a validated block while preserving bytes outside it.
func SpliceManagedBlock(content, body, startMarker, endMarker, prefix string) (string, error) {
	start, end, err := ManagedBlockSpan(content, startMarker, endMarker, prefix)
	if err != nil {
		return "", err
	}
	block := startMarker + "\n" + strings.TrimSuffix(body, "\n") + "\n" + endMarker
	if start >= 0 {
		return content[:start] + block + content[end:], nil
	}
	if content == "" {
		return block + "\n", nil
	}
	separator := "\n\n"
	if strings.HasSuffix(content, "\n") {
		separator = "\n"
	}
	return content + separator + block + "\n", nil
}
