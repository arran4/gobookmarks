package main

func NewRootCommand(version, commit, date string) *RootCommand {
	return &RootCommand{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
}

func (c *RootCommand) AddServeCmd() {
	cmd := &ServeCommand{RootCommand: c}
	c.AddCommand(cmd)
}

func (c *RootCommand) AddVersionCmd() {
	cmd := &VersionCommand{RootCommand: c}
	c.AddCommand(cmd)
}

func (c *RootCommand) AddVerifyFileCmd() {
	cmd := &VerifyFileCommand{RootCommand: c}
	c.AddCommand(cmd)
}

func (c *RootCommand) AddVerifyCredsCmd() {
	cmd := &VerifyCredsCommand{RootCommand: c}
	c.AddCommand(cmd)
}

func (c *RootCommand) AddDbCmd() {
	cmd := &DbCommand{RootCommand: c}
	c.AddCommand(cmd)
}

func (c *RootCommand) AddGitCmd() {
	cmd := &GitCommand{RootCommand: c}
	c.AddCommand(cmd)
}
