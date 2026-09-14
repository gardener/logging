// Copyright 2026 SPDX-FileCopyrightText: Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfigPopulation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Population Suite")
}
