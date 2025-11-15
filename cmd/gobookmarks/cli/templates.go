package cli

const helpTemplate = `{{if .Long}}{{.Long}}{{else}}{{.Short}}{{end}}

Usage:
  {{.FullPath}} [command]
{{if .SubCommands}}
Available Commands:
{{range .SubCommands}}  {{.Name | printf "%-11s"}} {{.Short}}
{{end}}{{end}}{{if .FlagUsages}}

Flags:
{{.FlagUsages}}{{end}}

Use "{{.FullPath}} [command] --help" for more information about a command.
`
