package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SkillManager coordinates operations on skills.
type SkillManager struct {
}

func NewSkillManager() *SkillManager {
	return &SkillManager{}
}

// Install installs a skill from the given source.
func (m *SkillManager) Install(ctx context.Context, source SkillSource, name string, agent string, scope TargetScope) error {
	target, err := GetAgentTarget(agent)
	if err != nil {
		return err
	}

	installDir, err := target.InstallDir(scope)
	if err != nil {
		return err
	}

	skillDestDir := filepath.Join(installDir, name)

	// Check if already installed
	if _, err := os.Stat(skillDestDir); err == nil {
		return fmt.Errorf("skill '%s' is already installed at %s", name, skillDestDir)
	}

	// Ensure destination parent directory exists before trying to create adjacent temp dir
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create agent skills directory: %w", err)
	}

	// Create temp dir for atomic install
	tempDir, err := os.MkdirTemp(installDir, fmt.Sprintf(".%s-install-*", name))
	if err != nil {
		// Fallback to os.TempDir if we can't create adjacent temp dir
		tempDir, err = os.MkdirTemp("", fmt.Sprintf("gobookmarks-skill-%s-*", name))
		if err != nil {
			return fmt.Errorf("failed to create temp install directory: %w", err)
		}
	}
	defer os.RemoveAll(tempDir) // Will be ignored if we successfully rename

	// Fetch to temp dir
	md, err := source.Fetch(ctx, tempDir)
	if err != nil {
		return fmt.Errorf("failed to fetch skill: %w", err)
	}

	// Update metadata
	md.InstalledAt = time.Now()
	md.AgentTarget = agent
	md.Scope = string(scope)

	if err := WriteMetadata(tempDir, &md); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Ensure destination parent directory exists
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("failed to create agent skills directory: %w", err)
	}

	// Because tempDir may have fallen back to os.TempDir(), it may be on a different
	// device. Try rename first, fallback to copy if it fails.
	if err := os.Rename(tempDir, skillDestDir); err != nil {
		// Fallback to copy if cross-device rename fails
		if copyErr := CopyDir(tempDir, skillDestDir); copyErr != nil {
			return fmt.Errorf("failed to move skill to final destination (copy fallback): %v (original rename err: %w)", copyErr, err)
		}
	}

	return nil
}

// Remove removes an installed skill.
func (m *SkillManager) Remove(name string, agent string, scope TargetScope) error {
	target, err := GetAgentTarget(agent)
	if err != nil {
		return err
	}

	installDir, err := target.InstallDir(scope)
	if err != nil {
		return err
	}

	skillDestDir := filepath.Join(installDir, name)

	if _, err := os.Stat(skillDestDir); os.IsNotExist(err) {
		return fmt.Errorf("skill '%s' is not installed", name)
	}

	// Verify it has our metadata to prevent accidental deletion of non-skill directories
	_, err = ReadMetadata(skillDestDir)
	if err != nil {
		return fmt.Errorf("directory %s does not appear to be a skill installed by us (no metadata found), refusing to delete", skillDestDir)
	}

	return os.RemoveAll(skillDestDir)
}

// List returns a list of installed skills for the given agent and scope.
func (m *SkillManager) List(agent string, scope TargetScope) ([]SkillMetadata, error) {
	target, err := GetAgentTarget(agent)
	if err != nil {
		return nil, err
	}

	installDir, err := target.InstallDir(scope)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(installDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SkillMetadata{}, nil
		}
		return nil, err
	}

	var skills []SkillMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(installDir, entry.Name())
		md, err := ReadMetadata(skillDir)
		if err != nil || md == nil {
			// Skip directories without valid metadata
			continue
		}
		skills = append(skills, *md)
	}

	return skills, nil
}
