package main

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
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
			"description": description,
			"firstLine":   firstLine,
		}).ParseFS(templateFS, "templates/*.gotmpl", "templates/partials/*.gotmpl"))
	})
	return templates
}

func renderTemplate(cmd Command, err error) string {
	data := templateContext(cmd, err)

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

func description(cmd Command) string {
	tpl := getTemplates().Lookup(fmt.Sprintf("description/%s", cmd.Name()))
	if tpl == nil {
		return ""
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, templateContext(cmd, nil)); err != nil {
		return ""
	}

	return strings.TrimSpace(out.String())
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

func templateContext(cmd Command, err error) templateData {
	var flags bytes.Buffer
	if fs := cmd.FlagSet(); fs != nil {
		fs.SetOutput(&flags)
		fs.PrintDefaults()
	}

	return templateData{
		Command:     cmd,
		Parent:      cmd.Parent(),
		Subcommands: cmd.Subcommands(),
		FlagsOutput: flags.String(),
		Error:       err,
	}
}
