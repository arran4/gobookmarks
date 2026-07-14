package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallLocalSkill(t *testing.T) {
	// Setup test environment
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // override user config home

	// Create a dummy local skill
	srcDir := t.TempDir()
	skillMdPath := filepath.Join(srcDir, "SKILL.md")
	if err := os.WriteFile(skillMdPath, []byte("# Dummy Skill"), 0644); err != nil {
		t.Fatal(err)
	}

	src := &LocalSource{
		Path: srcDir,
		Name: "test-skill",
	}

	mgr := NewSkillManager()
	err := mgr.Install(context.Background(), src, "test-skill", "common", ScopeUser)
	if err != nil {
		t.Fatalf("Failed to install local skill: %v", err)
	}

	// Verify it was installed
	target, _ := GetAgentTarget("common")
	installDir, _ := target.InstallDir(ScopeUser)

	destMd := filepath.Join(installDir, "test-skill", ".skill-metadata.json")
	if _, err := os.Stat(destMd); os.IsNotExist(err) {
		t.Errorf("Metadata file not created at %s", destMd)
	}

	destSkillMd := filepath.Join(installDir, "test-skill", "SKILL.md")
	if _, err := os.Stat(destSkillMd); os.IsNotExist(err) {
		t.Errorf("SKILL.md not copied to %s", destSkillMd)
	}
}

func TestListAndRemoveSkill(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# Skill 1"), 0644)

	mgr := NewSkillManager()

	// Install
	mgr.Install(context.Background(), &LocalSource{Path: srcDir, Name: "skill1"}, "skill1", "common", ScopeUser)

	// List
	skills, err := mgr.List("common", ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Fatalf("Expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "skill1" {
		t.Errorf("Expected skill name 'skill1', got '%s'", skills[0].Name)
	}

	// Remove
	if err := mgr.Remove("skill1", "common", ScopeUser); err != nil {
		t.Fatalf("Failed to remove skill: %v", err)
	}

	skillsAfter, _ := mgr.List("common", ScopeUser)
	if len(skillsAfter) != 0 {
		t.Fatalf("Expected 0 skills after removal, got %d", len(skillsAfter))
	}
}
