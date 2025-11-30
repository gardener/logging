// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc"
)

// MetricsSetup encapsulates metrics provider creation and configuration
type MetricsSetup struct {
	provider *sdkmetric.MeterProvider
}

// NewMetricsSetup creates and configures a meter provider with Prometheus exporter
func NewMetricsSetup() (*MetricsSetup, error) {
	promExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExporter),
	)

	// Set as global meter provider for instrumentation libraries
	otel.SetMeterProvider(meterProvider)

	return &MetricsSetup{provider: meterProvider}, nil
}

// GetProvider returns the configured meter provider
func (m *MetricsSetup) GetProvider() *sdkmetric.MeterProvider {
	return m.provider
}

// GetGRPCStatsHandler returns a gRPC stats handler for automatic metrics collection
func (m *MetricsSetup) GetGRPCStatsHandler() grpc.DialOption {
	return grpc.WithStatsHandler(otelgrpc.NewClientHandler(
		otelgrpc.WithMeterProvider(m.provider),
	))
}
