package projectguidance

import (
	"context"
	"slices"
	"strings"
)

// Issue identifies a local artifact that needs maintainer attention.
type Issue struct {
	Path   string
	Detail string
}

// Inspection describes local installation evidence, without checking upstream.
type Inspection struct {
	Revision string
	Files    []string
	Issues   []Issue
	Pending  bool
}

// Inspect checks installed bytes against their recorded ownership. Revision is
// reported only from an intact owned index; pending installation may mix revisions.
func Inspect(ctx context.Context, root string) (Inspection, error) {
	report := Inspection{}
	owned, pending, err := readInstallRecords(ctx, root)
	if err != nil {
		return report, err
	}
	report.Pending = pending != nil
	for name := range owned {
		report.Files = append(report.Files, name)
	}
	slices.Sort(report.Files)
	for _, name := range report.Files {
		data, _, err := readInstallFile(ctx, root, name)
		if err != nil {
			report.Issues = append(report.Issues, Issue{name, err.Error()})
			continue
		}
		content, err := ownedBytes(name, data)
		switch {
		case err != nil:
			report.Issues = append(report.Issues, Issue{name, err.Error()})
		case content == nil:
			report.Issues = append(report.Issues, Issue{name, "missing; run aiwf update to restore selected artifacts"})
		case digest(content) != owned[name] && !pending[name].recognizes(digest(content)):
			report.Issues = append(report.Issues, Issue{name, "locally modified; preserve your changes in .guidance/project.md or reconcile before selecting this artifact for update"})
		case name == indexFile:
			for _, line := range strings.Split(string(content), "\n") {
				if value, ok := strings.CutPrefix(line, "Installed commit: `"); ok {
					report.Revision = strings.TrimSuffix(value, "`")
					break
				}
			}
		}
	}
	return report, nil
}
