package main

import (
	"flag"
	"fmt"

	. "github.com/arran4/gobookmarks"
)

type DbUsersCommand struct {
	parent Command
	Flags  *flag.FlagSet
}

func (dc *DbCommand) NewDbUsersCommand() (*DbUsersCommand, error) {
	c := &DbUsersCommand{
		parent: dc,
		Flags:  flag.NewFlagSet("users", flag.ContinueOnError),
	}
	return c, nil
}

func (c *DbUsersCommand) Name() string {
	return c.Flags.Name()
}

func (c *DbUsersCommand) Parent() Command {
	return c.parent
}

func (c *DbUsersCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *DbUsersCommand) Subcommands() []Command {
	return nil
}

func (c *DbUsersCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}

	cfg := c.Parent().(*DbCommand).parent.(*RootCommand).cfg
	if cfg.DBConnectionProvider == "" || cfg.DBConnectionString == "" {
		err := fmt.Errorf("database connection not configured")
		printHelp(c, err)
		return err
	}

	Config = cfg

	db, err := OpenDB()
	if err != nil {
		printHelp(c, err)
		return err
	}
	defer db.Close()

	rows, err := db.Query("SELECT user FROM passwords")
	if err != nil {
		printHelp(c, err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var user string
		if err := rows.Scan(&user); err != nil {
			printHelp(c, err)
			return err
		}
		fmt.Println(user)
	}

	return nil
}
