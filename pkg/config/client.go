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

	// BufferConfig holds the configuration for the buffered client
	BufferConfig BufferConfig `mapstructure:",squash"`
}

// BufferConfig contains the buffer settings
type BufferConfig struct {
	Buffer     bool       `mapstructure:"Buffer"`
	DqueConfig DqueConfig `mapstructure:",squash"`
}

// DqueConfig contains the dqueue settings
type DqueConfig struct {
	QueueDir         string `mapstructure:"QueueDir"`
	QueueSegmentSize int    `mapstructure:"QueueSegmentSize"`
	QueueSync        bool   `mapstructure:"-"` // Handled specially in postProcessConfig
	QueueName        string `mapstructure:"QueueName"`
}

// DefaultBufferConfig holds the configurations for using output buffer
var DefaultBufferConfig = BufferConfig{
	Buffer:     false,
	DqueConfig: DefaultDqueConfig,
}

// DefaultDqueConfig holds dque configurations for the buffer
var DefaultDqueConfig = DqueConfig{
	QueueDir:         "/tmp/flb-storage/vali",
	QueueSegmentSize: 500,
	QueueSync:        false,
	QueueName:        "dque",
}

// OTLPConfig holds configuration for otlp endpoint
type OTLPConfig struct {
	Endpoint    string            `mapstructure:"Endpoint"`
	Insecure    bool              `mapstructure:"Insecure"`
	Compression int               `mapstructure:"Compression"`
	Timeout     time.Duration     `mapstructure:"Timeout"`
	Headers     map[string]string `mapstructure:"-"` // Handled manually in processOTLPConfig

	// Retry configuration fields
	RetryEnabled         bool          `mapstructure:"RetryEnabled"`
	RetryInitialInterval time.Duration `mapstructure:"RetryInitialInterval"`
	RetryMaxInterval     time.Duration `mapstructure:"RetryMaxInterval"`
	RetryMaxElapsedTime  time.Duration `mapstructure:"RetryMaxElapsedTime"`

	// RetryConfig - processed from the above fields
	RetryConfig *RetryConfig `mapstructure:"-"`

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
	Endpoint:              "localhost:4317",
	Insecure:              false,
	Compression:           0, // No compression by default
	Timeout:               30 * time.Second,
	Headers:               make(map[string]string),
	RetryEnabled:          true,
	RetryInitialInterval:  5 * time.Second,
	RetryMaxInterval:      30 * time.Second,
	RetryMaxElapsedTime:   time.Minute,
	RetryConfig:           nil, // Will be built from other fields
	TLSCertFile:           "",
	TLSKeyFile:            "",
	TLSCAFile:             "",
	TLSServerName:         "",
	TLSInsecureSkipVerify: false,
	TLSMinVersion:         "1.2", // TLS 1.2 as default minimum
	TLSMaxVersion:         "",    // Use Go's default maximum
	TLSConfig:             nil,   // Will be built from other fields
}
