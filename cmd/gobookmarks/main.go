package main

import (
	"fmt"
	"log"
	"os"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "serve":
		serveCmd := NewServeCmd(version, commit, date)
		if err := serveCmd.Run(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "version":
		versionCmd := NewVersionCmd(version, commit, date)
		if err := versionCmd.Run(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "verify-file":
		verifyFileCmd := NewVerifyFileCmd()
		if err := verifyFileCmd.Run(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "verify-creds":
		verifyCredsCmd := NewVerifyCredsCmd()
		if err := verifyCredsCmd.Run(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "db":
		dbCmd := NewDbCmd()
		if err := dbCmd.Run(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "git":
		gitCmd := NewGitCmd()
		if err := gitCmd.Run(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "--help", "help":
		usage()
	default:
		fmt.Printf("unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: gobookmarks [command]")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  serve          Starts the gobookmarks server")
	fmt.Println("  version        Print the version number of gobookmarks")
	fmt.Println("  verify-file    Verify a bookmarks file")
	fmt.Println("  verify-creds   Verify OAuth2 credentials")
	fmt.Println("  db             Database inspection commands")
	fmt.Println("  git            Git storage inspection commands")
}
