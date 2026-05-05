package htmlrender

// idToFileName maps an entity id to its rendered HTML filename. No
// subdirectory scheme — every entity ships at the top of OutDir.
// Composite ids never get their own page; they live as anchors inside
// the parent milestone's file (`M-NNN.html#ac-N`), reachable via the
// ACAnchor helper.
//
// Top-level kinds: `<id>.html`. Examples:
//
//	E-01      → E-01.html
//	M-007     → M-007.html
//	ADR-0042  → ADR-0042.html
//	G-099     → G-099.html
//	D-003     → D-003.html
//	C-100     → C-100.html
//
// Empty input returns "index.html" — the top-level page.
func idToFileName(id string) string {
	if id == "" {
		return "index.html"
	}
	return id + ".html"
}

// ACAnchor returns the in-page anchor for one AC inside the milestone
// page: e.g. ACAnchor("AC-3") == "ac-3". The CSS `:target` selector in
// step-5 templates uses this convention.
func ACAnchor(acID string) string {
	// AC ids are short and known-ASCII; lowercasing is sufficient.
	out := make([]byte, len(acID))
	for i := 0; i < len(acID); i++ {
		c := acID[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out[i] = c
	}
	return string(out)
}
