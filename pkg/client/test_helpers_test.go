// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"os"
	"path/filepath"
)

// GetTestTempDir returns the temporary directory for this test run.
// It creates a subdirectory with the given name if provided.
// This helper is intended for use in tests only.
func GetTestTempDir(subdir ...string) string {
	baseDir := os.Getenv("TEST_FLB_STORAGE_DIR")
	if baseDir == "" {
		// Fallback to /tmp/flb-storage-test if env var not set
		baseDir = "/tmp/flb-storage-test"
	}
	if len(subdir) > 0 && subdir[0] != "" {
		return filepath.Join(baseDir, subdir[0])
	}

	return baseDir
}
