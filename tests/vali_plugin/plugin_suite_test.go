// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package vali_plugin_test

import (
	"testing"

	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestOutputPlugin(t *testing.T) {
	gomega.RegisterFailHandler(ginkgov2.Fail)
	ginkgov2.RunSpecs(t, "Plugin Test")
}
