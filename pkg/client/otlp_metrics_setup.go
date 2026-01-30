// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

// Package client provides OTLP client implementations with integrated metrics collection.
// The metrics setup uses a singleton pattern to ensure only one Prometheus exporter
// is created across all clients, preventing duplicate metric collection errors.
package client

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/otlptranslator"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc"
)

// MetricsSetup encapsulates the OpenTelemetry meter provider and manages its lifecycle.
// It ensures idempotent shutdown and provides helpers for gRPC instrumentation.
// This type is thread-safe and designed to be used as a singleton.
type MetricsSetup struct {
	provider     *sdkmetric.MeterProvider
	shutdownOnce sync.Once
}

var (
	// globalMetricsSetup is the singleton instance shared across all OTLP clients.
	// globalMetricsSetup is the singleton instance shared across all OTLP clients.
	// It is initialized during package initialization and reused for all subsequent requests.
	globalMetricsSetup *MetricsSetup

	// metricsSetupErr stores any initialization error that occurred during setup.
	metricsSetupErr error
)

func init() {
	if globalMetricsSetup, metricsSetupErr = initializeMetricsSetup(); metricsSetupErr != nil {
		slog.Error("failed to initialize global metrics setup", "error", metricsSetupErr)
	}
}

// initializeMetricsSetup creates and configures the metrics infrastructure.
// This function is called exactly once by the singleton pattern.
func initializeMetricsSetup() (*MetricsSetup, error) {
	// Create Prometheus exporter using the default registry
	// This ensures OTLP metrics are exposed on the same /metrics endpoint
	// as the existing Prometheus metrics (port 2021)
	promExporter, err := prometheus.New(
		prometheus.WithRegisterer(promclient.DefaultRegisterer),
		prometheus.WithNamespace("output_plugin"),
		prometheus.WithTranslationStrategy(otlptranslator.UnderscoreEscapingWithSuffixes),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize prometheus exporter for OTLP metrics: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExporter),
	)

	// Set as global meter provider so instrumentation libraries can discover it
	otel.SetMeterProvider(meterProvider)

	return &MetricsSetup{
		provider: meterProvider,
	}, nil
}

// GetProvider returns the configured OpenTelemetry meter provider.
// The provider is used for creating meters and recording metrics.
func (m *MetricsSetup) GetProvider() *sdkmetric.MeterProvider {
	return m.provider
}

// GetGRPCStatsHandler returns a gRPC dial option that enables automatic
// metrics collection for gRPC client calls.
//
// The handler collects standard gRPC metrics like request count, duration,
// and message sizes using the OpenTelemetry meter provider.
func (m *MetricsSetup) GetGRPCStatsHandler() grpc.DialOption {
	return grpc.WithStatsHandler(otelgrpc.NewClientHandler(
		otelgrpc.WithMeterProvider(m.provider),
	))
}

// Shutdown gracefully shuts down the meter provider and stops metrics collection.
//
// This method is idempotent - multiple calls are safe and will only perform
// the actual shutdown once. Subsequent calls return nil immediately.
//
// The context is used to enforce a timeout on the shutdown operation.
// If the context expires before shutdown completes, the context error is returned.
//
// After shutdown, the meter provider should not be used for new metric operations.
func (m *MetricsSetup) Shutdown(ctx context.Context) error {
	var shutdownErr error

	m.shutdownOnce.Do(func() {
		if err := m.provider.Shutdown(ctx); err != nil {
			shutdownErr = fmt.Errorf("failed to shutdown meter provider: %w", err)
		}
	})

	return shutdownErr
}
