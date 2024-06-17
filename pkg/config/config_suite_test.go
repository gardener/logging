// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var testFileName string

var _ = AfterSuite(func() {
	os.Remove(testFileName)
})

func TestVali(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vali Config Suite")
}
