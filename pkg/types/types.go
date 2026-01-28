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
	// NOOP type represents a no-operation client type
	NOOP Type = iota
	// STDOUT type represents a standard output client type
	STDOUT
	// OTLPGRPC type represents an OTLP gRPC client type
	OTLPGRPC
	// OTLPHTTP type represents an OTLP HTTP client type
	OTLPHTTP
	// UNKNOWN type represents an unknown client type
	UNKNOWN
)

// GetClientTypeFromString converts a string representation of client type to Type. It returns NOOP for unknown types.
func GetClientTypeFromString(clientType string) Type {
	switch strings.ToUpper(clientType) {
	case "NOOP":
		return NOOP
	case "STDOUT":
		return STDOUT
	case "OTLPGRPC", "OTLP_GRPC":
		return OTLPGRPC
	case "OTLPHTTP", "OTLP_HTTP":
		return OTLPHTTP
	default:
		return NOOP
	}
}

// String returns the string representation of the client Type
func (t Type) String() string {
	switch t {
	case NOOP:
		return "noop"
	case STDOUT:
		return "stdout"
	case OTLPGRPC:
		return "otlp_grpc"
	case OTLPHTTP:
		return "otlp_http"
	case UNKNOWN:
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
