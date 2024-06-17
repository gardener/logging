// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var testDir string

var _ = BeforeSuite(func() {
	var err error
	testDir, err = ioutil.TempDir("/tmp", "curator-config")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	var err error
	os.RemoveAll(testDir)
	Expect(err).ToNot(HaveOccurred())
})

func TestVali(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Curator Config Suite")
}
