package templates

import (
	_ "embed"
)

//go:embed help.tpl
var HelpTemplate string
