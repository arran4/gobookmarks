package main

import (
	"bytes"
	"embed"
	"fmt"
	"sync"
	"text/template"
)

//go:embed templates/*.gotmpl templates/partials/*.gotmpl
var templateFS embed.FS

var (
	templates     *template.Template
	templatesOnce sync.Once
)

type templateData struct {
	Command     Command
	Parent      Command
	Subcommands []Command
	FlagsOutput string
	Error       error
}

func getTemplates() *template.Template {
	templatesOnce.Do(func() {
		templates = template.Must(template.New("root").Funcs(template.FuncMap{
			"commandPath": func(cmd Command) string { return formatCommandPath(cmd) },
		}).ParseFS(templateFS, "templates/*.gotmpl", "templates/partials/*.gotmpl"))
	})
	return templates
}

func renderTemplate(cmd Command, err error) string {
	var flags bytes.Buffer
	if fs := cmd.FlagSet(); fs != nil {
		fs.SetOutput(&flags)
		fs.PrintDefaults()
	}

	data := templateData{
		Command:     cmd,
		Parent:      cmd.Parent(),
		Subcommands: cmd.Subcommands(),
		FlagsOutput: flags.String(),
		Error:       err,
	}

	tpl := getTemplates().Lookup(fmt.Sprintf("%s.gotmpl", cmd.Name()))
	if tpl == nil {
		return fmt.Sprintf("missing help template for %s", cmd.Name())
	}
	var out bytes.Buffer
	if execErr := tpl.Execute(&out, data); execErr != nil {
		return execErr.Error()
	}
	return out.String()
}
