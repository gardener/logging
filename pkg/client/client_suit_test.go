// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"os"
	"testing"

	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var testTempDir string

var _ = ginkgov2.BeforeSuite(func() {
	var err error
	// Create a temporary directory for this test run with a descriptive suffix
	testTempDir, err = os.MkdirTemp("", "flb-storage-test-*")
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Set environment variable so tests can use this temp directory
	err = os.Setenv("TEST_FLB_STORAGE_DIR", testTempDir)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
})

var _ = ginkgov2.AfterSuite(func() {
	// Clean up the temporary directory after all tests complete
	if testTempDir != "" {
		_ = os.RemoveAll(testTempDir)
	}
})

func TestVali(t *testing.T) {
	gomega.RegisterFailHandler(ginkgov2.Fail)
	ginkgov2.RunSpecs(t, "Output Client Suite")
}
