// Package recipe owns the embedded-and-installable validator
// recipes that aiwf ships out of the box.
//
// A recipe is a single markdown file with a YAML frontmatter block
// that declares a validator. The frontmatter is the validator block
// the install verb appends to aiwf.yaml.contracts.validators; the
// markdown body is documentation served by `aiwf contract recipe
// show`.
//
// Frontmatter shape:
//
//	---
//	name: <validator-name>
//	command: <executable>
//	args:
//	  - <argv elements, may include {{schema}} {{fixture}} {{contract_id}} {{version}}>
//	---
//
// Two recipes ship in I1 — `cue` and `jsonschema`. New recipes are
// added by dropping a markdown file into embedded/ and rebuilding;
// no engine code change is required.
package recipe

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/23min/aiwf/internal/aiwfyaml"
)

//go:embed embedded
var embedFS embed.FS

// Recipe is the parsed, ready-to-install form of an embedded recipe.
// Validator carries the bytes that will land in
// aiwf.yaml.contracts.validators[Name]; Markdown is the full
// documentation including the frontmatter (so `recipe show` can
// reproduce the file as authored).
type Recipe struct {
	Name      string
	Validator aiwfyaml.Validator
	Markdown  []byte
}

// ErrNotFound is returned when a requested recipe name is not in the
// embedded set.
var ErrNotFound = errors.New("recipe not found")

// List returns every embedded recipe in name-sorted order. The byte
// content is freshly read from the embed each call.
func List() ([]Recipe, error) {
	entries, err := fs.ReadDir(embedFS, "embedded")
	if err != nil {
		return nil, fmt.Errorf("reading embedded recipes: %w", err)
	}
	out := make([]Recipe, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		raw, err := fs.ReadFile(embedFS, filepath.ToSlash(filepath.Join("embedded", name)))
		if err != nil {
			return nil, fmt.Errorf("reading embedded recipe %s: %w", name, err)
		}
		r, err := parseMarkdown(name, raw)
		if err != nil {
			return nil, fmt.Errorf("recipe %s: %w", name, err)
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Get returns the embedded recipe named name, or ErrNotFound if no
// embedded recipe carries that name in its frontmatter. The lookup
// is by frontmatter `name:`, not filename — though the convention is
// for them to match.
func Get(name string) (Recipe, error) {
	all, err := List()
	if err != nil {
		return Recipe{}, err
	}
	for _, r := range all {
		if r.Name == name {
			return r, nil
		}
	}
	return Recipe{}, fmt.Errorf("%w: %q (try 'aiwf contract recipes' for the shipped set)", ErrNotFound, name)
}

// ParseFile reads a custom-validator file (the `--from <path>` shape)
// and returns the validator block. The file is plain YAML with the
// same fields as a recipe's frontmatter; the markdown body is not
// part of the contract for `--from <path>`.
func ParseFile(path string) (Recipe, error) {
	raw, err := fs.ReadFile(osFS{}, path)
	if err != nil {
		return Recipe{}, fmt.Errorf("reading %s: %w", path, err)
	}
	return parseValidatorYAML(path, raw)
}

// parseMarkdown extracts the frontmatter from raw and returns a
// fully-formed Recipe. Errors out on a missing frontmatter delimiter
// or on a frontmatter that doesn't carry the required fields.
func parseMarkdown(filename string, raw []byte) (Recipe, error) {
	fm, _, err := splitFrontmatter(raw)
	if err != nil {
		return Recipe{}, fmt.Errorf("%s: %w", filename, err)
	}
	r, err := parseValidatorYAML(filename, fm)
	if err != nil {
		return Recipe{}, err
	}
	r.Markdown = raw
	return r, nil
}

// parseValidatorYAML decodes a YAML byte slice into a Recipe, with
// strict KnownFields handling so unknown keys surface as errors.
// Used by both the embedded-markdown frontmatter parser and the
// custom-validator `--from <path>` parser.
func parseValidatorYAML(filename string, raw []byte) (Recipe, error) {
	type rawRecipe struct {
		Name    string   `yaml:"name"`
		Command string   `yaml:"command"`
		Args    []string `yaml:"args"`
	}
	var rr rawRecipe
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&rr); err != nil {
		return Recipe{}, fmt.Errorf("%s: parsing validator block: %w", filename, err)
	}
	if rr.Name == "" {
		return Recipe{}, fmt.Errorf("%s: missing required field `name`", filename)
	}
	if rr.Command == "" {
		return Recipe{}, fmt.Errorf("%s: missing required field `command`", filename)
	}
	return Recipe{
		Name: rr.Name,
		Validator: aiwfyaml.Validator{
			Command: rr.Command,
			Args:    append([]string(nil), rr.Args...),
		},
	}, nil
}

// splitFrontmatter slices raw into the YAML frontmatter bytes and the
// markdown body bytes. Mirrors entity.Split's leniency: BOM-tolerant,
// CRLF-tolerant, expects the document to begin with `---\n` and to
// have a closing `---` on its own line.
func splitFrontmatter(raw []byte) (frontmatter, body []byte, err error) {
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	if !bytes.HasPrefix(raw, []byte("---\n")) && !bytes.HasPrefix(raw, []byte("---\r\n")) {
		return nil, nil, errors.New("missing YAML frontmatter (recipe must begin with `---`)")
	}
	nl := bytes.IndexByte(raw, '\n') + 1
	rest := raw[nl:]

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
			return fm.Bytes(), rest, nil
		}
		fm.Write(line)
		if idx < 0 {
			return nil, nil, errors.New("unterminated YAML frontmatter (no closing `---`)")
		}
		fm.WriteByte('\n')
	}
}

// osFS adapts the os package to fs.FS for ParseFile. Pulled out so
// the rest of the package can be tested without disk IO.
type osFS struct{}

// Open opens the named file from the host filesystem; satisfies fs.FS.
func (osFS) Open(name string) (fs.File, error) { return openOSFile(name) }
