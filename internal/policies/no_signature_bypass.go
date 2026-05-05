package policies

// signatureBypassSubstrings are the flags / env-var assignments
// that bypass git's hook + signing infrastructure. Hook bypass
// (`--no-verify`) skips the pre-push aiwf check that makes the
// kernel's guarantees real; signing bypass (`--no-gpg-sign`,
// `commit.gpgsign=false`) lets a code path produce unsigned
// commits even when the contributor's config requires signing.
//
// Production code must never set these. Tests that exercise the
// hook itself are exempt (the policy already excludes _test.go).
var signatureBypassSubstrings = []string{
	`"--no-verify"`,
	`"--no-gpg-sign"`,
	`"commit.gpgsign=false"`,
	`"-c", "commit.gpgsign=false"`,
}

// PolicyNoSignatureBypass flags non-test code that contains a
// known hook / signing bypass spelling.
func PolicyNoSignatureBypass(root string) ([]Violation, error) {
	files, err := WalkGoFiles(root, true)
	if err != nil {
		return nil, err
	}
	var out []Violation
	for _, f := range files {
		for _, sub := range signatureBypassSubstrings {
			offsets := FindAllOffsets(f.Contents, sub)
			for _, off := range offsets {
				out = append(out, Violation{
					Policy: "no-signature-bypass",
					File:   f.Path,
					Line:   LineOf(f.Contents, off),
					Detail: "production code references " + sub +
						"; the kernel's pre-push and signing guarantees must not be bypassed by aiwf itself",
				})
			}
		}
	}
	return out, nil
}
