// Copyright 2025 SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package plugin_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOutputPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Test")
}
