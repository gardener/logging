// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// FetcherLogEntry represents a JSON log entry from the fetcher pod
type FetcherLogEntry struct {
	Time  string `json:"time"`
	Level string `json:"level"`
	Msg   string `json:"msg"`
	Query string `json:"query"`
	Count string `json:"count"`
	Error string `json:"error,omitempty"`
}

func TestSystemdLogs(t *testing.T) {
	f1 := features.New("systemd/logs").
		WithLabel("type", "systemd-logs").
		Assess("kubelet and containerd logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			g := NewWithT(t)

			// Retrieve the kubeconfig path from context
			kubeconfigPath := cfg.KubeconfigFile()
			var kubeletCount, containerdCount int

			// Use Eventually to poll for positive counts with timeout
			g.Eventually(func(g Gomega) {
				// Fetch logs from the fetcher pod using kubectl logs
				logs, err := getLogsFromFetcherPod(ctx, t, namespace, kubeconfigPath)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get logs from fetcher pod")

				// Parse the JSON logs and extract query results
				kubelet, containerd, err := parseAndExtractCounts(logs)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to parse fetcher logs")

				t.Logf("Found kubelet-service count: %d, containerd-service count: %d", kubelet, containerd)

				// Expect both counts to be positive
				g.Expect(kubelet).To(BeNumerically(">", 0), "kubelet-service log count should be positive")
				g.Expect(containerd).To(BeNumerically(">", 0), "containerd-service log count should be positive")

				// Store the counts for final logging
				kubeletCount = kubelet
				containerdCount = containerd
			}).WithTimeout(2 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

			t.Logf("Successfully verified systemd logs: kubelet-service=%d, containerd-service=%d", kubeletCount, containerdCount)

			return ctx
		}).Feature()

	// test feature
	testenv.Test(t, f1)
}

// getLogsFromFetcherPod retrieves logs from the log-fetcher pod using kubectl
func getLogsFromFetcherPod(ctx context.Context, t *testing.T, namespace, kubeconfigPath string) (string, error) {
	// Get the pod name first
	getPodCmd := fmt.Sprintf("kubectl --kubeconfig=%s get pods -n %s -l app=log-fetcher -o jsonpath='{.items[0].metadata.name}'", kubeconfigPath, namespace)
	podNameBytes, err := execCommand(ctx, "sh", "-c", getPodCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get fetcher pod name: %w", err)
	}

	podName := strings.Trim(string(podNameBytes), "'")
	if podName == "" {
		return "", errors.New("log-fetcher pod not found")
	}

	// Get logs from the pod (last 100 lines to avoid too much data)
	getLogsCmd := fmt.Sprintf("kubectl --kubeconfig=%s logs -n %s %s --tail=5", kubeconfigPath, namespace, podName)
	logs, err := execCommand(ctx, "sh", "-c", getLogsCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get logs from pod %s: %w", podName, err)
	}

	return string(logs), nil
}

// parseAndExtractCounts parses JSON logs and extracts the most recent counts for kubelet and containerd
func parseAndExtractCounts(logs string) (kubeletCount, containerdCount int, err error) {
	scanner := bufio.NewScanner(strings.NewReader(logs))

	kubeletCount = -1
	containerdCount = -1

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry FetcherLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip lines that aren't valid JSON
			continue
		}

		// Only process successful query results
		if entry.Msg != "result" || entry.Count == "" {
			continue
		}

		// Parse the count value
		count, err := strconv.Atoi(entry.Count)
		if err != nil {
			// Skip if count is not a valid integer
			continue
		}

		// Update the counts based on query name
		switch entry.Query {
		case "kubelet-service":
			kubeletCount = count
		case "containerd-service":
			containerdCount = count
		default:
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("error reading logs: %w", err)
	}

	if kubeletCount == -1 || containerdCount == -1 {
		return 0, 0, errors.New("could not find both kubelet and containerd counts in logs")
	}

	return kubeletCount, containerdCount, nil
}

// execCommand executes a command and returns its output
func execCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}
