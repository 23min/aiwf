package policies

// timestampEnvVars are the git env vars that override the author /
// committer date. Setting them lets a process backdate a commit,
// which would corrupt the chronological-order assumption every
// standing rule (and `aiwf history`) relies on.
var timestampEnvVars = []string{
	"GIT_AUTHOR_DATE",
	"GIT_COMMITTER_DATE",
}

// PolicyNoTimestampManipulation forbids the GIT_AUTHOR_DATE and
// GIT_COMMITTER_DATE env vars from appearing in non-test source.
// A code path that sets them is implicitly fabricating an audit
// trail.
func PolicyNoTimestampManipulation(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, f := range files {
		for _, env := range timestampEnvVars {
			offsets := FindAllOffsets(f.Contents, env)
			for _, off := range offsets {
				out = append(out, Violation{
					Policy: "no-timestamp-manipulation",
					File:   f.Path,
					Line:   LineOf(f.Contents, off),
					Detail: "production code references " + env +
						"; backdating a commit corrupts the chronological order the standing rules rely on",
				})
			}
		}
	}
	return out, nil
}
