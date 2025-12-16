// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"crypto/tls"
	"time"
)

// ClientConfig holds configuration for the chain of clients.
type ClientConfig struct {
	SeedType  string `mapstructure:"SeedType"`  // e.g., "OTLPGRPC"
	ShootType string `mapstructure:"ShootType"` // e.g., "STDOUT"
}

// DqueConfig contains the dqueue settings
type DqueConfig struct {
	DqueDir         string `mapstructure:"DqueDir"`
	DqueSegmentSize int    `mapstructure:"DqueSegmentSize"`
	DqueSync        bool   `mapstructure:"-"` // Handled specially in postProcessConfig
	DqueName        string `mapstructure:"DqueName"`
}

// DefaultDqueConfig holds dque configurations for the buffer
var DefaultDqueConfig = DqueConfig{
	DqueDir:         "/tmp/flb-storage",
	DqueSegmentSize: 500,
	DqueSync:        false,
	DqueName:        "dque",
}

// OTLPConfig holds configuration for otlp endpoint
type OTLPConfig struct {
	Endpoint    string            `mapstructure:"Endpoint"`
	Insecure    bool              `mapstructure:"Insecure"`
	Compression int               `mapstructure:"Compression"`
	Timeout     time.Duration     `mapstructure:"Timeout"`
	Headers     map[string]string `mapstructure:"-"` // Handled manually in processOTLPConfig

	DqueConfig DqueConfig `mapstructure:",squash"`

	// Batch Processor configuration fields
	DqueBatchProcessorMaxQueueSize     int           `mapstructure:"DqueBatchProcessorMaxQueueSize"`
	DqueBatchProcessorMaxBatchSize     int           `mapstructure:"DqueBatchProcessorMaxBatchSize"`
	DqueBatchProcessorExportTimeout    time.Duration `mapstructure:"DqueBatchProcessorExportTimeout"`
	DqueBatchProcessorExportInterval   time.Duration `mapstructure:"DqueBatchProcessorExportInterval"`
	DqueBatchProcessorExportBufferSize int           `mapstructure:"DqueBatchProcessorExportBufferSize"`

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

	DqueConfig: DefaultDqueConfig, // Use default dque config

	// Batch Processor defaults - tuned to prevent OOM under high load
	DqueBatchProcessorMaxQueueSize:     512,              // Max records in queue before dropping
	DqueBatchProcessorMaxBatchSize:     256,              // Max records per export batch
	DqueBatchProcessorExportTimeout:    30 * time.Second, // Timeout for single export
	DqueBatchProcessorExportInterval:   1 * time.Second,  // Flush interval
	DqueBatchProcessorExportBufferSize: 10,
}
