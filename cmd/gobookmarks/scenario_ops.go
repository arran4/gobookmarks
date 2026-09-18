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

type UserCreateOp struct{}

func (o *UserCreateOp) Validate(e *Event) error {
	if e.Props["Username"] == "" {
		return fmt.Errorf("missing Username")
	}
	return nil
}

func (o *UserCreateOp) Apply(ctx context.Context, sCtx *ScenarioContext, e *Event) error {
	p := gobookmarks.GetProvider("sql")
	sqlP, ok := p.(*gobookmarks.SQLProvider)
	if !ok {
		return fmt.Errorf("provider is not SQLProvider")
	}

	username := e.Props["Username"]
	// Use a default password for tests/scenarios
	if err := sqlP.CreateUser(ctx, username, "password"); err != nil {
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
	p := gobookmarks.GetProvider("sql")
	sqlP, ok := p.(*gobookmarks.SQLProvider)
	if !ok {
		return fmt.Errorf("provider is not SQLProvider")
	}

	userRef := e.Props["User"]
	username := sCtx.Refs[userRef]
	if username == "" {
		username = userRef // fallback if not a ref
	}

	name := e.Props["Name"]

	if err := sqlP.CreateRepo(ctx, username, nil, name); err != nil {
		return err
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
	p := gobookmarks.GetProvider("sql")
	sqlP, ok := p.(*gobookmarks.SQLProvider)
	if !ok {
		return fmt.Errorf("provider is not SQLProvider")
	}

	userRef := e.Props["User"]
	username := sCtx.Refs[userRef]
	if username == "" {
		username = userRef
	}

	branch := e.Props["Branch"]
	if branch == "" {
		branch = "main"
	}

	body := strings.TrimSpace(e.Body)

	// In the real application, UpdateBookmarks requires the original ref and sourceRef logic.
	// But CreateBookmarks acts like an upsert for sqlite.
	if err := sqlP.CreateBookmarks(ctx, username, nil, branch, body); err != nil {
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
