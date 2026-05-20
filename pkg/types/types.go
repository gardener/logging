// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

//nolint:revive // var-naming: package name "types" is acceptable for this types definition package
package types

import (
	"strings"
	"time"
)

// Type represents the type of OutputClient
type Type int

const (
	// Noop represents a no-operation client type
	Noop Type = iota
	// StdOut represents a standard output client type
	StdOut
	// OTLPGRPC represents an OTLP gRPC client type
	OTLPGRPC
	// OTLPHTTP represents an OTLP HTTP client type
	OTLPHTTP
	// Unknown represents an unknown client type
	Unknown
)

// GetClientTypeFromString converts a string representation of client type to Type. It returns Noop for unknown types.
func GetClientTypeFromString(clientType string) Type {
	switch strings.ToUpper(clientType) {
	case "NOOP":
		return Noop
	case "STDOUT":
		return StdOut
	case "OTLPGRPC", "OTLP_GRPC":
		return OTLPGRPC
	case "OTLPHTTP", "OTLP_HTTP":
		return OTLPHTTP
	default:
		return Noop
	}
}

// String returns the string representation of the client Type
func (t Type) String() string {
	switch t {
	case Noop:
		return "noop"
	case StdOut:
		return "stdout"
	case OTLPGRPC:
		return "otlp_grpc"
	case OTLPHTTP:
		return "otlp_http"
	case Unknown:
		return "unknown"
	default:
		return ""
	}
}

// OutputEntry represents a log record with a timestamp
type OutputEntry struct {
	Timestamp time.Time
	Record    map[string]any
}
