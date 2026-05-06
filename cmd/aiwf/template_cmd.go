package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/render"
)

// templateOut is the per-kind payload `aiwf template` emits in both
// text and JSON modes. Kept at package level so writeTemplateText
// can take a typed slice rather than an inline struct.
type templateOut struct {
	Kind entity.Kind `json:"kind"`
	Body string      `json:"body"`
}

// newTemplateCmd builds `aiwf template [kind]`: prints the per-kind
// body template that `aiwf add` would scaffold after the frontmatter.
// Read-only; produces no commit and does not require a consumer repo.
//
// Companion to `aiwf schema`. Schema describes the *frontmatter*
// contract; template describes the body shape (section headers).
// Together they are the full per-kind picture an AI skill scaffolder
// needs to author files outside the `aiwf add` path.
//
// With no kind: emits every kind, separated by `KIND: <kind>` headers.
// With a kind: emits just that template, raw and unprefixed — so
// `aiwf template epic > new_epic_body.md` works as a one-liner.
func newTemplateCmd() *cobra.Command {
	var (
		format string
		pretty bool
	)
	cmd := &cobra.Command{
		Use:           "template [kind]",
		Short:         "Print the body-section template aiwf add would scaffold",
		Args:          cobra.MaximumNArgs(1),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(c *cobra.Command, args []string) error {
			return wrapExitCode(runTemplateCmd(args, format, pretty))
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

func runTemplateCmd(args []string, format string, pretty bool) int {
	if format != "text" && format != "json" {
		fmt.Fprintf(os.Stderr, "aiwf template: --format must be 'text' or 'json', got %q\n", format)
		return exitUsage
	}
	if pretty && format != "json" {
		fmt.Fprintln(os.Stderr, "aiwf template: --pretty has no effect without --format=json")
	}

	var templates []templateOut
	if len(args) == 1 {
		k := entity.Kind(args[0])
		if _, ok := entity.SchemaForKind(k); !ok {
			fmt.Fprintf(os.Stderr, "aiwf template: unknown kind %q (known: %s)\n", args[0], joinKinds(entity.AllKinds()))
			return exitUsage
		}
		templates = []templateOut{{Kind: k, Body: string(entity.BodyTemplate(k))}}
	} else {
		for _, k := range entity.AllKinds() {
			templates = append(templates, templateOut{Kind: k, Body: string(entity.BodyTemplate(k))})
		}
	}

	switch format {
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
		if err := render.JSON(os.Stdout, env, pretty); err != nil {
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
