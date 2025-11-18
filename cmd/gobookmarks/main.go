
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	. "github.com/arran4/gobookmarks"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Command is the interface that all commands must implement.
type Command interface {
	Execute(args []string) error
	Name() string
	Fs() *flag.FlagSet
}

type RootCommand struct {
	fs *flag.FlagSet

	Config string
	cfg    Config
}

func NewRootCommand() *RootCommand {
	c := &RootCommand{
		fs: flag.NewFlagSet("gobookmarks", flag.ExitOnError),
	}
	c.fs.StringVar(&c.Config, "config", "", "path to config file")
	return c
}

func (c *RootCommand) Run(args []string) error {
	c.fs.Parse(args)
	if err := c.loadConfig(); err != nil {
		return err
	}

	if c.fs.NArg() < 1 {
		printHelp(c, c.subcommands()...)
		return nil
	}

	var cmd Command
	subcommands := c.subcommands()
	cmdName := c.fs.Arg(0)
	for _, sub := range subcommands {
		if sub.Name() == cmdName {
			cmd = sub
			break
		}
	}

	if cmd == nil {
		if cmdName == "help" {
			if c.fs.NArg() < 2 {
				printHelp(c, c.subcommands()...)
				return nil
			}
			cmdName = c.fs.Arg(1)
			for _, sub := range subcommands {
				if sub.Name() == cmdName {
					printHelp(sub)
					return nil
				}
			}
			return fmt.Errorf("unknown command: %s", cmdName)
		}
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	cmd.Fs().Usage = func() { printHelp(cmd) }
	return cmd.Execute(c.fs.Args()[1:])
}

func (c *RootCommand) subcommands() []Command {
	return []Command{
		NewServeCommand(c),
		NewVersionCommand(c),
		NewDbCommand(c),
		NewVerifyFileCommand(c),
		NewVerifyCredsCommand(c),
		NewImportCommand(c),
		NewExportCommand(c),
	}
}

func (c *RootCommand) loadConfig() error {
	envPath := os.Getenv("GOBM_ENV_FILE")
	if envPath == "" {
		envPath = "/etc/gobookmarks/gobookmarks.env"
	}
	if err := LoadEnvFile(envPath); err != nil {
		log.Printf("unable to load env file %s: %v", envPath, err)
	}

	c.cfg = Config{
		GithubClientID:       os.Getenv("GITHUB_CLIENT_ID"),
		GithubSecret:         os.Getenv("GITHUB_SECRET"),
		GitlabClientID:       os.Getenv("GITLAB_CLIENT_ID"),
		GitlabSecret:         os.Getenv("GITLAB_SECRET"),
		DBConnectionProvider: os.Getenv("DB_CONNECTION_PROVIDER"),
		DBConnectionString:   os.Getenv("DB_CONNECTION_STRING"),
	}

	configPath := DefaultConfigPath()
	if c.Config != "" {
		configPath = c.Config
	}

	cfgSpecified := c.Config != "" || os.Getenv("GOBM_CONFIG_FILE") != ""
	if fileCfg, found, err := LoadConfigFile(configPath); err == nil {
		if found {
			MergeConfig(&c.cfg, fileCfg)
		}
	} else {
		if os.IsNotExist(err) && !cfgSpecified {
			log.Printf("config file %s not found", configPath)
		} else {
			return fmt.Errorf("unable to load config file %s: %w", configPath, err)
		}
	}
	return nil
}

func (c *RootCommand) Name() string {
	return "gobookmarks"
}

func (c *RootCommand) Fs() *flag.FlagSet {
	return c.fs
}

func (c *RootCommand) Execute(args []string) error {
	return c.Run(args)
}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	root := NewRootCommand()
	if err := root.Run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func printUsage() {
	NewRootCommand().fs.Usage()
}
