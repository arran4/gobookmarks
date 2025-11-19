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
	FlagSet() *flag.FlagSet
	Parent() Command
	Subcommands() []Command
}

type VersionInfo struct {
	Version string
	Commit  string
	Date    string
}

type RootCommand struct {
	Flags       *flag.FlagSet
	ConfigPath  string
	cfg         Config
	VersionInfo VersionInfo

	ServeCmd       *ServeCommand
	VersionCmd     *VersionCommand
	DbCmd          *DbCommand
	VerifyFileCmd  *VerifyFileCommand
	VerifyCredsCmd *VerifyCredsCommand
	ImportCmd      *ImportCommand
	ExportCmd      *ExportCommand
	HelpCmd        *HelpCommand
}

func NewRootCommand() *RootCommand {
	rc := &RootCommand{
		Flags:       flag.NewFlagSet("gobookmarks", flag.ContinueOnError),
		VersionInfo: VersionInfo{Version: version, Commit: commit, Date: date},
	}
	rc.Flags.StringVar(&rc.ConfigPath, "config", "", "path to config file")

	rc.ServeCmd, _ = rc.NewServeCommand()
	rc.VersionCmd, _ = rc.NewVersionCommand()
	rc.DbCmd, _ = rc.NewDbCommand()
	rc.VerifyFileCmd, _ = rc.NewVerifyFileCommand()
	rc.VerifyCredsCmd, _ = rc.NewVerifyCredsCommand()
	rc.ImportCmd, _ = rc.NewImportCommand()
	rc.ExportCmd, _ = rc.NewExportCommand()
	rc.HelpCmd = NewHelpCommand(rc)
	return rc
}

func (c *RootCommand) Name() string {
	return c.Flags.Name()
}

func (c *RootCommand) Parent() Command {
	return nil
}

func (c *RootCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *RootCommand) Subcommands() []Command {
	return []Command{c.ServeCmd, c.VersionCmd, c.DbCmd, c.VerifyFileCmd, c.VerifyCredsCmd, c.ImportCmd, c.ExportCmd, c.HelpCmd}
}

func (c *RootCommand) Execute(args []string) error {
	c.Flags.Usage = func() { printHelp(c, nil) }
	if err := c.Flags.Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	if err := c.loadConfig(); err != nil {
		printHelp(c, err)
		return err
	}

	remaining := c.Flags.Args()
	if len(remaining) == 0 {
		printHelp(c, nil)
		return nil
	}

	switch remaining[0] {
	case "-h", "--help", "help":
		return c.HelpCmd.Execute(remaining[1:])
	case c.ServeCmd.Name():
		return c.ServeCmd.Execute(remaining[1:])
	case c.VersionCmd.Name():
		return c.VersionCmd.Execute(remaining[1:])
	case c.DbCmd.Name():
		return c.DbCmd.Execute(remaining[1:])
	case c.VerifyFileCmd.Name():
		return c.VerifyFileCmd.Execute(remaining[1:])
	case c.VerifyCredsCmd.Name():
		return c.VerifyCredsCmd.Execute(remaining[1:])
	case c.ImportCmd.Name():
		return c.ImportCmd.Execute(remaining[1:])
	case c.ExportCmd.Name():
		return c.ExportCmd.Execute(remaining[1:])
	default:
		err := fmt.Errorf("unknown command: %s", remaining[0])
		printHelp(c, err)
		return err
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
	if c.ConfigPath != "" {
		configPath = c.ConfigPath
	}

	cfgSpecified := c.ConfigPath != "" || os.Getenv("GOBM_CONFIG_FILE") != ""
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

func printHelp(cmd Command, err error) {
	fmt.Print(renderTemplate(cmd, err))
}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	root := NewRootCommand()
	if err := root.Execute(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
