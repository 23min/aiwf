package htmlrender

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
)

// templatesFS holds the per-page Go html/template sources. Step 3
// ships placeholder templates so the seam compiles and the
// determinism / link-integrity tests have something to operate on;
// step 5 replaces them with the full per-page layout.
//
//go:embed embedded/*.tmpl
var templatesFS embed.FS

// embeddedStyleCSS is the single hand-written stylesheet shipped
// alongside every render. Step 3 ships an empty-but-present
// placeholder so the file always lands at <out>/assets/style.css;
// step 5 replaces with the real content (target: under 5 KB).
//
//go:embed embedded/style.css
var embeddedStyleCSS []byte

// loadTemplates parses every .tmpl file in templatesFS into a single
// *template.Template tree. Each template is named after its filename
// (e.g. "index.tmpl") and looked up by name at render time.
//
// The custom functions:
//   - acAnchor: lowercase AC anchor used by milestone-page links;
//     templates use it to construct `#ac-N` fragments without
//     re-deriving the rule.
//   - cssHref: returns the stylesheet href with a content-hash
//     cache-buster query string. file:// browsers cache stylesheets
//     across reloads (even Cmd+Shift+R sometimes), so changing the
//     CSS without the buster leaves users staring at the old palette.
//     The query-string fingerprint forces a fresh fetch on any
//     CSS change.
func loadTemplates() (*template.Template, error) {
	root := template.New("").Funcs(template.FuncMap{
		"acAnchor": ACAnchor,
		"cssHref":  cssHref,
	})
	tmpls, err := root.ParseFS(templatesFS, "embedded/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing embedded templates: %w", err)
	}
	return tmpls, nil
}

// cssHashShort is the first 8 hex chars of sha256(embeddedStyleCSS).
// Computed once at process start; the var is stable across renders
// in the same binary.
var cssHashShort = func() string {
	sum := sha256.Sum256(embeddedStyleCSS)
	return hex.EncodeToString(sum[:])[:8]
}()

// cssHref returns the stylesheet href every template should use.
// Format: "assets/style.css?v=<8-hex>". Deterministic per CSS
// content — re-renders against the same binary produce the same
// href, so the determinism test still passes byte-for-byte.
func cssHref() string {
	return "assets/style.css?v=" + cssHashShort
}
