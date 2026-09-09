package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	gobookmarks "github.com/arran4/gobookmarks"
)

type ConvertCommand struct {
	parent   Command
	Flags    *flag.FlagSet
	From     string
	To       string
	FromFile string
}

func (rc *RootCommand) NewConvertCommand() (*ConvertCommand, error) {
	c := &ConvertCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("convert", flag.ContinueOnError),
	}
	c.Flags.StringVar(&c.From, "from", "bookmarks", "source format: bookmarks or json")
	c.Flags.StringVar(&c.To, "to", "json", "destination format: bookmarks or json")
	return c, nil
}

func (c *ConvertCommand) Name() string {
	return c.Flags.Name()
}

func (c *ConvertCommand) Parent() Command {
	return c.parent
}

func (c *ConvertCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *ConvertCommand) Subcommands() []Command {
	return nil
}

func (c *ConvertCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	if forwardHelpIfRequested(c, args) {
		return nil
	}

	if c.From != "bookmarks" && c.From != "json" {
		err := fmt.Errorf("invalid from format: %s", c.From)
		printHelp(c, err)
		return err
	}

	if c.To != "bookmarks" && c.To != "json" {
		err := fmt.Errorf("invalid to format: %s", c.To)
		printHelp(c, err)
		return err
	}

	filePath := ""
	if c.FlagSet().NArg() > 0 {
		filePath = c.FlagSet().Arg(0)
	}

	var data []byte
	var err error
	if filePath == "" || filePath == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(filePath)
	}
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	var list gobookmarks.BookmarkList
	if c.From == "bookmarks" {
		list, err = gobookmarks.StrictParseBookmarks(string(data))
		if err != nil {
			return fmt.Errorf("failed to parse bookmarks strictly: %w", err)
		}
	} else {
		var tabs []*gobookmarks.JSONTab
		if err := json.Unmarshal(data, &tabs); err != nil {
			return fmt.Errorf("failed to parse json: %w", err)
		}
		list, err = gobookmarks.BookmarkListFromJSON(tabs)
		if err != nil {
			return fmt.Errorf("failed to construct bookmarks from json: %w", err)
		}
	}

	if c.To == "json" {
		out, err := json.MarshalIndent(list.ToJSON(), "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format json: %w", err)
		}
		fmt.Println(string(out))
	} else {
		fmt.Print(list.String())
	}

	return nil
}
