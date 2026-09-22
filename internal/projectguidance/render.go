package projectguidance

import (
	"fmt"
	"net/url"
	"strings"
)

const routeBody = `Before using engineering guidance, check for .guidance/.aiwf-pending. If it exists,
report that installation is incomplete and ask the maintainer to rerun aiwf update;
do not treat the installed pack files as a complete policy set.
Otherwise read [.guidance/project.md](.guidance/project.md) first when it exists,
then [.guidance/index.md](.guidance/index.md) and the packs relevant to the task.
Handwritten project overrides take precedence over pack guidance. Load supporting
rubrics only when relevant to the task. Do not also load legacy engineering guidance.`

func markdownText(text string) string {
	return strings.NewReplacer("\\", "\\\\", "`", "\\`", "[", "\\[", "]", "\\]", "<", "&lt;", ">", "&gt;", "*", "\\*", "_", "\\_", "\n", " ", "\r", " ").Replace(text)
}

func renderInstallation(snapshot *Snapshot, opts InstallOptions) (map[string][]byte, error) {
	if snapshot == nil || snapshot.Commit == "" {
		return nil, fmt.Errorf("%w: source snapshot has no commit", ErrInvalidSource)
	}
	files := make(map[string][]byte)
	var index strings.Builder
	index.WriteString("<!-- aiwf:engineering-guidance -->\n# Project engineering guidance\n\n")
	fmt.Fprintf(&index, "Source: %s\n\nInstalled commit: `%s`\n\n", markdownText(opts.Source), snapshot.Commit)
	index.WriteString("Read [project overrides](project.md) first when present; they take precedence over these packs.\n\n")
	packs := make(map[string]Pack)
	for _, pack := range snapshot.Catalogue.Packs {
		packs[pack.ID] = pack
	}
	seen := make(map[string]bool)
	for _, id := range opts.Selected {
		pack, ok := packs[id]
		if !ok || seen[id] || len(pack.Files) == 0 {
			return nil, fmt.Errorf("%w: selected pack %q is missing, repeated, or has no entry point", ErrInvalidSource, id)
		}
		seen[id] = true
		link := (&url.URL{Path: pack.Files[0]}).String()
		fmt.Fprintf(&index, "- [%s](%s) — %s\n", id, link, markdownText(pack.Description))
		for _, name := range pack.Files {
			content, ok := snapshot.Documents[name]
			if !ok {
				return nil, fmt.Errorf("%w: selected document %s was not retrieved", ErrInvalidSource, name)
			}
			files[".guidance/"+name] = content
		}
	}
	files[indexFile] = []byte(index.String())
	for _, name := range opts.HostFiles {
		if !hostFile(name) {
			return nil, fmt.Errorf("%w: unknown host file %q", ErrInstallConflict, name)
		}
		files[name] = []byte(routeBody)
	}
	return files, nil
}
