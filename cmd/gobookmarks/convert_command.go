package main

import (
	"bytes"
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
	Stdin    io.Reader
	Stdout   io.Writer
}

func (rc *RootCommand) NewConvertCommand() (*ConvertCommand, error) {
	c := &ConvertCommand{
		parent: rc,
		Flags:  flag.NewFlagSet("convert", flag.ContinueOnError),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
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

	var r io.Reader
	if filePath == "" || filePath == "-" {
		r = c.Stdin
	} else {
		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		defer func() { _ = f.Close() }()
		r = f
	}

	return Convert(c.From, c.To, r, c.Stdout)
}

func Convert(fromFormat, toFormat string, r io.Reader, w io.Writer) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	var list gobookmarks.BookmarkList
	if fromFormat == "bookmarks" {
		list, err = gobookmarks.StrictParseBookmarks(string(data))
		if err != nil {
			return fmt.Errorf("failed to parse bookmarks strictly: %w", err)
		}
	} else {
		var tabs []*gobookmarks.JSONTab
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&tabs); err != nil {
			return fmt.Errorf("failed to parse json (ensure no unknown fields are present): %w", err)
		}

		// Ensure there is exactly one JSON value and no trailing garbage.
		var remaining json.RawMessage
		if err := decoder.Decode(&remaining); err != io.EOF {
			return fmt.Errorf("failed to parse json: multiple top-level values or trailing non-whitespace data found")
		}

		list, err = gobookmarks.BookmarkListFromJSON(tabs)
		if err != nil {
			return fmt.Errorf("failed to construct bookmarks from json: %w", err)
		}
	}

	if toFormat == "json" {
		out, err := json.MarshalIndent(list.ToJSON(), "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format json: %w", err)
		}
		if _, err := fmt.Fprintln(w, string(out)); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	} else {
		if _, err := fmt.Fprint(w, list.String()); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	return nil
}
