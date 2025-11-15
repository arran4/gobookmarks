package main

import (
	"bytes"
	"text/template"
	"flag"
	"os"
)

const helpTemplate = `{{if .Long}}{{.Long}}{{else}}{{.Short}}{{end}}

Usage:
  {{.Name}} [flags]
{{if .FlagUsages}}
Flags:
{{.FlagUsages}}{{end}}
`

func (c *ServeCommand) Usage() {
	printUsage(c.FlagSet, "serve", "Starts the gobookmarks server", `Starts the gobookmarks server, serving the web UI and API.`)
}

func printUsage(fs *flag.FlagSet, name, short, long string) {
	var buf bytes.Buffer
	fs.SetOutput(&buf)
	fs.PrintDefaults()

	tmpl := template.Must(template.New("usage").Parse(helpTemplate))
	tmpl.Execute(os.Stdout, struct {
		Name       string
		Short      string
		Long       string
		FlagUsages string
	}{
		Name:       name,
		Short:      short,
		Long:       long,
		FlagUsages: buf.String(),
	})
}
