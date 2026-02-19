// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricsSetup Singleton", func() {
	It("should be initialized during package init", func() {
		Expect(metricsSetupErr).ToNot(HaveOccurred())
		Expect(globalMetricsSetup).ToNot(BeNil())
	})

	It("should return the same meter provider instance", func() {
		provider1 := globalMetricsSetup.GetProvider()
		provider2 := globalMetricsSetup.GetProvider()

		// Should be the exact same provider instance
		Expect(provider1).To(BeIdenticalTo(provider2))
	})

	It("should shutdown idempotently - multiple shutdowns should not error", func() {
		Expect(metricsSetupErr).ToNot(HaveOccurred())
		Expect(globalMetricsSetup).ToNot(BeNil())

		ctx := context.Background()

		// First shutdown
		err := globalMetricsSetup.Shutdown(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Second shutdown should not error (idempotent)
		err = globalMetricsSetup.Shutdown(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Third shutdown should also not error
		err = globalMetricsSetup.Shutdown(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should handle concurrent shutdown calls", func() {
		Expect(metricsSetupErr).ToNot(HaveOccurred())

		const goroutines = 10
		errors := make([]error, goroutines)
		done := make(chan bool)

		ctx := context.Background()

		// Multiple goroutines try to shutdown simultaneously
		for i := range goroutines {
			go func(index int) {
				errors[index] = globalMetricsSetup.Shutdown(ctx)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for range goroutines {
			<-done
		}

		// None should error (idempotent shutdown)
		for i := range goroutines {
			Expect(errors[i]).ToNot(HaveOccurred())
		}
	})
})
