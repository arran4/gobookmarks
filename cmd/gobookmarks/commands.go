package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"text/template"
)

// Command is the interface that all commands must implement.
type Command interface {
	Name() string
	Short() string
	Long() string
	Run(args []string) error
	SetFlagSet(*flag.FlagSet)
	FlagSet() *flag.FlagSet
	AddCommand(Command)
	SubCommands() []Command
	SetParent(Command)
	Parent() Command
	FullPath() string
}

// RootCommand is the root command of the application.
type RootCommand struct {
	flagSet     *flag.FlagSet
	subCommands []Command
	Version     string
	Commit      string
	Date        string
}

func (c *RootCommand) Name() string                     { return "gobookmarks" }
func (c *RootCommand) Short() string                    { return "A bookmark manager" }
func (c *RootCommand) Long() string                     { return `gobookmarks is a self-hosted bookmark manager written in Go.` }
func (c *RootCommand) Run(args []string) error          { return c.Usage() }
func (c *RootCommand) SetFlagSet(fs *flag.FlagSet)      { c.flagSet = fs }
func (c *RootCommand) FlagSet() *flag.FlagSet           { return c.flagSet }
func (c *RootCommand) AddCommand(sub Command)           { c.subCommands = append(c.subCommands, sub) }
func (c *RootCommand) SubCommands() []Command           { return c.subCommands }
func (c *RootCommand) SetParent(parent Command)         {}
func (c *RootCommand) Parent() Command                  { return nil }

func (c *RootCommand) Execute(args []string) error {
	if c.flagSet == nil {
		c.flagSet = flag.NewFlagSet(c.Name(), flag.ExitOnError)
	}

	if len(args) > 0 {
		for _, sub := range c.subCommands {
			if sub.Name() == args[0] {
				sub.SetFlagSet(c.flagSet)
				return sub.Run(args[1:])
			}
		}
	}

	if err := c.flagSet.Parse(args); err != nil {
		return err
	}

	return c.Run(c.flagSet.Args())
}

func (c *RootCommand) Usage() error {
	return c.UsageTo(os.Stderr)
}

func (c *RootCommand) UsageTo(w io.Writer) error {
	tmpl := template.Must(template.New("usage").Parse(helpTemplate))
	return tmpl.Execute(w, c)
}

func (c *RootCommand) FullPath() string {
	if c.Parent() == nil {
		return c.Name()
	}
	return fmt.Sprintf("%s %s", c.Parent().FullPath(), c.Name())
}

func (c *RootCommand) FlagUsages() string {
	var buf bytes.Buffer
	c.FlagSet().SetOutput(&buf)
	c.FlagSet().PrintDefaults()
	return buf.String()
}

const helpTemplate = `{{if .Long}}{{.Long}}{{else}}{{.Short}}{{end}}

Usage:
  {{.FullPath}} [command]
{{if .SubCommands}}
Available Commands:
{{range .SubCommands}}  {{.Name | printf "%-11s"}} {{.Short}}
{{end}}{{end}}{{if .FlagUsages}}

Flags:
{{.FlagUsages}}{{end}}

Use "{{.FullPath}} [command] --help" for more information about a command.
`

// ServeCommand is the serve command.
type ServeCommand struct {
	*RootCommand
}

// VersionCommand is the version command.
type VersionCommand struct {
	*RootCommand
}

// VerifyFileCommand is the verify-file command.
type VerifyFileCommand struct {
	*RootCommand
}

// VerifyCredsCommand is the verify-creds command.
type VerifyCredsCommand struct {
	*RootCommand
}

// DbCommand is the db command.
type DbCommand struct {
	*RootCommand
}

// GitCommand is the git command.
type GitCommand struct {
	*RootCommand
}
