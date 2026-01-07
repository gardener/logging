// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"strconv"

	otlplog "go.opentelemetry.io/otel/log"
)

// mapSeverity maps log level from various common formats to OTLP severity
// Supports: level, severity, loglevel fields as string or numeric values
// Returns both the OTLP severity enum and the original severity text
func mapSeverity(record map[string]any) (otlplog.Severity, string) {
	// Try common field names for log level
	levelFields := []string{"level", "severity", "loglevel", "log_level", "lvl"}

	for _, field := range levelFields {
		if levelValue, ok := record[field]; ok {
			// Handle string levels
			if levelStr, ok := levelValue.(string); ok {
				return mapSeverityString(levelStr), levelStr
			}
			// Handle numeric levels (e.g., syslog severity)
			if levelNum, ok := levelValue.(int); ok {
				return mapSeverityNumeric(levelNum), strconv.Itoa(levelNum)
			}
			if levelNum, ok := levelValue.(float64); ok {
				return mapSeverityNumeric(int(levelNum)), strconv.Itoa(int(levelNum))
			}
		}
	}

	// Default to Info if no level found
	return otlplog.SeverityInfo, "Info"
}

// mapSeverityString maps string log levels to OTLP severity
func mapSeverityString(level string) otlplog.Severity {
	// Normalize to lowercase for case-insensitive matching
	//nolint:revive // identical-switch-branches: default fallback improves readability
	switch level {
	case "trace", "TRACE", "Trace":
		return otlplog.SeverityTrace
	case "debug", "DEBUG", "Debug", "dbg", "DBG":
		return otlplog.SeverityDebug
	case "info", "INFO", "Info", "information", "INFORMATION":
		return otlplog.SeverityInfo
	case "warn", "WARN", "Warn", "warning", "WARNING", "Warning":
		return otlplog.SeverityWarn
	case "error", "ERROR", "Error", "err", "ERR":
		return otlplog.SeverityError
	case "fatal", "FATAL", "Fatal", "critical", "CRITICAL", "Critical", "crit", "CRIT":
		return otlplog.SeverityFatal
	default:
		return otlplog.SeverityInfo
	}
}

// mapSeverityNumeric maps numeric log levels (e.g., syslog severity) to OTLP severity
// Uses syslog severity scale: 0=Emergency, 1=Alert, 2=Critical, 3=Error, 4=Warning, 5=Notice, 6=Info, 7=Debug
func mapSeverityNumeric(level int) otlplog.Severity {
	//nolint:revive // identical-switch-branches: default fallback improves readability
	switch level {
	case 0, 1: // Emergency, Alert
		return otlplog.SeverityFatal4
	case 2: // Critical
		return otlplog.SeverityFatal
	case 3: // Error
		return otlplog.SeverityError
	case 4: // Warning
		return otlplog.SeverityWarn
	case 5, 6: // Notice, Info
		return otlplog.SeverityInfo
	case 7: // Debug
		return otlplog.SeverityDebug
	default:
		return otlplog.SeverityInfo
	}
}
