
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
	c.fs.Usage = func() { printHelp(c, c.subcommands()...) }
	if len(args) < 1 {
		c.fs.Usage()
		return nil
	}

	var cmd Command
	subcommands := c.subcommands()
	for _, sub := range subcommands {
		if sub.Name() == args[0] {
			cmd = sub
			break
		}
	}

	if cmd == nil {
		if args[0] == "help" {
			if len(args) < 2 {
				c.fs.Usage()
				return nil
			}
			for _, sub := range subcommands {
				if sub.Name() == args[1] {
					sub.Fs().Usage()
					return nil
				}
			}
			return fmt.Errorf("unknown command: %s", args[1])
		}
		return fmt.Errorf("unknown command: %s", args[0])
	}

	cmd.Fs().Usage = func() { printHelp(cmd) }
	for _, arg := range args[1:] {
		if arg == "-h" || arg == "-help" {
			cmd.Fs().Usage()
			return nil
		}
	}

	if err := c.loadConfig(); err != nil {
		return err
	}

	return cmd.Execute(args[1:])
}

func (c *RootCommand) subcommands() []Command {
	return []Command{
		NewServeCommand(c),
		NewVersionCommand(c),
		NewDbCommand(c),
		NewVerifyFileCommand(c),
		NewVerifyCredsCommand(c),
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
