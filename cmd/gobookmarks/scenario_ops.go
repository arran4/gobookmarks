package main

import (
	"context"
	"fmt"
	"strings"

	gobookmarks "github.com/arran4/gobookmarks"
)

type ScenarioContext struct {
	Refs  map[string]string
	Files map[string]string
}

type Operation interface {
	Validate(e *Event) error
	Apply(ctx context.Context, sCtx *ScenarioContext, e *Event) error
}

var operations = map[string]Operation{
	"user.create":     &UserCreateOp{},
	"repo.create":     &RepoCreateOp{},
	"bookmark.create": &BookmarkCreateOp{},
}

func getScenarioProvider(e *Event) (gobookmarks.Provider, error) {
	providerName := e.Props["Provider"]
	if providerName == "" {
		providerName = "sql"
	}
	p := gobookmarks.GetProvider(providerName)
	if p == nil {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}
	return p, nil
}

type userCreator interface {
	CreateUser(ctx context.Context, user, password string) error
}

type UserCreateOp struct{}

func (o *UserCreateOp) Validate(e *Event) error {
	if e.Props["Username"] == "" {
		return fmt.Errorf("missing Username")
	}
	return nil
}

func (o *UserCreateOp) Apply(ctx context.Context, sCtx *ScenarioContext, e *Event) error {
	p, err := getScenarioProvider(e)
	if err != nil {
		return err
	}

	uc, ok := p.(userCreator)
	if !ok {
		return fmt.Errorf("provider %s does not support user creation", p.Name())
	}

	username := e.Props["Username"]
	if err := uc.CreateUser(ctx, username, "password"); err != nil {
		if err != gobookmarks.ErrUserExists {
			return err
		}
	}

	if e.Ref != "" {
		sCtx.Refs[e.Ref] = username
	}

	return nil
}

type RepoCreateOp struct{}

func (o *RepoCreateOp) Validate(e *Event) error {
	if e.Props["User"] == "" {
		return fmt.Errorf("missing User")
	}
	if e.Props["Name"] == "" {
		return fmt.Errorf("missing Name")
	}
	return nil
}

func (o *RepoCreateOp) Apply(ctx context.Context, sCtx *ScenarioContext, e *Event) error {
	p, err := getScenarioProvider(e)
	if err != nil {
		return err
	}

	userRef := e.Props["User"]
	username, ok := sCtx.Refs[userRef]
	if !ok {
		return fmt.Errorf("unknown user ref: %s", userRef)
	}

	name := e.Props["Name"]

	if err := p.CreateRepo(ctx, username, nil, name); err != nil {
		return err
	}

	if e.Ref != "" {
		sCtx.Refs[e.Ref] = name
	}

	return nil
}

type BookmarkCreateOp struct{}

func (o *BookmarkCreateOp) Validate(e *Event) error {
	if e.Props["User"] == "" {
		return fmt.Errorf("missing User")
	}
	return nil
}

func (o *BookmarkCreateOp) Apply(ctx context.Context, sCtx *ScenarioContext, e *Event) error {
	p, err := getScenarioProvider(e)
	if err != nil {
		return err
	}

	userRef := e.Props["User"]
	username, ok := sCtx.Refs[userRef]
	if !ok {
		return fmt.Errorf("unknown user ref: %s", userRef)
	}

	branch := e.Props["Branch"]
	if branch == "" {
		branch = "main"
	}

	body := strings.TrimSpace(e.Body)
	if e.Props["Asset"] != "" {
		assetBody, ok := sCtx.Files[e.Props["Asset"]]
		if !ok {
			return fmt.Errorf("missing asset: %s", e.Props["Asset"])
		}
		body = strings.TrimSpace(assetBody)
	}

	if err := p.CreateBookmarks(ctx, username, nil, branch, body); err != nil {
		return err
	}

	return nil
}

func ValidateScenario(s *Scenario) error {
	for _, e := range s.Events {
		op, ok := operations[e.Op]
		if !ok {
			return fmt.Errorf("unknown operation: %s", e.Op)
		}
		if err := op.Validate(e); err != nil {
			return fmt.Errorf("event %s: %w", e.Name, err)
		}
	}
	return nil
}

func ApplyScenario(ctx context.Context, s *Scenario) error {
	sCtx := &ScenarioContext{
		Refs:  make(map[string]string),
		Files: s.Files,
	}

	for _, e := range s.Events {
		op := operations[e.Op]
		if err := op.Apply(ctx, sCtx, e); err != nil {
			return fmt.Errorf("event %s: %w", e.Name, err)
		}
	}
	return nil
}
