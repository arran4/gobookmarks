package main

import (
	"flag"
	"fmt"

	. "github.com/arran4/gobookmarks"
)

type DbUsersCommand struct {
	*RootCommand
	fs *flag.FlagSet
}

func NewDbUsersCommand(root *RootCommand) *DbUsersCommand {
	c := &DbUsersCommand{
		RootCommand: root,
		fs:          flag.NewFlagSet("users", flag.ExitOnError),
	}
	return c
}

func (c *DbUsersCommand) Name() string {
	return c.fs.Name()
}

func (c *DbUsersCommand) Fs() *flag.FlagSet {
	return c.fs
}

func (c *DbUsersCommand) Execute(args []string) error {
	c.fs.Parse(args)
	cfg := c.RootCommand.cfg

	if cfg.DBConnectionProvider == "" || cfg.DBConnectionString == "" {
		return fmt.Errorf("database connection not configured")
	}

	DBConnectionProvider = cfg.DBConnectionProvider
	DBConnectionString = cfg.DBConnectionString

	db, err := OpenDB()
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query("SELECT user FROM passwords")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var user string
		if err := rows.Scan(&user); err != nil {
			return err
		}
		fmt.Println(user)
	}

	return nil
}
