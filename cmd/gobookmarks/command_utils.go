package main

import "strings"

func formatCommandPath(cmd Command) string {
	names := []string{}
	current := cmd
	for current != nil {
		names = append([]string{current.Name()}, names...)
		current = current.Parent()
	}
	return strings.Join(names, " ")
}

func forwardHelpIfRequested(cmd Command, args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			printHelp(cmd, nil)
			return true
		}
	}
	return false
}
