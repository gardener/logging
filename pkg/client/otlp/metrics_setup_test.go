// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package otlp

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	promclient "github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("MetricsSetup Singleton", func() {
	var metricsSetup *MetricsSetup

	BeforeEach(func() {
		reg := promclient.NewRegistry()
		var err error
		metricsSetup, err = RegisterMetricsSetup(reg)
		Expect(err).ToNot(HaveOccurred())
		Expect(metricsSetup).ToNot(BeNil())
	})

	It("should be initialized successfully", func() {
		Expect(metricsSetup).ToNot(BeNil())
	})

	It("should return the same meter provider instance", func() {
		provider1 := metricsSetup.Provider()
		provider2 := metricsSetup.Provider()

		// Should be the exact same provider instance
		Expect(provider1).To(BeIdenticalTo(provider2))
	})

	It("should shutdown idempotently - multiple shutdowns should not error", func() {
		ctx := context.Background()

		// First shutdown
		err := metricsSetup.Shutdown(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Second shutdown should not error (idempotent)
		err = metricsSetup.Shutdown(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Third shutdown should also not error
		err = metricsSetup.Shutdown(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should handle concurrent shutdown calls", func() {
		const goroutines = 10
		errors := make([]error, goroutines)
		done := make(chan bool)

		ctx := context.Background()

		// Multiple goroutines try to shutdown simultaneously
		for i := range goroutines {
			go func(index int) {
				errors[index] = metricsSetup.Shutdown(ctx)
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
