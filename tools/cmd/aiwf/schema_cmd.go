package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/render"
)

// runSchema handles `aiwf schema [kind]`: prints the frontmatter
// contract for one kind or for all six. Read-only; produces no commit
// and does not require a consumer repo. The intended audience is skill
// authors writing recipes that hand-edit aiwf-managed files — they can
// read the schema once and stop guessing field names.
func runSchema(args []string) int {
	fs := flag.NewFlagSet("schema", flag.ContinueOnError)
	format := fs.String("format", "text", "output format: text or json")
	pretty := fs.Bool("pretty", false, "indent JSON output (only with --format=json)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf schema: --format must be 'text' or 'json', got %q\n", *format)
		return exitUsage
	}
	if *pretty && *format != "json" {
		fmt.Fprintln(os.Stderr, "aiwf schema: --pretty has no effect without --format=json")
	}

	rest := fs.Args()
	if len(rest) > 1 {
		fmt.Fprintf(os.Stderr, "aiwf schema: expected zero or one kind argument, got %d\n", len(rest))
		return exitUsage
	}

	var schemas []entity.Schema
	if len(rest) == 1 {
		k := entity.Kind(rest[0])
		s, ok := entity.SchemaForKind(k)
		if !ok {
			fmt.Fprintf(os.Stderr, "aiwf schema: unknown kind %q (known: %s)\n", rest[0], joinKinds(entity.AllKinds()))
			return exitUsage
		}
		schemas = []entity.Schema{s}
	} else {
		schemas = entity.AllSchemas()
	}

	switch *format {
	case "text":
		if err := writeSchemaText(os.Stdout, schemas); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf schema: writing output: %v\n", err)
			return exitInternal
		}
	case "json":
		env := render.Envelope{
			Tool:    "aiwf",
			Version: Version,
			Status:  "ok",
			Result:  map[string]any{"schemas": schemas},
		}
		if err := render.JSON(os.Stdout, env, *pretty); err != nil {
			fmt.Fprintf(os.Stderr, "aiwf schema: writing output: %v\n", err)
			return exitInternal
		}
	}
	return exitOK
}

// writeSchemaText renders schemas one block per kind, in a fixed-column
// layout that aligns the field-description rows under each header.
func writeSchemaText(w io.Writer, schemas []entity.Schema) error {
	for i := range schemas {
		s := &schemas[i]
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "KIND: %s\n", s.Kind); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  id format:        %s\n", s.IDFormat); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  allowed statuses: %s\n", strings.Join(s.AllowedStatuses, ", ")); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  required fields:  %s\n", strings.Join(s.RequiredFields, ", ")); err != nil {
			return err
		}
		if len(s.OptionalFields) > 0 {
			if _, err := fmt.Fprintf(w, "  optional fields:  %s\n", strings.Join(s.OptionalFields, ", ")); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(w, "  optional fields:  (none)"); err != nil {
				return err
			}
		}
		if len(s.References) > 0 {
			if _, err := fmt.Fprintln(w, "  reference fields:"); err != nil {
				return err
			}
			for j := range s.References {
				r := &s.References[j]
				kinds := "(any)"
				if len(r.AllowedKinds) > 0 {
					kinds = joinKinds(r.AllowedKinds)
				}
				req := "optional"
				if !r.Optional {
					req = "required"
				}
				if _, err := fmt.Fprintf(w, "    %-15s %-7s -> %-22s (%s)\n", r.Name, r.Cardinality, kinds, req); err != nil {
					return err
				}
			}
		} else {
			if _, err := fmt.Fprintln(w, "  reference fields: (none)"); err != nil {
				return err
			}
		}
	}
	return nil
}

func joinKinds(ks []entity.Kind) string {
	parts := make([]string, len(ks))
	for i, k := range ks {
		parts[i] = string(k)
	}
	return strings.Join(parts, ", ")
}
