// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

//nolint:revive // var-naming: package name "types" is acceptable for this types definition package
package types

import "strings"

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
	case "OTLPGRPC":
		return OTLPGRPC
	case "OTLPHTTP":
		return OTLPHTTP
	default:
		return NOOP
	}
}

// String returns the string representation of the client Type
func (t Type) String() string {
	switch t {
	case NOOP:
		return "NOOP"
	case STDOUT:
		return "STDOUT"
	case OTLPGRPC:
		return "OTLPGRPC"
	case OTLPHTTP:
		return "OTLPHTTP"
	case UNKNOWN:
		return "UNKNOWN"
	default:
		return ""
	}
}
