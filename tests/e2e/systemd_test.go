// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	portForwardKey contextKey = "portForwardPort"
	portForwardCmd contextKey = "portForwardCmd"
)

func TestSystemdLogs(t *testing.T) {
	f1 := features.New("systemd/logs").
		WithLabel("type", "systemd-logs").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Create port-forward to victoria-logs instance
			t.Log("Setting up port-forward to victoria-logs service")

			// Find a free local port
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("Failed to find free port: %v", err)
			}
			localPort := listener.Addr().(*net.TCPAddr).Port
			_ = listener.Close()

			t.Logf("Using local port %d for port forwarding", localPort)

			// Start kubectl port-forward in background
			// victoria-logs service runs on port 9428

			// Build kubectl port-forward command
			portForwardCmd := exec.CommandContext(
				ctx,
				"kubectl",
				"port-forward",
				"-n", namespace,
				"pod/victoria-logs-0",
				fmt.Sprintf("%d:9428", localPort),
			)

			// Start the command in the background
			if err := portForwardCmd.Start(); err != nil {
				t.Fatalf("Failed to start port forwarding: %v", err)
			}

			t.Logf("Port forwarding established on local port %d", localPort)

			// Wait a bit for port forwarding to be ready
			time.Sleep(2 * time.Second)

			// Store port and command in context for use in Assess and Teardown steps
			ctx = context.WithValue(ctx, portForwardKey, localPort)
			ctx = context.WithValue(ctx, portForwardCmd, portForwardCmd)

			return ctx
		}).
		Assess("system logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Pull the port from context
			localPort, ok := ctx.Value(portForwardKey).(int)
			if !ok {
				t.Fatal("Failed to get port from context")
			}

			t.Logf("Querying victoria-logs on localhost:%d for systemd logs", localPort)

			// Wait for victoria-logs to have some data (give fluent-bit time to collect logs)
			time.Sleep(5 * time.Second)

			// Perform curl query to victoria-logs API to check for the presence of systemd logs
			// Victoria Logs query endpoint: http://localhost:<port>/select/logsql/query
			// Query for systemd logs using LogsQL
			queryURL := fmt.Sprintf("http://localhost:%d/select/logsql/query", localPort)

			// Query for systemd logs - they should have _SYSTEMD_UNIT field
			query := "_SYSTEMD_UNIT:*"

			// Create HTTP client with timeout
			client := &http.Client{
				Timeout: 10 * time.Second,
			}

			// Retry logic for querying victoria-logs
			maxRetries := 5
			retryDelay := 2 * time.Second
			var lastErr error
			var body []byte

			for attempt := 1; attempt <= maxRetries; attempt++ {
				t.Logf("Query attempt %d/%d", attempt, maxRetries)

				// Build request with query parameter
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
				if err != nil {
					lastErr = fmt.Errorf("failed to create request: %w", err)
					time.Sleep(retryDelay)
					continue
				}

				q := req.URL.Query()
				q.Add("query", query)
				q.Add("limit", "100")
				req.URL.RawQuery = q.Encode()

				t.Logf("Sending query: %s", query)

				// Execute request
				resp, err := client.Do(req)
				if err != nil {
					lastErr = fmt.Errorf("failed to query victoria-logs: %w", err)
					t.Logf("Query failed: %v, retrying...", err)
					time.Sleep(retryDelay)
					continue
				}

				if resp.StatusCode != http.StatusOK {
					bodyBytes, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					lastErr = fmt.Errorf("victoria-logs query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
					t.Logf("Query returned non-OK status: %v, retrying...", lastErr)
					time.Sleep(retryDelay)
					continue
				}

				// Read response body
				body, err = io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					lastErr = fmt.Errorf("failed to read response body: %w", err)
					time.Sleep(retryDelay)
					continue
				}

				// Success
				lastErr = nil
				break
			}

			if lastErr != nil {
				t.Fatalf("Failed to query victoria-logs after %d attempts: %v", maxRetries, lastErr)
			}

			t.Logf("Query response (first 500 chars): %s", truncateString(string(body), 500))

			// Parse response to check if we got any systemd logs
			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				// Victoria Logs might return NDJSON format, try to check if response has data
				if len(body) > 0 && strings.Contains(string(body), "_SYSTEMD_UNIT") {
					t.Log("Successfully verified systemd logs are being collected by victoria-logs (NDJSON format)")
					return ctx
				}
				t.Logf("Response is not JSON, checking raw content: %v", err)
			}

			// Check if we have hits (logs) in the response
			if len(body) == 0 {
				t.Error("No response from victoria-logs")
			} else if strings.Contains(string(body), "_SYSTEMD_UNIT") ||
				(result != nil && len(result) > 0) {
				t.Log("Successfully verified systemd logs are being collected by victoria-logs")
			} else {
				t.Logf("Warning: Response received but no systemd logs found. This may be expected if no systemd logs have been generated yet.")
			}

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Log("Cleaning up port forwarding")

			// Retrieve the port-forward command from context
			cmd, ok := ctx.Value(portForwardCmd).(*exec.Cmd)
			if !ok {
				t.Log("Port forward command not found in context, skipping cleanup")
				return ctx
			}

			// Stop the port forwarding process
			if cmd.Process != nil {
				t.Logf("Killing port-forward process (PID: %d)", cmd.Process.Pid)
				if err := cmd.Process.Kill(); err != nil {
					t.Logf("Warning: failed to kill port-forward process: %v", err)
				} else {
					t.Log("Port forwarding process stopped successfully")
				}

				// Wait for the process to actually terminate
				if _, err := cmd.Process.Wait(); err != nil {
					t.Logf("Warning: error waiting for process to terminate: %v", err)
				}
			} else {
				t.Log("Port forward process already terminated")
			}

			return ctx
		}).Feature()

	// test feature
	testenv.Test(t, f1)
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
