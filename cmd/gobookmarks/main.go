package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	gobookmarks "github.com/arran4/gobookmarks"
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
	cfg         gobookmarks.Configuration
	VersionInfo VersionInfo

	ServeCmd       *ServeCommand
	VersionCmd     *VersionCommand
	DbCmd          *DbCommand
	LintCmd        *LintCommand
	VerifyFileCmd  *VerifyFileCommand
	VerifyCredsCmd *VerifyCredsCommand
	ImportCmd      *ImportCommand
	ExportCmd      *ExportCommand
	TestCmd        *TestCommand
	ConvertCmd     *ConvertCommand
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
	rc.LintCmd, _ = rc.NewLintCommand()
	rc.VerifyFileCmd, _ = rc.NewVerifyFileCommand()
	rc.VerifyCredsCmd, _ = rc.NewVerifyCredsCommand()
	rc.ImportCmd, _ = rc.NewImportCommand()
	rc.ExportCmd, _ = rc.NewExportCommand()
	rc.TestCmd, _ = rc.NewTestCommand()
	rc.ConvertCmd, _ = rc.NewConvertCommand()
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
	return []Command{c.ServeCmd, c.VersionCmd, c.DbCmd, c.LintCmd, c.VerifyFileCmd, c.VerifyCredsCmd, c.ImportCmd, c.ExportCmd, c.TestCmd, c.ConvertCmd, c.HelpCmd}
}

func (c *RootCommand) Execute(args []string) error {
	c.Flags.Usage = func() { printHelp(c, nil) }
	if err := c.Flags.Parse(args); err != nil {
		printHelp(c, err)
		return err
	}
	remaining := c.Flags.Args()
	if len(remaining) == 0 {
		printHelp(c, nil)
		return nil
	}

	loadCfg := false
	switch remaining[0] {
	case "-h", "--help", "help":
		return c.HelpCmd.Execute(remaining[1:])
	case c.VersionCmd.Name():
		return c.VersionCmd.Execute(remaining[1:])
	case c.TestCmd.Name():
		return c.TestCmd.Execute(remaining[1:])
	case c.ServeCmd.Name(), c.DbCmd.Name(), c.VerifyCredsCmd.Name(), c.ImportCmd.Name(), c.ExportCmd.Name():
		loadCfg = true
	case c.LintCmd.Name(), c.VerifyFileCmd.Name(), c.ConvertCmd.Name():
		// lint / verify-file / convert do not load configuration
	default:
		err := fmt.Errorf("unknown command: %s", remaining[0])
		printHelp(c, err)
		return err
	}

	if loadCfg {
		if err := c.loadConfig(); err != nil {
			printHelp(c, err)
			return err
		}
	}
	switch remaining[0] {
	case c.ServeCmd.Name():
		return c.ServeCmd.Execute(remaining[1:])
	case c.DbCmd.Name():
		return c.DbCmd.Execute(remaining[1:])
	case c.LintCmd.Name():
		return c.LintCmd.Execute(remaining[1:])
	case c.VerifyFileCmd.Name():
		// run the exact same logic as lint
		return c.LintCmd.Execute(remaining[1:])
	case c.VerifyCredsCmd.Name():
		return c.VerifyCredsCmd.Execute(remaining[1:])
	case c.ImportCmd.Name():
		return c.ImportCmd.Execute(remaining[1:])
	case c.ExportCmd.Name():
		return c.ExportCmd.Execute(remaining[1:])
	case c.ConvertCmd.Name():
		return c.ConvertCmd.Execute(remaining[1:])
	}
	return nil
}

func (c *RootCommand) loadConfig(ops ...any) error {
	var env gobookmarks.Environment = gobookmarks.DefaultEnvironment{}
	for _, opt := range ops {
		if e, ok := opt.(gobookmarks.Environment); ok {
			env = e
		}
	}
	envPath := env.Getenv("GOBM_ENV_FILE")
	if envPath == "" {
		envPath = "/etc/gobookmarks/gobookmarks.env"
	}
	if err := gobookmarks.LoadEnvFile(envPath, ops...); err != nil {
		log.Printf("unable to load env file %s: %v", envPath, err)
	}

	c.cfg = gobookmarks.Configuration{
		GithubClientID:       env.Getenv("GITHUB_CLIENT_ID"),
		GithubSecret:         env.Getenv("GITHUB_SECRET"),
		GitlabClientID:       env.Getenv("GITLAB_CLIENT_ID"),
		GitlabSecret:         env.Getenv("GITLAB_SECRET"),
		ExternalURL:          env.Getenv("EXTERNAL_URL"),
		CSSColumns:           getenvSet("GBM_CSS_COLUMNS", ops...),
		DevMode:              getenvBoolPtr("GBM_DEV_MODE", ops...),
		Namespace:            env.Getenv("GBM_NAMESPACE"),
		Title:                env.Getenv("GBM_TITLE"),
		GithubServer:         env.Getenv("GITHUB_SERVER"),
		GitlabServer:         env.Getenv("GITLAB_SERVER"),
		FaviconCacheDir:      env.Getenv("FAVICON_CACHE_DIR"),
		FaviconCacheSize:     getenvInt64("FAVICON_CACHE_SIZE", ops...),
		FaviconMaxCacheCount: getenvInt("FAVICON_MAX_CACHE_COUNT", ops...),
		LocalGitPath:         env.Getenv("LOCAL_GIT_PATH"),
		NoFooter:             getenvBool("GBM_NO_FOOTER", ops...),
		SessionKey:           env.Getenv("SESSION_KEY"),
		SessionName:          env.Getenv("SESSION_NAME"),
		DBConnectionProvider: env.Getenv("DB_CONNECTION_PROVIDER"),
		DBConnectionString:   env.Getenv("DB_CONNECTION_STRING"),
		ProviderOrder:        getenvStringSlice("PROVIDER_ORDER", ops...),
		CommitsPerPage:       getenvInt("COMMITS_PER_PAGE", ops...),
	}

	configPath := gobookmarks.DefaultConfigPath(ops...)
	if envCfg := env.Getenv("GOBM_CONFIG_FILE"); envCfg != "" {
		configPath = envCfg
	}
	if c.ConfigPath != "" {
		configPath = c.ConfigPath
	}

	cfgSpecified := c.ConfigPath != "" || env.Getenv("GOBM_CONFIG_FILE") != ""
	found, err := gobookmarks.LoadConfigFileInto(&c.cfg, configPath, ops...)
	if err != nil {
		return fmt.Errorf("unable to load config file %s: %w", configPath, err)
	}
	if !found && cfgSpecified {
		return fmt.Errorf("unable to load config file %s: not found", configPath)
	}
	return nil
}

func printHelp(cmd Command, err error) {
	fmt.Print(renderTemplate(cmd, err))
}

func getenvSet(key string, ops ...any) bool {
	var env gobookmarks.Environment = gobookmarks.DefaultEnvironment{}
	for _, opt := range ops {
		if e, ok := opt.(gobookmarks.Environment); ok {
			env = e
		}
	}
	val := env.Getenv(key)
	return val != ""
}

func getenvBool(key string, ops ...any) bool {
	var env gobookmarks.Environment = gobookmarks.DefaultEnvironment{}
	for _, opt := range ops {
		if e, ok := opt.(gobookmarks.Environment); ok {
			env = e
		}
	}
	val := env.Getenv(key)
	if val == "" {
		return false
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return true // fallback per original logic
	}
	return b
}

func getenvBoolPtr(key string, ops ...any) *bool {
	var env gobookmarks.Environment = gobookmarks.DefaultEnvironment{}
	for _, opt := range ops {
		if e, ok := opt.(gobookmarks.Environment); ok {
			env = e
		}
	}
	val := env.Getenv(key)
	if val == "" {
		return nil
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		t := true
		return &t
	}
	return &b
}

func getenvInt(key string, ops ...any) int {
	var env gobookmarks.Environment = gobookmarks.DefaultEnvironment{}
	for _, opt := range ops {
		if e, ok := opt.(gobookmarks.Environment); ok {
			env = e
		}
	}
	val := env.Getenv(key)
	if val == "" {
		return 0
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return i
}

func getenvInt64(key string, ops ...any) int64 {
	var env gobookmarks.Environment = gobookmarks.DefaultEnvironment{}
	for _, opt := range ops {
		if e, ok := opt.(gobookmarks.Environment); ok {
			env = e
		}
	}
	val := env.Getenv(key)
	if val == "" {
		return 0
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

func getenvStringSlice(key string, ops ...any) []string {
	var env gobookmarks.Environment = gobookmarks.DefaultEnvironment{}
	for _, opt := range ops {
		if e, ok := opt.(gobookmarks.Environment); ok {
			env = e
		}
	}
	val := env.Getenv(key)
	if val == "" {
		return nil
	}
	var res []string
	for _, s := range strings.Split(val, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			res = append(res, s)
		}
	}
	return res
}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	root := NewRootCommand()
	if err := root.Execute(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
