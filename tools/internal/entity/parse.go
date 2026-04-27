package entity

import (
	"bytes"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ErrNoFrontmatter is returned when the input does not contain a
// `---`-delimited YAML block at the top of the file.
var ErrNoFrontmatter = errors.New("no YAML frontmatter found")

// Parse decodes a markdown file's YAML frontmatter into an Entity. The
// body (everything after the closing `---`) is discarded; aiwf does not
// validate body prose. Path is recorded on the returned entity for
// downstream finding-context.
//
// Parse does not assign Entity.Kind. The tree loader resolves kind from
// the file's path before handing the entity to checks.
func Parse(path string, content []byte) (*Entity, error) {
	body, ok := extractFrontmatter(content)
	if !ok {
		return nil, ErrNoFrontmatter
	}
	var e Entity
	dec := yaml.NewDecoder(bytes.NewReader(body))
	dec.KnownFields(true)
	if err := dec.Decode(&e); err != nil {
		return nil, fmt.Errorf("decoding frontmatter: %w", err)
	}
	e.Path = path
	return &e, nil
}

// extractFrontmatter returns the YAML block between the opening and
// closing `---` lines. Returns false if the file does not begin with
// `---`, or if no closing delimiter is found.
//
// Tolerant of CRLF line endings and a leading UTF-8 BOM.
func extractFrontmatter(content []byte) ([]byte, bool) {
	content = bytes.TrimPrefix(content, []byte("\xef\xbb\xbf"))

	// First line must be exactly "---".
	if !bytes.HasPrefix(content, []byte("---\n")) && !bytes.HasPrefix(content, []byte("---\r\n")) {
		return nil, false
	}

	// Skip past the opening delimiter and its newline.
	nl := bytes.IndexByte(content, '\n') + 1
	rest := content[nl:]

	var fm bytes.Buffer
	for {
		idx := bytes.IndexByte(rest, '\n')
		var line []byte
		if idx < 0 {
			line = rest
			rest = nil
		} else {
			line = rest[:idx]
			rest = rest[idx+1:]
		}
		if bytes.Equal(bytes.TrimRight(line, "\r"), []byte("---")) {
			return fm.Bytes(), true
		}
		fm.Write(line)
		if idx < 0 {
			break
		}
		fm.WriteByte('\n')
	}
	return nil, false
}
