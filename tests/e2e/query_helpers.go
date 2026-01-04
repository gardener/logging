// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// queryCurl executes a curl command in the log-fetcher deployment to query victoria-logs
func queryCurl(ctx context.Context, cfg *envconf.Config, namespace, query string) (string, error) {
	// Get a pod from the deployment
	var podList corev1.PodList
	if err := cfg.Client().Resources(namespace).List(ctx, &podList); err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	// Filter pods by label
	var pod *corev1.Pod
	for i := range podList.Items {
		if podList.Items[i].Labels["app"] == "log-fetcher" {
			pod = &podList.Items[i]

			break
		}
	}

	if pod == nil {
		return "", fmt.Errorf("no pods found for log-fetcher deployment")
	}

	// Execute curl command in the pod
	var stdout, stderr bytes.Buffer
	command := []string{
		"curl", "-s",
		"http://victoria-logs-0.victoria-logs.fluent-bit.svc.cluster.local:9428/select/logsql/query",
		"--data-urlencode", fmt.Sprintf("query=%s", query),
	}

	if err := cfg.Client().Resources().ExecInPod(ctx, namespace, pod.Name, "curl", command, &stdout, &stderr); err != nil {
		return "", fmt.Errorf("failed to exec in pod: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// parseQueryResponse parses the victoria-logs query response and extracts the count
func parseQueryResponse(response string) (int, error) {
	if response == "" {
		return 0, fmt.Errorf("empty response from victoria-logs")
	}

	// Parse NDJSON response - each line is a separate JSON object
	lines := strings.Split(strings.TrimSpace(response), "\n")
	if len(lines) == 0 {
		return 0, fmt.Errorf("no data in response")
	}

	// Try to parse the first line as JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &result); err != nil {
		return 0, fmt.Errorf("failed to parse response as JSON: %w", err)
	}

	// Look for count field - victoria-logs returns count() as "count(*)"
	if countVal, ok := result["count(*)"]; ok {
		switch v := countVal.(type) {
		case string:
			count, err := strconv.Atoi(v)
			if err != nil {
				return 0, fmt.Errorf("failed to convert count to int: %w", err)
			}

			return count, nil
		case float64:
			return int(v), nil
		case int:
			return v, nil
		}
	}

	return 0, fmt.Errorf("count field not found in response")
}
