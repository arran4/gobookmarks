package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arran4/gobookmarks/skill"
)

type SkillDoctorCommand struct {
	parent Command
	Flags  *flag.FlagSet

	Scope string
	Agent string
}

func (sc *SkillCommand) NewSkillDoctorCommand() *SkillDoctorCommand {
	c := &SkillDoctorCommand{
		parent: sc,
		Flags:  flag.NewFlagSet("doctor", flag.ContinueOnError),
	}
	c.Flags.StringVar(&c.Scope, "scope", "project", "installation scope: user or project")
	c.Flags.StringVar(&c.Agent, "agent", "common", "agent target (e.g., common, cursor, copilot)")
	return c
}

func (c *SkillDoctorCommand) Name() string {
	return c.Flags.Name()
}

func (c *SkillDoctorCommand) Parent() Command {
	return c.parent
}

func (c *SkillDoctorCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *SkillDoctorCommand) Subcommands() []Command {
	return nil
}

func (c *SkillDoctorCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	if forwardHelpIfRequested(c, args) {
		return nil
	}

	scope := skill.TargetScope(c.Scope)
	target, err := skill.GetAgentTarget(c.Agent)
	if err != nil {
		return err
	}

	installDir, err := target.InstallDir(scope)
	if err != nil {
		return err
	}

	fmt.Printf("Checking skills in %s...\n", installDir)

	entries, err := os.ReadDir(installDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Install directory does not exist. No skills installed.")
			return nil
		}
		return err
	}

	issuesFound := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillName := entry.Name()
		skillDir := filepath.Join(installDir, skillName)

		fmt.Printf("- Checking '%s': ", skillName)

		md, err := skill.ReadMetadata(skillDir)
		if err != nil {
			fmt.Printf("FAIL (error reading metadata: %v)\n", err)
			issuesFound++
			continue
		}
		if md == nil {
			fmt.Printf("WARN (no metadata found, might not be a skill managed by us)\n")
			issuesFound++
			continue
		}

		skillMdPath := filepath.Join(skillDir, "SKILL.md")
		if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
			fmt.Printf("FAIL (SKILL.md not found in directory)\n")
			issuesFound++
			continue
		}

		fmt.Printf("OK\n")
	}

	if issuesFound == 0 {
		fmt.Println("\nAll installed skills appear healthy.")
	} else {
		fmt.Printf("\nFound %d issues.\n", issuesFound)
	}

	return nil
}
