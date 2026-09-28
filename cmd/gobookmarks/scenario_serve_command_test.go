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
		{"explicit all", "0.0.0.0:8081", "0.0.0.0:8081", false},
		{"explicit ipv6 loopback", "[::1]:8081", "[::1]:8081", false},
		{"invalid address", "invalid-address", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScenarioPort(tt.input)
			if (err != nil) != tt.hasErr {
				t.Fatalf("expected error: %v, got: %v", tt.hasErr, err)
			}
			if !tt.hasErr && got != tt.expected {
				t.Errorf("expected: %s, got: %s", tt.expected, got)
			}
		})
	}
}

func TestScenarioServeAddressParsing(t *testing.T) {
	tests := []struct {
		name         string
		portArg      string
		expectErr    bool
		expectedHost string
	}{
		{
			name:         "numeric_ephemeral_port",
			portArg:      "0",
			expectErr:    false,
			expectedHost: "127.0.0.1",
		},
		{
			name:         "colon_ephemeral_port",
			portArg:      ":0",
			expectErr:    false,
			expectedHost: "127.0.0.1",
		},
		{
			name:         "explicit_loopback_ephemeral",
			portArg:      "127.0.0.1:0",
			expectErr:    false,
			expectedHost: "127.0.0.1",
		},
		{
			name:         "explicit_non-loopback_ephemeral",
			portArg:      "0.0.0.0:0",
			expectErr:    false,
			expectedHost: "0.0.0.0", // we can bind to all interfaces
		},
		{
			name:         "explicit_ipv6_loopback",
			portArg:      "[::1]:0",
			expectErr:    false,
			expectedHost: "::1",
		},
		{
			name:      "invalid_address",
			portArg:   "invalid-address",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := NewRootCommand()

			ctx, cancel := context.WithCancel(context.Background())
			errCh := make(chan error, 1)

			var cleanedUp bool
			t.Cleanup(func() {
				if !cleanedUp {
					cancel()
					<-errCh
				}
			})

			readyPortCh := make(chan string, 1)

			sc := root.ScenarioCmd.ServeCommand
			sc.readyPort = readyPortCh

			args := []string{}
			if tt.portArg != "" {
				args = append(args, "--port", tt.portArg)
			}
			args = append(args, "scenarios/complex-bookmarks.txtar")

			go func() {
				errCh <- sc.ExecuteContext(ctx, args)
			}()

			var boundPort string
			var serveErr error

			select {
			case boundPort = <-readyPortCh:
			case serveErr = <-errCh:
				cleanedUp = true // since errCh produced a result, it exited
			case <-time.After(5 * time.Second):
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

			if boundPort == "" {
				t.Fatalf("Expected bound port to be reported")
			}

			host, _, err := net.SplitHostPort(boundPort)
			if err != nil {
				t.Fatalf("Failed to split reported bound port %q: %v", boundPort, err)
			}

			isValidHost := host == tt.expectedHost || (tt.expectedHost == "0.0.0.0" && host == "::")
			if !isValidHost {
				t.Errorf("Expected host %q, got %q", tt.expectedHost, host)
			}

			cancel()
			if !cleanedUp {
				<-errCh
				cleanedUp = true
			}
		})
	}
}
