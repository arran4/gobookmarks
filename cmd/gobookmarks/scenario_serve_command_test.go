package main

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestParseScenarioPort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasErr   bool
	}{
		{"default empty", "", "127.0.0.1:8080", false},
		{"numeric port", "8081", "127.0.0.1:8081", false},
		{"colon port", ":8081", "127.0.0.1:8081", false},
		{"explicit loopback", "127.0.0.1:8081", "127.0.0.1:8081", false},
		{"explicit external", "0.0.0.0:8081", "0.0.0.0:8081", false},
		{"malformed address", "invalid::port", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScenarioPort(tt.input)
			if tt.hasErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if got != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, got)
				}
			}
		})
	}
}

func TestScenarioServeAddressParsing(t *testing.T) {
	tests := []struct {
		name       string
		portArg    string
		expectErr  bool
		expectHost string
	}{
		{"numeric ephemeral port", "0", false, "127.0.0.1"},
		{"colon ephemeral port", ":0", false, "127.0.0.1"},
		{"explicit loopback ephemeral", "127.0.0.1:0", false, "127.0.0.1"},
		{"explicit non-loopback ephemeral", "0.0.0.0:0", false, "0.0.0.0"},
		{"explicit ipv6 loopback", "[::1]:0", false, "::1"},
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
				if tt.expectHost == "0.0.0.0" {
					if host != "0.0.0.0" && host != "::" {
						t.Errorf("Expected external wildcard host (0.0.0.0 or ::), got %q (bound address: %s)", host, boundPort)
					}
				} else if host != tt.expectHost {
					t.Errorf("Expected host %q, got %q (bound address: %s)", tt.expectHost, host, boundPort)
				}
			}

			cancel()
			<-errCh
		})
	}
}
