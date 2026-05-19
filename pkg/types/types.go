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
	// NoopType type represents a no-operation client type
	NoopType Type = iota
	// StdOutType type represents a standard output client type
	StdOutType
	// OTLPGRPCType type represents an OTLP gRPC client type
	OTLPGRPCType
	// OTLPHTTPType type represents an OTLP HTTP client type
	OTLPHTTPType
	// UnknownType type represents an unknown client type
	UnknownType
)

// GetClientTypeFromString converts a string representation of client type to Type. It returns NOOP for unknown types.
func GetClientTypeFromString(clientType string) Type {
	switch strings.ToUpper(clientType) {
	case "NOOP":
		return NoopType
	case "STDOUT":
		return StdOutType
	case "OTLPGRPC", "OTLP_GRPC":
		return OTLPGRPCType
	case "OTLPHTTP", "OTLP_HTTP":
		return OTLPHTTPType
	default:
		return NoopType
	}
}

// String returns the string representation of the client Type
func (t Type) String() string {
	switch t {
	case NoopType:
		return "noop"
	case StdOutType:
		return "stdout"
	case OTLPGRPCType:
		return "otlp_grpc"
	case OTLPHTTPType:
		return "otlp_http"
	case UnknownType:
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

// OutputClient represents an instance which sends logs to Vali ingester
type OutputClient interface {
	// Handle processes logs and then sends them to Vali ingester
	Handle(log OutputEntry) error
	// Stop shut down the client immediately without waiting to send the saved logs
	Stop()
	// StopWait stops the client of receiving new logs and waits all saved logs to be sent until shutting down
	StopWait()
	// GetEndpoint returns the target logging backend endpoint
	GetEndpoint() string
}
