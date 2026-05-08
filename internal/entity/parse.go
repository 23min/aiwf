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
	fm, _, ok := Split(content)
	if !ok {
		return nil, ErrNoFrontmatter
	}
	var e Entity
	dec := yaml.NewDecoder(bytes.NewReader(fm))
	dec.KnownFields(true)
	if err := dec.Decode(&e); err != nil {
		return nil, fmt.Errorf("decoding frontmatter: %w", err)
	}
	e.Path = path
	return &e, nil
}
