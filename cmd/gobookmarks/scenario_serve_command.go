package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type ScenarioServeCommand struct {
	parent Command
	Flags  *flag.FlagSet
	Port   stringFlag
}

func (sc *ScenarioCommand) NewScenarioServeCommand() (*ScenarioServeCommand, error) {
	c := &ScenarioServeCommand{
		parent: sc,
		Flags:  flag.NewFlagSet("serve", flag.ContinueOnError),
	}
	c.Flags.Var(&c.Port, "port", "Port to serve on (default: 8080)")
	return c, nil
}

func (c *ScenarioServeCommand) Name() string {
	return c.Flags.Name()
}

func (c *ScenarioServeCommand) Parent() Command {
	return c.parent
}

func (c *ScenarioServeCommand) FlagSet() *flag.FlagSet {
	return c.Flags
}

func (c *ScenarioServeCommand) Subcommands() []Command {
	return nil
}

func (c *ScenarioServeCommand) Execute(args []string) error {
	c.FlagSet().Usage = func() { printHelp(c, nil) }
	if err := c.FlagSet().Parse(args); err != nil {
		printHelp(c, err)
		return err
	}

	remaining := c.FlagSet().Args()
	if len(remaining) == 0 {
		return fmt.Errorf("missing scenario PATH")
	}

	path := remaining[0]
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open scenario: %w", err)
	}
	defer func() { _ = file.Close() }()

	scenario, err := ParseScenario(file)
	if err != nil {
		return fmt.Errorf("failed to parse scenario: %w", err)
	}

	if err := ValidateScenario(scenario); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	cleanup, err := setupScenarioBackend()
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Create mock transport using same methodology as regression tests
	mockTransport := &mockOAuthRoundTripper{
		fakeToken: "fake-scenario-token",
		userLogin: "charlie", // Fallback, would ideally be extracted dynamically
	}
	mockClient := &http.Client{Transport: mockTransport}

	// Apply scenario with isolated OAuth client context
	applyCtx := context.WithValue(context.Background(), oauth2.HTTPClient, mockClient)
	if err := ApplyScenario(applyCtx, scenario); err != nil {
		return fmt.Errorf("failed to apply scenario: %w", err)
	}

	r := newApplicationRouter()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Use the scenario's provider isolation via context
		// This uses golang.org/x/oauth2 HTTPClient key
		ctx := context.WithValue(req.Context(), oauth2.HTTPClient, mockClient)
		r.ServeHTTP(w, req.WithContext(ctx))
	})

	port := ":8080"
	if c.Port.set {
		port = c.Port.value
		if port != "" && port[0] != ':' {
			port = ":" + port
		}
	}

	httpServer := &http.Server{
		Addr:    port,
		Handler: testHandler,
	}

	var sigCh chan os.Signal
	go func() {
		sigCh = make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh

		timeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server error during shutdown: %v", err)
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Printf("Scenario serve HTTP server listening on %s...\n", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	wg.Wait()
	return nil
}
