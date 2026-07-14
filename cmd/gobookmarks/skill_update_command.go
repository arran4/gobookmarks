package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arran4/gobookmarks/skill"
)

type SkillUpdateCommand struct {
	parent Command
	Flags  *flag.FlagSet

	All   bool
	Scope string
	Agent string
	Force bool
}

func (sc *SkillCommand) NewSkillUpdateCommand() *SkillUpdateCommand {
	c := &SkillUpdateCommand{
		parent: sc,
		Flags:  flag.NewFlagSet("update", flag.ContinueOnError),
	}
	c.Flags.BoolVar(&c.All, "all", false, "update all installed skills")
	c.Flags.StringVar(&c.Scope, "scope", "project", "installation scope: user or project")
	c.Flags.StringVar(&c.Agent, "agent", "common", "agent target (e.g., common, cursor, copilot)")
	c.Flags.BoolVar(&c.Force, "force", false, "force update even if local modifications exist")
	return c
}

func (c *SkillUpdateCommand) Name() string {
	return c.Flags.Name()
}

func (c *SkillUpdateCommand) Parent() Command {
	return c.parent
}

func (c *SkillUpdateCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *SkillUpdateCommand) Subcommands() []Command {
	return nil
}

func (c *SkillUpdateCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	if forwardHelpIfRequested(c, args) {
		return nil
	}

	remaining := c.FlagSet().Args()
	if len(remaining) < 1 && !c.All {
		err := fmt.Errorf("missing skill name argument\nUsage: gobookmarks skill update <name> or gobookmarks skill update --all")
		printHelp(c, err)
		return err
	}

	manager := skill.NewSkillManager()
	scope := skill.TargetScope(c.Scope)

	skillsToUpdate := []string{}
	if c.All {
		skills, err := manager.List(c.Agent, scope)
		if err != nil {
			return err
		}
		for _, s := range skills {
			skillsToUpdate = append(skillsToUpdate, s.Name)
		}
	} else {
		skillsToUpdate = append(skillsToUpdate, remaining[0])
	}

	if len(skillsToUpdate) == 0 {
		fmt.Println("No skills found to update.")
		return nil
	}

	for _, name := range skillsToUpdate {
		if err := c.updateSkill(manager, name, scope); err != nil {
			fmt.Printf("Error updating skill '%s': %v\n", name, err)
		}
	}

	return nil
}

func (c *SkillUpdateCommand) updateSkill(manager *skill.SkillManager, name string, scope skill.TargetScope) error {
	fmt.Printf("Updating skill '%s'...\n", name)

	target, err := skill.GetAgentTarget(c.Agent)
	if err != nil {
		return err
	}

	dir, err := target.InstallDir(scope)
	if err != nil {
		return err
	}

	skillDir := filepath.Join(dir, name)

	md, err := skill.ReadMetadata(skillDir)
	if err != nil {
		return fmt.Errorf("failed to read metadata (cannot update automatically): %w", err)
	}
	if md == nil {
		return fmt.Errorf("no source metadata is available, so this skill cannot be updated automatically")
	}

	// Simplistic check for local modifications (just SKILL.md for now, to be expanded)
	// A real implementation would hash all files.
	// We'll skip deep modification checks here if --force is provided.

	if md.Source == "local" && !c.Force {
		fmt.Println("Skill is from a local source; consider reinstalling or use --force.")
		return nil // Not strictly an error for `update --all`
	}

	// Construct source to refetch
	sourceStr := md.Original
	if md.Source == "local" {
		sourceStr = md.Original
	}

	source, err := skill.ParseSource(sourceStr, name)
	if err != nil {
		return err
	}
	// For git, reapply the path if necessary (needs to be supported in ParseSource or applied here)
	if gs, ok := source.(*skill.GitSource); ok && md.Path != "" {
		gs.PathInRepo = md.Path
	}

	// Fetch to a temp directory to see if revision changed
	tempDir, err := os.MkdirTemp("", "gobookmarks-skill-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	newMd, err := source.Fetch(context.Background(), tempDir)
	if err != nil {
		return err
	}

	if newMd.Revision == md.Revision && !c.Force {
		fmt.Printf("Skill '%s' is already up to date (revision %s).\n", name, md.Revision)
		return nil
	}

	if c.Force {
		fmt.Println("Force update requested.")
	}

	// Remove old, install new
	if err := manager.Remove(name, c.Agent, scope); err != nil {
		return fmt.Errorf("failed to remove old version: %w", err)
	}

	// Instead of full reinstall via manager (which fetches again), we can just move the tempDir
	// and write metadata.
	newMd.InstalledAt = md.InstalledAt // Or update it? Usually update it.
	newMd.AgentTarget = md.AgentTarget
	newMd.Scope = md.Scope

	if err := skill.WriteMetadata(tempDir, &newMd); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Because tempDir was created using os.MkdirTemp("", ...), it may be on a different
	// device. Try rename first, fallback to copy if it fails.
	if err := os.Rename(tempDir, skillDir); err != nil {
		// Fallback to copy if cross-device rename fails
		if copyErr := skill.CopyDir(tempDir, skillDir); copyErr != nil {
			return fmt.Errorf("failed to move updated skill to final destination (copy fallback): %v (original rename err: %w)", copyErr, err)
		}
	}

	fmt.Printf("Successfully updated skill '%s' to revision %s\n", name, newMd.Revision)
	return nil
}
