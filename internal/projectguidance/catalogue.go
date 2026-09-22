// Package projectguidance retrieves the external engineering guidance corpus.
package projectguidance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/23min/aiwf/internal/config"
)

// ErrInvalidSource identifies a source that cannot supply valid selected guidance.
var ErrInvalidSource = errors.New("invalid guidance source")

// Catalogue describes every pack available from one source revision.
type Catalogue struct {
	Packs []Pack `json:"packs"`
}

// Pack describes applicability and ordered, repository-relative documents.
type Pack struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
	Detect      []string `json:"detect"`
}

func parseCatalogue(raw []byte) (Catalogue, error) {
	var wire struct {
		Packs []json.RawMessage `json:"packs"`
	}
	if !utf8.Valid(raw) {
		return Catalogue{}, fmt.Errorf("%w: catalogue.json must be UTF-8", ErrInvalidSource)
	}
	if err := decodeObject(raw, &wire, []string{"packs"}); err != nil {
		return Catalogue{}, err
	}
	if len(wire.Packs) == 0 {
		return Catalogue{}, fmt.Errorf("%w: catalogue.packs must be nonempty", ErrInvalidSource)
	}
	catalogue := Catalogue{}
	ids, files := make(map[string]bool), make(map[string]bool)
	for _, rawPack := range wire.Packs {
		var pack Pack
		if err := decodeObject(rawPack, &pack, []string{"id", "description", "files", "detect"}); err != nil {
			return Catalogue{}, err
		}
		if !config.ValidGuidancePackID(pack.ID) || ids[pack.ID] {
			return Catalogue{}, fmt.Errorf("%w: invalid or duplicate pack id %q", ErrInvalidSource, pack.ID)
		}
		ids[pack.ID] = true
		if strings.TrimSpace(pack.Description) == "" || len(pack.Files) == 0 {
			return Catalogue{}, fmt.Errorf("%w: pack %q requires a description, nonempty files, and a detect array", ErrInvalidSource, pack.ID)
		}
		for _, name := range pack.Files {
			if !strings.HasPrefix(name, "packs/"+pack.ID+"/") || path.Clean(name) != name || strings.Contains(name, "\\") || strings.ContainsFunc(name, unicode.IsControl) || path.Ext(name) != ".md" || files[name] {
				return Catalogue{}, fmt.Errorf("%w: pack %q has unsafe or duplicate Markdown path %q", ErrInvalidSource, pack.ID, name)
			}
			files[name] = true
		}
		patterns := make(map[string]bool)
		for _, pattern := range pack.Detect {
			if !regexp.MustCompile(`^[A-Za-z0-9_.?*-]+$`).MatchString(pattern) || pattern == "." || pattern == ".." || strings.Contains(pattern, "**") || patterns[pattern] {
				return Catalogue{}, fmt.Errorf("%w: pack %q has invalid or duplicate detection pattern %q", ErrInvalidSource, pack.ID, pattern)
			}
			patterns[pattern] = true
		}
		catalogue.Packs = append(catalogue.Packs, pack)
	}
	return catalogue, nil
}

// Exact keys reject missing fields and JSON's otherwise case-insensitive aliases.
func decodeObject(raw []byte, destination any, fields []string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return fmt.Errorf("%w: decoding catalogue object: %w", ErrInvalidSource, err)
	}
	if len(object) != len(fields) {
		return fmt.Errorf("%w: catalogue object must contain exactly %v", ErrInvalidSource, fields)
	}
	for _, key := range fields {
		if value, ok := object[key]; !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("%w: catalogue object requires non-null field %q", ErrInvalidSource, key)
		}
	}
	if err := json.Unmarshal(raw, destination); err != nil {
		return fmt.Errorf("%w: decoding catalogue fields: %w", ErrInvalidSource, err)
	}
	return nil
}
