package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang.org/x/tools/txtar"
)

type Scenario struct {
	Preamble string
	Manifest map[string]string
	// StorageProvider owns repositories and bookmark history. AuthProvider is
	// the identity used when a scenario exercises an external login flow.
	StorageProvider string
	AuthProvider    string
	AuthUser        string
	Events          []*Event
	Files           map[string]string
}

type Event struct {
	Name  string
	Op    string
	Ref   string
	Props map[string]string
	Body  string
}

func ParseScenario(r io.Reader) (*Scenario, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	ar := txtar.Parse(data)

	s := &Scenario{
		Preamble: string(ar.Comment),
		Manifest: make(map[string]string),
		Files:    make(map[string]string),
	}

	for _, f := range ar.Files {
		if strings.HasSuffix(f.Name, ".meta") {
			props, _, err := parsePropsAndBody(f.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s: %w", f.Name, err)
			}
			for k, v := range props {
				s.Manifest[k] = v
			}
		} else if strings.HasSuffix(f.Name, ".event") {
			props, body, err := parsePropsAndBody(f.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s: %w", f.Name, err)
			}
			e := &Event{
				Name:  f.Name,
				Op:    props["Op"],
				Ref:   props["Ref"],
				Props: props,
				Body:  body,
			}
			s.Events = append(s.Events, e)
		} else {
			s.Files[f.Name] = string(f.Data)
		}
	}
	// SQL is deliberately the default for small local fixtures.  Authentication
	// and persistence are separate: external identities never imply remote user
	// creation.
	s.StorageProvider = s.Manifest["StorageProvider"]
	if s.StorageProvider == "" {
		s.StorageProvider = "sql"
	}
	s.AuthProvider = s.Manifest["AuthProvider"]
	if s.AuthProvider == "" {
		s.AuthProvider = s.StorageProvider
	}
	s.AuthUser = s.Manifest["AuthUser"]

	return s, nil
}

func parsePropsAndBody(data []byte) (map[string]string, string, error) {
	props := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var bodyBuilder strings.Builder
	inBody := false

	for scanner.Scan() {
		line := scanner.Text()
		if inBody {
			bodyBuilder.WriteString(line)
			bodyBuilder.WriteString("\n")
			continue
		}

		if line == "" {
			inBody = true
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			props[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		} else {
			inBody = true
			bodyBuilder.WriteString(line)
			bodyBuilder.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", err
	}

	return props, bodyBuilder.String(), nil
}
