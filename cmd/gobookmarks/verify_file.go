package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func NewVerifyFileCmd() *Command {
	return &Command{
		Name:  "verify-file",
		Short: "Verify a bookmarks file",
		Long:  `Reads a bookmarks file and checks it for errors by attempting to unmarshal it as JSON.`,
		Run: func(cmd *Command, args []string) error {
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
		},
	}
}
