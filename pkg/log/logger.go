// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package log

import (
	"log/slog"
	"os"
	"strings"

	"github.com/go-logr/logr"
)

// NewLogger creates a new logr.Logger with slog backend
func NewLogger(level string) logr.Logger {
	return NewLoggerWithHandler(level, os.Stderr)
}

// NewLoggerWithHandler creates a new logr.Logger with custom output
func NewLoggerWithHandler(level string, output *os.File) logr.Logger {
	slogLevel := parseSlogLevel(level)

	opts := &slog.HandlerOptions{
		Level:     slogLevel,
		AddSource: slogLevel == slog.LevelDebug,
	}

	var handler slog.Handler
	switch slogLevel {
	case slog.LevelDebug:
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}

	return logr.FromSlogHandler(handler)
}

// NewNopLogger creates a no-op logger for testing
func NewNopLogger() logr.Logger {
	return logr.Discard()
}

// parseSlogLevel converts a string log level to slog.Level
func parseSlogLevel(level string) slog.Level {
	//nolint:revive // identical-switch-branches: default fallback improves readability
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
