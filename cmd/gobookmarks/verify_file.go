package main

import (
	"encoding/json"
	"fmt"
	"os"
	"flag"
)

func NewVerifyFileCmd() *VerifyFileCommand {
	return &VerifyFileCommand{
		FlagSet: flag.NewFlagSet("verify-file", flag.ExitOnError),
	}
}

func (c *VerifyFileCommand) Run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("verify-file requires exactly one argument: the path to the file")
	}
	filepath := args[0]
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("file is not a valid JSON file: %w", err)
	}

	fmt.Println("File is a valid JSON file.")
	return nil
}

func (c *VerifyFileCommand) Usage() {
	printUsage(c.FlagSet, "verify-file", "Verify a bookmarks file", `Reads a bookmarks file and checks it for errors by attempting to unmarshal it as JSON.`)
}
