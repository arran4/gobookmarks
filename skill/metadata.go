package skill

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// SkillMetadata holds provenance information for an installed skill.
type SkillMetadata struct {
	Name        string    `json:"name"`
	Source      string    `json:"source"` // e.g., "git", "local"
	Original    string    `json:"original"` // repository URL or original path
	Path        string    `json:"path,omitempty"` // path within the repository
	Revision    string    `json:"revision"` // commit hash or content digest
	InstalledAt time.Time `json:"installed_at"`
	AgentTarget string    `json:"agent_target"`
	Scope       string    `json:"scope"`
}

// ReadMetadata reads the skill metadata from the given skill directory.
func ReadMetadata(skillDir string) (*SkillMetadata, error) {
	mdPath := filepath.Join(skillDir, ".skill-metadata.json")
	b, err := os.ReadFile(mdPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No metadata found
		}
		return nil, err
	}

	var md SkillMetadata
	if err := json.Unmarshal(b, &md); err != nil {
		return nil, err
	}

	return &md, nil
}

// WriteMetadata writes the skill metadata to the given skill directory.
func WriteMetadata(skillDir string, md *SkillMetadata) error {
	mdPath := filepath.Join(skillDir, ".skill-metadata.json")
	b, err := json.MarshalIndent(md, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(mdPath, b, 0644)
}
