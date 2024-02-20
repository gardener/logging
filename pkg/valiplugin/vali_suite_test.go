// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package valiplugin_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVali(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ValiPlugin Suite")
}
