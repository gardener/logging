// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package e2e_test

import (
	"flag"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/gardener/test/framework"
)

func TestMain(m *testing.M) {
	framework.RegisterGardenerFrameworkFlags()
	flag.Parse()
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test E2E Gardener-Logging Suite")
}
