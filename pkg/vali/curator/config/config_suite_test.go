// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"testing"

	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var testDir string

var _ = ginkgov2.BeforeSuite(func() {
	var err error

	testDir, err = os.MkdirTemp("/tmp", "curator-config")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
})

var _ = ginkgov2.AfterSuite(func() {
	var err error

	_ = os.RemoveAll(testDir)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
})

func TestVali(t *testing.T) {
	gomega.RegisterFailHandler(ginkgov2.Fail)
	ginkgov2.RunSpecs(t, "Curator Config Suite")
}
