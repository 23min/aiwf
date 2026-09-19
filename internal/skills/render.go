package skills

import (
	"errors"
	"fmt"
	"strings"
)

// Render error identities distinguish malformed sources from incomplete host
// bindings. Each returned error also names its source, host, and offending token.
var (
	ErrInvalidRenderSyntax   = errors.New("invalid aiwf template syntax")
	ErrUnknownRenderBinding  = errors.New("unknown aiwf binding")
	ErrUnknownRenderFragment = errors.New("unknown aiwf fragment")
	ErrMissingRenderBinding  = errors.New("missing aiwf host binding")
)

// HostFragments contains host-specific instructions in shared workflow sources.
// Distinct worktree contexts retain their own slots; hosts may share their values.
// Fragments may reference path bindings, not other fragments.
type HostFragments struct {
	SkillInvocation            string
	WorktreeEntry              string
	ReviewDispatch             string
	EpicWorktreeEntry          string
	MilestoneWorktreeEntry     string
	EpicWorktreePlacement      string
	MilestoneWorktreePlacement string
	EpicExternalWorktree       string
	MilestoneExternalWorktree  string
}

// RenderBindings supplies the existing artifact layout and the named host
// instructions. Optional target paths are required only when referenced.
type RenderBindings struct {
	Target    Target
	Fragments HostFragments
}

const renderPrefix = "{{aiwf:"

// RenderSkills expands only reserved {{aiwf:...}} tokens, preserving all other
// bytes. It does no I/O and returns no partial batch on failure, so callers can
// render their complete artifact set before starting any filesystem mutation.
// Results own their byte slices; neither sources nor bindings are modified.
//
// Bindings are host, host_label, skills_dir, agents_dir, templates_dir, hooks_dir.
// Fragment tokens select skill_invocation, review_dispatch, worktree_entry,
// or the epic/milestone variants of worktree_entry, worktree_placement, and
// external_worktree. Names are exact, without surrounding whitespace.
func RenderSkills(sources []Skill, bindings RenderBindings) ([]Skill, error) {
	if len(sources) == 0 {
		return nil, nil
	}
	out := make([]Skill, len(sources))
	for i, source := range sources {
		if bindings.Target.Name == "" {
			return nil, fmt.Errorf("rendering %q: %w: host", source.Name, ErrMissingRenderBinding)
		}
		content, err := renderText(string(source.Content), bindings, true)
		if err != nil {
			return nil, fmt.Errorf("rendering %q for host %q: %w", source.Name, bindings.Target.Name, err)
		}
		if start := strings.Index(content, renderPrefix); start != -1 {
			return nil, fmt.Errorf("rendering %q for host %q: %w: substitution produced a reserved token %q", source.Name, bindings.Target.Name, ErrInvalidRenderSyntax, content[start:])
		}
		out[i] = Skill{Name: source.Name, Content: []byte(content)}
	}
	return out, nil
}

func renderText(source string, bindings RenderBindings, allowFragments bool) (string, error) {
	var out strings.Builder
	for {
		start := strings.Index(source, renderPrefix)
		if start == -1 {
			out.WriteString(source)
			return out.String(), nil
		}
		out.WriteString(source[:start])
		rest := source[start+len(renderPrefix):]
		end := strings.Index(rest, "}}")
		if end == -1 {
			return "", fmt.Errorf("%w: unclosed token %q", ErrInvalidRenderSyntax, source[start:])
		}
		token := rest[:end]
		if token == "" || token == "fragment:" || strings.ContainsAny(token, " \t\r\n{}") {
			return "", fmt.Errorf("%w: %q", ErrInvalidRenderSyntax, source[start:start+len(renderPrefix)+end+2])
		}
		var value string
		if strings.HasPrefix(token, "fragment:") {
			if !allowFragments {
				return "", fmt.Errorf("%w: nested fragment %q", ErrInvalidRenderSyntax, token)
			}
			switch token {
			case "fragment:skill_invocation":
				value = bindings.Fragments.SkillInvocation
			case "fragment:worktree_entry":
				value = bindings.Fragments.WorktreeEntry
			case "fragment:review_dispatch":
				value = bindings.Fragments.ReviewDispatch
			case "fragment:epic_worktree_entry":
				value = bindings.Fragments.EpicWorktreeEntry
			case "fragment:milestone_worktree_entry":
				value = bindings.Fragments.MilestoneWorktreeEntry
			case "fragment:epic_worktree_placement":
				value = bindings.Fragments.EpicWorktreePlacement
			case "fragment:milestone_worktree_placement":
				value = bindings.Fragments.MilestoneWorktreePlacement
			case "fragment:epic_external_worktree":
				value = bindings.Fragments.EpicExternalWorktree
			case "fragment:milestone_external_worktree":
				value = bindings.Fragments.MilestoneExternalWorktree
			default:
				return "", fmt.Errorf("%w: %q", ErrUnknownRenderFragment, token)
			}
			if value == "" {
				return "", fmt.Errorf("%w: %q", ErrMissingRenderBinding, token)
			}
			rendered, err := renderText(value, bindings, false)
			if err != nil {
				return "", fmt.Errorf("expanding %q: %w", token, err)
			}
			value = rendered
		} else {
			switch token {
			case "host":
				value = bindings.Target.Name
			case "host_label":
				value = bindings.Target.Name
				switch value {
				case "claude":
					value = "Claude Code"
				case "codex":
					value = "Codex"
				}
			case "skills_dir":
				value = bindings.Target.SkillsDir
			case "agents_dir":
				value = bindings.Target.AgentsDir
			case "templates_dir":
				value = bindings.Target.TemplatesDir
			case "hooks_dir":
				value = bindings.Target.HooksDir
			default:
				return "", fmt.Errorf("%w: %q", ErrUnknownRenderBinding, token)
			}
			if value == "" {
				return "", fmt.Errorf("%w: %q", ErrMissingRenderBinding, token)
			}
			if strings.Contains(value, renderPrefix) {
				return "", fmt.Errorf("%w: binding %q contains a reserved token", ErrInvalidRenderSyntax, token)
			}
		}
		out.WriteString(value)
		source = rest[end+2:]
	}
}
