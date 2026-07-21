package verb

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/entity"
)

// nameSubstitution selects which of substituteNamePart's two fixed
// call shapes to run — rename.go's (keep the id, change the slug) or
// reallocate.go's (keep the slug, change the id). The two dimensions
// this used to vary independently (which part replacement
// substitutes; what to do when name has no existing slug) are only
// ever called together in these two combinations, so one enum
// expresses the real constraint instead of two orthogonal parameters
// that could be paired invalidly.
type nameSubstitution int

const (
	// substituteSlugMode keeps the id prefix and replaces the slug —
	// rename.go's job. A slug-less id ("E-01") gains one by appending
	// replacement.
	substituteSlugMode nameSubstitution = iota
	// substituteIDMode keeps the slug and replaces the id prefix —
	// reallocate.go's job. A slug-less name has nothing to preserve,
	// so replacement is returned bare.
	substituteIDMode
)

// substituteNamePart splits name (shaped
// "<kind-letter>-<digits>-<slug>") at the second hyphen into an
// id-prefix and a slug, replaces one of the two parts with
// replacement per mode, and rejoins them. The no-second-hyphen
// fallback is the one place the two callers' behavior genuinely
// diverges, not just cosmetically (docs/initiatives/
// verb-layer-cleanup.md F2's "verified nuance"): rename appends the
// new slug, reallocate discards and replaces.
func substituteNamePart(name, replacement string, mode nameSubstitution) (string, error) {
	first := strings.IndexByte(name, '-')
	if first < 0 {
		return "", fmt.Errorf("name %q has no id prefix", name)
	}
	second := strings.IndexByte(name[first+1:], '-')
	if second < 0 {
		if mode == substituteSlugMode {
			return name + "-" + replacement, nil
		}
		return replacement, nil
	}
	idPart := name[:first+1+second]
	slug := name[first+1+second+1:]
	if mode == substituteSlugMode {
		return idPart + "-" + replacement, nil
	}
	return replacement + "-" + slug, nil
}

// rewriteEntityName computes the (source, dest) move for e's file or
// directory rename: for directory-based kinds (epic, contract), the
// source is the entity's containing directory and dest is the dir's
// new name; for file-based kinds, the source is the entity file
// itself. substitute is applied to the old basename (the ".md" suffix
// stripped for file-based kinds) to produce the new one.
//
// Shared by renamePaths and reallocatePaths — the kind-switch and
// path-join shape is genuinely identical between the two; substitute
// is where their real behavior diverges (rename replaces the slug and
// keeps the id, reallocate replaces the id and keeps the slug — see
// substituteNamePart).
func rewriteEntityName(e *entity.Entity, substitute func(name string) (string, error)) (source, dest string, err error) {
	switch e.Kind {
	case entity.KindEpic, entity.KindContract:
		// Containing directory moves; the file inside keeps its name.
		dir := filepath.Dir(e.Path)
		parent, oldName := filepath.Split(dir)
		newName, err := substitute(oldName)
		if err != nil {
			return "", "", err
		}
		// strip trailing separator from parent
		parent = strings.TrimRight(parent, "/")
		return dir, filepath.Join(parent, newName), nil
	default:
		// File renames: the .md basename gets a new name.
		dir, oldName := filepath.Split(e.Path)
		newName, err := substitute(strings.TrimSuffix(oldName, ".md"))
		if err != nil {
			return "", "", err
		}
		dir = strings.TrimRight(dir, "/")
		return e.Path, filepath.Join(dir, newName+".md"), nil
	}
}
