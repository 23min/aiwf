package skills

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ArtifactFamily distinguishes verb skills from advisory ritual families.
type ArtifactFamily string

// Artifact families inspected by doctor.
const (
	FamilySkills    ArtifactFamily = "skills"
	FamilyRituals   ArtifactFamily = "rituals"
	FamilyAgents    ArtifactFamily = "agents"
	FamilyTemplates ArtifactFamily = "templates"
	FamilyGuidance  ArtifactFamily = "guidance"
)

// ArtifactState describes a disk comparison, not delivery into a model context.
type ArtifactState string

// States produced by an artifact inspection.
const (
	ArtifactCurrent ArtifactState = "current"
	ArtifactMissing ArtifactState = "missing"
	ArtifactDrifted ArtifactState = "drifted"
	ArtifactBlocked ArtifactState = "blocked"
)

// ArtifactStatus identifies an expected generated file and its disk state.
type ArtifactStatus struct {
	Family ArtifactFamily
	Path   string
	State  ArtifactState
	Detail string
}

// InspectArtifacts compares selected-host files with the same rendered families
// materialization writes, including configured agent tiers. It never writes and
// rejects symlink components using the installer's path safety check.
func InspectArtifacts(ctx context.Context, root string, target Target, tiers map[string]AgentTier) ([]ArtifactStatus, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("inspecting artifacts: %w", err)
	}
	sources, err := loadArtifactSources()
	if err != nil {
		return nil, err
	}
	families, err := renderArtifactFamilies(target, tiers, sources)
	if err != nil {
		return nil, err
	}
	var statuses []ArtifactStatus
	for _, family := range families {
		kind := FamilyTemplates
		if family.dir == target.AgentsDir {
			kind = FamilyAgents
		}
		for _, file := range family.files {
			relative := filepath.Join(family.dir, file.Name)
			if family.skills {
				kind = FamilyRituals
				if strings.HasPrefix(file.Name, "aiwf-") {
					kind = FamilySkills
				}
				relative = filepath.Join(relative, "SKILL.md")
			}
			statuses = append(statuses, InspectArtifact(ctx, root, kind, relative, file.Content))
		}
	}
	return statuses, nil
}

// InspectArtifact compares one expected file using the materializer's path guards.
func InspectArtifact(ctx context.Context, root string, family ArtifactFamily, relative string, expected []byte) ArtifactStatus {
	status := ArtifactStatus{Family: family, Path: relative, State: ArtifactCurrent}
	info, err := checkedArtifactPath(ctx, root, relative)
	switch {
	case err != nil:
		status.State, status.Detail = ArtifactBlocked, err.Error()
	case info == nil:
		status.State = ArtifactMissing
	case !info.Mode().IsRegular():
		status.State, status.Detail = ArtifactBlocked, "expected a regular file; repair the path before running aiwf update"
	default:
		content, readErr := os.ReadFile(filepath.Join(root, relative))
		if readErr != nil {
			status.State, status.Detail = ArtifactBlocked, readErr.Error()
		} else if !bytes.Equal(content, expected) {
			status.State = ArtifactDrifted
		}
	}
	return status
}
