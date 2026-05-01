package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/render"
)

// templateOut is the per-kind payload `aiwf template` emits in both
// text and JSON modes. Kept at package level so writeTemplateText
// can take a typed slice rather than an inline struct.
type templateOut struct {
	Kind entity.Kind `json:"kind"`
	Body string      `json:"body"`
}

// runTemplate handles `aiwf template [kind]`: prints the per-kind body
// template that `aiwf add` would scaffold after the frontmatter. Read-
// only; produces no commit and does not require a consumer repo.
//
// Companion to `aiwf schema`. Schema describes the *frontmatter*
// contract; template describes the body shape (section headers).
// Together they are the full per-kind picture an AI skill scaffolder
// needs to author files outside the `aiwf add` path.
//
// With no kind: emits every kind, separated by `KIND: <kind>` headers.
// With a kind: emits just that template, raw and unprefixed — so
// `aiwf template epic > new_epic_body.md` works as a one-liner.
func runTemplate(args []string) int {
	fs := flag.NewFlagSet("template", flag.ContinueOnError)
	format := fs.String("format", "text", "output format: text or json")
	pretty := fs.Bool("pretty", false, "indent JSON output (only with --format=json)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf template: --format must be 'text' or 'json', got %q\n", *format)
		return exitUsage
	}
	if *pretty && *format != "json" {
		fmt.Fprintln(os.Stderr, "aiwf template: --pretty has no effect without --format=json")
	}

	rest := fs.Args()
	if len(rest) > 1 {
		fmt.Fprintf(os.Stderr, "aiwf template: expected zero or one kind argument, got %d\n", len(rest))
		return exitUsage
	}

	var templates []templateOut
	if len(rest) == 1 {
		k := entity.Kind(rest[0])
		if _, ok := entity.SchemaForKind(k); !ok {
			fmt.Fprintf(os.Stderr, "aiwf template: unknown kind %q (known: %s)\n", rest[0], joinKinds(entity.AllKinds()))
			return exitUsage
		}
		templates = []templateOut{{Kind: k, Body: string(entity.BodyTemplate(k))}}
	} else {
		for _, k := range entity.AllKinds() {
			templates = append(templates, templateOut{Kind: k, Body: string(entity.BodyTemplate(k))})
		}
	}

	switch *format {
	case "text":
		single := len(templates) == 1
		if err := writeTemplateText(os.Stdout, templates, single); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf template: writing output: %v\n", err)
			return exitInternal
		}
	case "json":
		env := render.Envelope{
			Tool:    "aiwf",
			Version: Version,
			Status:  "ok",
			Result:  map[string]any{"templates": templates},
		}
		if err := render.JSON(os.Stdout, env, *pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf template: writing output: %v\n", err)
			return exitInternal
		}
	}
	return exitOK
}

// writeTemplateText emits the body templates. When single is true,
// the body is written raw (no header) so the output can be piped
// directly into a new file. When single is false, each body is
// prefixed by a `KIND: <kind>` header so the kinds can be told
// apart in the stream.
func writeTemplateText(w io.Writer, ts []templateOut, single bool) error {
	for i, t := range ts {
		if !single {
			if i > 0 {
				if _, err := fmt.Fprintln(w); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(w, "KIND: %s\n", t.Kind); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, t.Body); err != nil {
			return err
		}
	}
	return nil
}
