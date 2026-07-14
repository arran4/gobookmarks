package main

import (
	"flag"
	"fmt"
)

type SkillCommand struct {
	parent Command
	Flags  *flag.FlagSet

	InstallCmd *SkillInstallCommand
	UpdateCmd  *SkillUpdateCommand
	RemoveCmd  *SkillRemoveCommand
	ListCmd    *SkillListCommand
	InspectCmd *SkillInspectCommand
	DoctorCmd  *SkillDoctorCommand
}

func (rc *RootCommand) NewSkillCommand() (*SkillCommand, error) {
	c := &SkillCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("skill", flag.ContinueOnError),
	}

	c.InstallCmd = c.NewSkillInstallCommand()
	c.UpdateCmd = c.NewSkillUpdateCommand()
	c.RemoveCmd = c.NewSkillRemoveCommand()
	c.ListCmd = c.NewSkillListCommand()
	c.InspectCmd = c.NewSkillInspectCommand()
	c.DoctorCmd = c.NewSkillDoctorCommand()

	return c, nil
}

func (c *SkillCommand) Name() string {
	return c.Flags.Name()
}

func (c *SkillCommand) Parent() Command {
	return c.parent
}

func (c *SkillCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *SkillCommand) Subcommands() []Command {
	return []Command{c.InstallCmd, c.UpdateCmd, c.RemoveCmd, c.ListCmd, c.InspectCmd, c.DoctorCmd}
}

func (c *SkillCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}

	remaining := c.FlagSet().Args()
	if len(remaining) == 0 {
		printHelp(c, nil)
		return nil
	}

	switch remaining[0] {
	case c.InstallCmd.Name():
		return c.InstallCmd.Execute(remaining[1:])
	case c.UpdateCmd.Name():
		return c.UpdateCmd.Execute(remaining[1:])
	case c.RemoveCmd.Name():
		return c.RemoveCmd.Execute(remaining[1:])
	case c.ListCmd.Name():
		return c.ListCmd.Execute(remaining[1:])
	case c.InspectCmd.Name():
		return c.InspectCmd.Execute(remaining[1:])
	case c.DoctorCmd.Name():
		return c.DoctorCmd.Execute(remaining[1:])
	default:
		err := fmt.Errorf("unknown skill command: %s", remaining[0])
		printHelp(c, err)
		return err
	}
}
