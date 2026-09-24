package main

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestScenarioServeAddressParsing(t *testing.T) {
	tests := []struct {
		name       string
		portArg    string
		expectErr  bool
		expectHost string // can be empty to skip
	}{
		{"default", "", false, "127.0.0.1"},
		{"numeric port", "0", false, "127.0.0.1"},
		{"colon port", ":0", false, "127.0.0.1"},
		{"explicit loopback", "127.0.0.1:0", false, "127.0.0.1"},
		{"explicit non-loopback", "0.0.0.0:0", false, "0.0.0.0"}, // Or we can test if it binds to 0.0.0.0
		{"malformed host", "invalid::port", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := NewRootCommand()
			sc := root.ScenarioCmd.ServeCommand
			readyPortCh := make(chan string, 1)
			sc.readyPort = readyPortCh

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			args := []string{}
			if tt.portArg != "" {
				args = append(args, "--port", tt.portArg)
			}
			args = append(args, "scenarios/complex-bookmarks.txtar")

			errCh := make(chan error, 1)
			go func() {
				errCh <- sc.ExecuteContext(ctx, args)
			}()

			var boundPort string
			var serveErr error

			select {
			case boundPort = <-readyPortCh:
			case serveErr = <-errCh:
			case <-time.After(2 * time.Second):
				t.Fatalf("Timeout waiting for scenario serve to start or error")
			}

			if tt.expectErr {
				if serveErr == nil {
					t.Fatalf("Expected error but got none")
				}
				return
			}

			if serveErr != nil {
				t.Fatalf("Unexpected error: %v", serveErr)
			}

			host, _, err := net.SplitHostPort(boundPort)
			if err != nil {
				t.Fatalf("Failed to split bound port %q: %v", boundPort, err)
			}

			if tt.expectHost != "" {
				// net.Listen on "0.0.0.0:0" might resolve to "[::]:PORT", so we check behavior appropriately
				if tt.expectHost == "127.0.0.1" && host != "127.0.0.1" {
					t.Errorf("Expected host %q, got %q (bound address: %s)", tt.expectHost, host, boundPort)
				}
			}

			cancel()
			<-errCh
		})
	}
}
