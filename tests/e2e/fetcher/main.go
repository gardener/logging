// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultVictoriaLogsAddr = "http://victoria-logs:9428"
	defaultInterval         = 30 * time.Second
)

// Query represents a victoria-logs query configuration
type Query struct {
	Name  string
	Query string
}

// QueryResult represents the result from victoria-logs
type QueryResult struct {
	Count string `json:"count(*)"`
}

var queries = []Query{
	{
		Name:  "kubelet-service",
		Query: `_time:24h unit:"kubelet.service" | count()`,
	},
	{
		Name:  "containerd-service",
		Query: `_time:24h unit:"containerd.service" | count()`,
	},
	{
		Name:  "logger-container",
		Query: `_time:24h k8s.container.name:"logger" | count()`,
	},
	{
		Name:  "event-logger-container",
		Query: `_time:24h k8s.container.name:"event-logger" | count()`,
	},
}

func main() {
	victoriaLogsAddr := os.Getenv("VLOGS_ADDR")
	if victoriaLogsAddr == "" {
		victoriaLogsAddr = defaultVictoriaLogsAddr
	}

	intervalStr := os.Getenv("INTERVAL")
	interval := defaultInterval
	if intervalStr != "" {
		if d, err := time.ParseDuration(intervalStr); err == nil {
			interval = d
		}
	}

	fmt.Printf("Starting Victoria Logs fetcher...\n")
	fmt.Printf("Endpoint: %s\n", victoriaLogsAddr)
	fmt.Printf("Query interval: %v\n\n", interval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down fetcher...")
		cancel()
	}()

	// Create HTTP client
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on start
	fetchAllQueries(ctx, client, victoriaLogsAddr)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fetchAllQueries(ctx, client, victoriaLogsAddr)
		}
	}
}

func fetchAllQueries(ctx context.Context, client *http.Client, victoriaLogsAddr string) {
	fmt.Println("==========================================")
	fmt.Printf("Victoria Logs Query Results - %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Println("==========================================")

	for _, q := range queries {
		count, err := queryVictoriaLogs(ctx, client, victoriaLogsAddr, q.Query)
		if err != nil {
			fmt.Printf("%-25s | ERROR: %v\n", q.Name, err)
			continue
		}
		fmt.Printf("%-25s | count: %s\n", q.Name, count)
	}

	fmt.Println()
}

func queryVictoriaLogs(ctx context.Context, client *http.Client, victoriaLogsAddr, query string) (string, error) {
	// Build query URL
	queryURL := fmt.Sprintf("%s/select/logsql/query", victoriaLogsAddr)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to query victoria-logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response - victoria-logs returns NDJSON, we take the first line
	var result QueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		// If JSON parsing fails, return raw body (might be NDJSON or error message)
		return string(body), nil
	}

	return result.Count, nil
}
