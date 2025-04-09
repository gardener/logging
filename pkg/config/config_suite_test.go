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

var testFileName string

var _ = ginkgov2.AfterSuite(func() {
	_ = os.Remove(testFileName)
})

func TestVali(t *testing.T) {
	gomega.RegisterFailHandler(ginkgov2.Fail)
	ginkgov2.RunSpecs(t, "Vali Config Suite")
}
