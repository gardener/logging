// Copyright 2025 SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
// SPDX-License-Identifier: Apache-2.0

package controller_test

import (
	"testing"

	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestVali(t *testing.T) {
	gomega.RegisterFailHandler(ginkgov2.Fail)
	ginkgov2.RunSpecs(t, "Controller Suite")
}
