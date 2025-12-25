// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"crypto/tls"
	"time"
)

// DQueConfig contains the dqueue settings
type DQueConfig struct {
	DQueDir         string `mapstructure:"DQueDir"`
	DQueSegmentSize int    `mapstructure:"DQueSegmentSize"`
	DQueSync        bool   `mapstructure:"-"` // Handled specially in postProcessConfig
	DQueName        string `mapstructure:"DQueName"`
}

// DefaultDQueConfig holds dque configurations for the buffer
var DefaultDQueConfig = DQueConfig{
	DQueDir:         "/tmp/flb-storage",
	DQueSegmentSize: 500,
	DQueSync:        false,
	DQueName:        "dque",
}

// OTLPConfig holds configuration for otlp endpoint
type OTLPConfig struct {
	Endpoint        string            `mapstructure:"Endpoint"`
	EndpointURL     string            `mapstructure:"EndpointURL"`
	EndpointURLPath string            `mapstructure:"EndpointURLPath"`
	Insecure        bool              `mapstructure:"Insecure"`
	Compression     int               `mapstructure:"Compression"`
	Timeout         time.Duration     `mapstructure:"Timeout"`
	Headers         map[string]string `mapstructure:"-"` // Handled manually in processOTLPConfig

	DQueConfig DQueConfig `mapstructure:",squash"`

	// Batch Processor configuration fields
	DQueBatchProcessorMaxQueueSize     int           `mapstructure:"DQueBatchProcessorMaxQueueSize"`
	DQueBatchProcessorMaxBatchSize     int           `mapstructure:"DQueBatchProcessorMaxBatchSize"`
	DQueBatchProcessorExportTimeout    time.Duration `mapstructure:"DQueBatchProcessorExportTimeout"`
	DQueBatchProcessorExportInterval   time.Duration `mapstructure:"DQueBatchProcessorExportInterval"`
	DQueBatchProcessorExportBufferSize int           `mapstructure:"DQueBatchProcessorExportBufferSize"`

	// Retry configuration fields
	RetryEnabled         bool          `mapstructure:"RetryEnabled"`
	RetryInitialInterval time.Duration `mapstructure:"RetryInitialInterval"`
	RetryMaxInterval     time.Duration `mapstructure:"RetryMaxInterval"`
	RetryMaxElapsedTime  time.Duration `mapstructure:"RetryMaxElapsedTime"`

	// RetryConfig - processed from the above fields
	RetryConfig *RetryConfig `mapstructure:"-"`

	// Throttle configuration fields
	ThrottleEnabled        bool `mapstructure:"ThrottleEnabled"`
	ThrottleRequestsPerSec int  `mapstructure:"ThrottleRequestsPerSec"` // Maximum requests per second, 0 means no limit

	// TLS configuration fields
	TLSCertFile           string `mapstructure:"TLSCertFile"`
	TLSKeyFile            string `mapstructure:"TLSKeyFile"`
	TLSCAFile             string `mapstructure:"TLSCAFile"`
	TLSServerName         string `mapstructure:"TLSServerName"`
	TLSInsecureSkipVerify bool   `mapstructure:"TLSInsecureSkipVerify"`
	TLSMinVersion         string `mapstructure:"TLSMinVersion"`
	TLSMaxVersion         string `mapstructure:"TLSMaxVersion"`

	// TLS configuration - processed from the above fields
	TLSConfig *tls.Config `mapstructure:"-"`
}

// DefaultOTLPConfig holds the default configuration for OTLP
var DefaultOTLPConfig = OTLPConfig{
	Endpoint:               "localhost:4317",
	EndpointURL: "",
	EndpointURLPath:        "/v1/logs",
	Insecure:               false,
	Compression:            0, // No compression by default
	Timeout:                30 * time.Second,
	Headers:                make(map[string]string),
	RetryEnabled:           true,
	RetryInitialInterval:   5 * time.Second,
	RetryMaxInterval:       30 * time.Second,
	RetryMaxElapsedTime:    1 * time.Minute,
	RetryConfig:            nil, // Will be built from other fields
	ThrottleEnabled:        false,
	ThrottleRequestsPerSec: 0, // No throttling by default
	TLSCertFile:            "",
	TLSKeyFile:             "",
	TLSCAFile:              "",
	TLSServerName:          "",
	TLSInsecureSkipVerify:  false,
	TLSMinVersion:          "1.2", // TLS 1.2 as default minimum
	TLSMaxVersion:          "",    // Use Go's default maximum
	TLSConfig:              nil,   // Will be built from other fields

	DQueConfig: DefaultDQueConfig, // Use default dque config

	// Batch Processor defaults - tuned to prevent OOM under high load
	DQueBatchProcessorMaxQueueSize:     512,              // Max records in queue before dropping
	DQueBatchProcessorMaxBatchSize:     256,              // Max records per export batch
	DQueBatchProcessorExportTimeout:    30 * time.Second, // Timeout for single export
	DQueBatchProcessorExportInterval:   1 * time.Second,  // Flush interval
	DQueBatchProcessorExportBufferSize: 10,
}
