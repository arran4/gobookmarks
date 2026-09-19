package main

import (
	"context"
	"fmt"
	"strings"

	gobookmarks "github.com/arran4/gobookmarks"
)

type ScenarioContext struct {
	Refs            map[string]string
	Files           map[string]string
	StorageProvider string
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

func getScenarioProvider(sCtx *ScenarioContext, e *Event) (gobookmarks.Provider, error) {
	providerName := sCtx.StorageProvider
	if eventProvider := e.Props["Provider"]; eventProvider != "" && eventProvider != providerName {
		return nil, fmt.Errorf("event provider %q conflicts with scenario StorageProvider %q", eventProvider, providerName)
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
	p, err := getScenarioProvider(sCtx, e)
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
	p, err := getScenarioProvider(sCtx, e)
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
	_, err := getScenarioProvider(sCtx, e)
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

	// Use the normal mutation access layer so scenario seeding observes the
	// same cache invalidation rules as application mutations.
	ctx = context.WithValue(ctx, gobookmarks.ContextValues("provider"), sCtx.StorageProvider)
	if err := gobookmarks.CreateBookmarks(ctx, username, nil, branch, body); err != nil {
		return err
	}

	return nil
}

func ValidateScenario(s *Scenario) error {
	if gobookmarks.GetProvider(s.StorageProvider) == nil {
		return fmt.Errorf("unsupported StorageProvider: %s", s.StorageProvider)
	}
	if s.AuthProvider != "" && gobookmarks.GetProvider(s.AuthProvider) == nil {
		return fmt.Errorf("unsupported AuthProvider: %s", s.AuthProvider)
	}
	refs := make(map[string]bool)
	for _, e := range s.Events {
		op, ok := operations[e.Op]
		if !ok {
			return fmt.Errorf("unknown operation: %s", e.Op)
		}
		if err := op.Validate(e); err != nil {
			return fmt.Errorf("event %s: %w", e.Name, err)
		}
		if eventProvider := e.Props["Provider"]; eventProvider != "" && eventProvider != s.StorageProvider {
			return fmt.Errorf("event %s: Provider %q conflicts with StorageProvider %q", e.Name, eventProvider, s.StorageProvider)
		}
		if e.Op == "repo.create" || e.Op == "bookmark.create" {
			if !refs[e.Props["User"]] {
				return fmt.Errorf("event %s: unresolved user ref: %s", e.Name, e.Props["User"])
			}
		}
		if e.Ref != "" {
			if refs[e.Ref] {
				return fmt.Errorf("event %s: duplicate ref: %s", e.Name, e.Ref)
			}
			refs[e.Ref] = true
		}

		// Ensure asset exists in context files during validation if provided
		if e.Props["Asset"] != "" {
			if _, ok := s.Files[e.Props["Asset"]]; !ok {
				return fmt.Errorf("event %s: missing asset: %s", e.Name, e.Props["Asset"])
			}
		}
	}
	return nil
}

func ApplyScenario(ctx context.Context, s *Scenario) error {
	sCtx := &ScenarioContext{
		Refs:            make(map[string]string),
		Files:           s.Files,
		StorageProvider: s.StorageProvider,
	}

	for _, e := range s.Events {
		op := operations[e.Op]
		if err := op.Apply(ctx, sCtx, e); err != nil {
			return fmt.Errorf("event %s: %w", e.Name, err)
		}
	}
	return nil
}
