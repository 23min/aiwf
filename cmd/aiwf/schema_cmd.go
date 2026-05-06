package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/render"
)

// newSchemaCmd builds `aiwf schema [kind]`: prints the frontmatter
// contract for one kind or for all six. Read-only; produces no commit
// and does not require a consumer repo. The intended audience is skill
// authors writing recipes that hand-edit aiwf-managed files — they can
// read the schema once and stop guessing field names.
func newSchemaCmd() *cobra.Command {
	var (
		format string
		pretty bool
	)
	cmd := &cobra.Command{
		Use:   "schema [kind]",
		Short: "Print the frontmatter contract for one or all kinds",
		Example: `  # Print every kind's contract
  aiwf schema

  # Print just the milestone contract
  aiwf schema milestone`,
		Args:          cobra.MaximumNArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runSchemaCmd(args, format, pretty))
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().BoolVar(&pretty, "pretty", false, "indent JSON output (only with --format=json)")
	registerFormatCompletion(cmd)
	cmd.ValidArgsFunction = func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return allKindNames(), cobra.ShellCompDirectiveNoFileComp
	}
	return cmd
}

func runSchemaCmd(args []string, format string, pretty bool) int {
	if format != "text" && format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf schema: --format must be 'text' or 'json', got %q\n", format)
		return exitUsage
	}
	if pretty && format != "json" {
		fmt.Fprintln(os.Stderr, "aiwf schema: --pretty has no effect without --format=json")
	}

	var schemas []entity.Schema
	if len(args) == 1 {
		k := entity.Kind(args[0])
		s, ok := entity.SchemaForKind(k)
		if !ok {
			fmt.Fprintf(os.Stderr, "aiwf schema: unknown kind %q (known: %s)\n", args[0], joinKinds(entity.AllKinds()))
			return exitUsage
		}
		schemas = []entity.Schema{s}
	} else {
		schemas = entity.AllSchemas()
	}

	switch format {
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
		if err := render.JSON(os.Stdout, env, pretty); err != nil {
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
