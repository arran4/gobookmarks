package main

import (
	"bytes"
	"os"
)

type helpData struct {
	Name           string
	HasSubcommands bool
	Subcommands    []Command
	Flags          string
}

func printHelp(cmd Command, subcommands ...Command) {
	fs := cmd.Fs()

	var flags bytes.Buffer
	fs.SetOutput(&flags)
	fs.PrintDefaults()

	data := helpData{
		Name:           cmd.Name(),
		HasSubcommands: len(subcommands) > 0,
		Subcommands:    subcommands,
		Flags:          flags.String(),
	}

	err := GetTemplates().ExecuteTemplate(os.Stdout, "help.go.tmpl", data)
	if err != nil {
		panic(err)
	}
}
