package htmlrender

import (
	"embed"
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
// The custom function "acAnchor" exposes the lowercase AC anchor
// used by milestone-page links; templates use it to construct
// `#ac-N` fragments without re-deriving the rule.
func loadTemplates() (*template.Template, error) {
	root := template.New("").Funcs(template.FuncMap{
		"acAnchor": ACAnchor,
	})
	tmpls, err := root.ParseFS(templatesFS, "embedded/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing embedded templates: %w", err)
	}
	return tmpls, nil
}
