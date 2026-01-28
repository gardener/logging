// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/go-logr/logr"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/gardener/logging/v1/pkg/config"
)

// BatchProcessorType defines the type of batch processor to use
type BatchProcessorType string

const (
	// BatchProcessorTypeDQue uses the custom DQueBatchProcessor with disk persistence
	BatchProcessorTypeDQue BatchProcessorType = "dque"
	// BatchProcessorTypeSDK uses the OTEL SDK BatchProcessor (in-memory only)
	BatchProcessorTypeSDK BatchProcessorType = "sdk"
)

// BatchProcessorFactory creates batch processors based on configuration
type BatchProcessorFactory struct {
	logger logr.Logger
}

// NewBatchProcessorFactory creates a new BatchProcessorFactory
func NewBatchProcessorFactory(logger logr.Logger) *BatchProcessorFactory {
	return &BatchProcessorFactory{
		logger: logger,
	}
}

// Create creates the appropriate batch processor based on the configuration
// The clientName is used to distinguish between different clients (e.g., "otlp-grpc", "otlp-http")
func (f *BatchProcessorFactory) Create(
	ctx context.Context,
	cfg config.Config,
	exporter sdklog.Exporter,
	clientName string,
) (sdklog.Processor, error) {
	if cfg.OTLPConfig.UseSDKBatchProcessor {
		return f.createSDKProcessor(cfg, exporter)
	}

	return f.createDQueProcessor(ctx, cfg, exporter, clientName)
}

// createSDKProcessor creates an OTEL SDK BatchProcessor
func (f *BatchProcessorFactory) createSDKProcessor(
	cfg config.Config,
	exporter sdklog.Exporter,
) (sdklog.Processor, error) {
	opts := []sdklog.BatchProcessorOption{}

	if cfg.OTLPConfig.SDKBatchMaxQueueSize > 0 {
		opts = append(opts, sdklog.WithMaxQueueSize(cfg.OTLPConfig.SDKBatchMaxQueueSize))
	}

	if cfg.OTLPConfig.SDKBatchExportTimeout > 0 {
		opts = append(opts, sdklog.WithExportTimeout(cfg.OTLPConfig.SDKBatchExportTimeout))
	}

	if cfg.OTLPConfig.SDKBatchExportInterval > 0 {
		opts = append(opts, sdklog.WithExportInterval(cfg.OTLPConfig.SDKBatchExportInterval))
	}

	if cfg.OTLPConfig.SDKBatchExportMaxBatchSize > 0 {
		opts = append(opts, sdklog.WithExportMaxBatchSize(cfg.OTLPConfig.SDKBatchExportMaxBatchSize))
	}

	f.logger.V(1).Info("creating SDK batch processor",
		"maxQueueSize", cfg.OTLPConfig.SDKBatchMaxQueueSize,
		"exportTimeout", cfg.OTLPConfig.SDKBatchExportTimeout,
		"exportInterval", cfg.OTLPConfig.SDKBatchExportInterval,
		"maxBatchSize", cfg.OTLPConfig.SDKBatchExportMaxBatchSize,
	)

	return sdklog.NewBatchProcessor(exporter, opts...), nil
}

// createDQueProcessor creates a DQueBatchProcessor with disk persistence
func (f *BatchProcessorFactory) createDQueProcessor(
	ctx context.Context,
	cfg config.Config,
	exporter sdklog.Exporter,
	clientName string,
) (sdklog.Processor, error) {
	dQueueDir := filepath.Join(
		cfg.OTLPConfig.DQueConfig.DQueDir,
		cfg.OTLPConfig.DQueConfig.DQueName,
	)

	processor, err := NewDQueBatchProcessor(
		ctx,
		exporter,
		f.logger,
		WithEndpoint(cfg.OTLPConfig.Endpoint),
		WithDQueueDir(dQueueDir),
		WithDQueueName(clientName),
		WithDQueueSegmentSize(cfg.OTLPConfig.DQueConfig.DQueSegmentSize),
		WithDQueueSync(cfg.OTLPConfig.DQueConfig.DQueSync),
		WithMaxQueueSize(cfg.OTLPConfig.DQueBatchProcessorMaxQueueSize),
		WithMaxBatchSize(cfg.OTLPConfig.DQueBatchProcessorMaxBatchSize),
		WithExportTimeout(cfg.OTLPConfig.DQueBatchProcessorExportTimeout),
		WithExportInterval(cfg.OTLPConfig.DQueBatchProcessorExportInterval),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DQue batch processor: %w", err)
	}

	f.logger.V(1).Info("creating DQue batch processor",
		"dqueueDir", dQueueDir,
		"clientName", clientName,
		"maxQueueSize", cfg.OTLPConfig.DQueBatchProcessorMaxQueueSize,
		"maxBatchSize", cfg.OTLPConfig.DQueBatchProcessorMaxBatchSize,
	)

	return processor, nil
}

// GetProcessorType returns the type of processor that would be created based on config
func GetProcessorType(cfg config.Config) BatchProcessorType {
	if cfg.OTLPConfig.UseSDKBatchProcessor {
		return BatchProcessorTypeSDK
	}

	return BatchProcessorTypeDQue
}
