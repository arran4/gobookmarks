package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"text/template"
)

// Command represents a command-line command.
type Command struct {
	*Command
	Name        string
	Short       string
	Long        string
	Run         func(cmd *Command, args []string) error
	SubCommands []*Command
	FlagSet     *flag.FlagSet
}

// AddCommand adds a subcommand to this command.
func (c *Command) AddCommand(sub *Command) {
	sub.Command = c
	c.SubCommands = append(c.SubCommands, sub)
}

// Execute executes the command.
func (c *Command) Execute(args []string) error {
	if c.FlagSet == nil {
		c.FlagSet = flag.NewFlagSet(c.Name, flag.ExitOnError)
	}

	if len(args) > 0 {
		for _, sub := range c.SubCommands {
			if sub.Name == args[0] {
				return sub.Execute(args[1:])
			}
		}
	}

	if err := c.FlagSet.Parse(args); err != nil {
		return err
	}

	if c.Run != nil {
		return c.Run(c, c.FlagSet.Args())
	}

	return c.Usage()
}

// Usage prints the command's usage to stderr.
func (c *Command) Usage() error {
	return c.UsageTo(os.Stderr)
}

// UsageTo prints the command's usage to the given writer.
func (c *Command) UsageTo(w io.Writer) error {
	tmpl := template.Must(template.New("usage").Parse(helpTemplate))
	return tmpl.Execute(w, c)
}

// FullPath returns the full command path.
func (c *Command) FullPath() string {
	if c.Command == nil {
		return c.Name
	}
	return fmt.Sprintf("%s %s", c.Command.FullPath(), c.Name)
}

// FlagUsages returns a string containing the usage information for all flags.
func (c *Command) FlagUsages() string {
	var buf bytes.Buffer
	c.FlagSet.SetOutput(&buf)
	c.FlagSet.PrintDefaults()
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
