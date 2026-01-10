// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricsSetup Singleton", func() {
	It("should return the same instance on multiple calls", func() {
		setup1, err1 := NewMetricsSetup()
		Expect(err1).ToNot(HaveOccurred())
		Expect(setup1).ToNot(BeNil())

		setup2, err2 := NewMetricsSetup()
		Expect(err2).ToNot(HaveOccurred())
		Expect(setup2).ToNot(BeNil())

		// Should be the exact same instance (pointer equality)
		Expect(setup1).To(BeIdenticalTo(setup2))
	})

	It("should return the same meter provider on multiple calls", func() {
		setup1, _ := NewMetricsSetup()
		setup2, _ := NewMetricsSetup()

		provider1 := setup1.GetProvider()
		provider2 := setup2.GetProvider()

		// Should be the exact same provider instance
		Expect(provider1).To(BeIdenticalTo(provider2))
	})

	It("should work correctly with concurrent calls", func() {
		const goroutines = 10
		results := make([]*MetricsSetup, goroutines)
		errors := make([]error, goroutines)
		done := make(chan bool)

		for i := 0; i < goroutines; i++ {
			go func(index int) {
				results[index], errors[index] = NewMetricsSetup()
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < goroutines; i++ {
			<-done
		}

		// All should succeed
		for i := 0; i < goroutines; i++ {
			Expect(errors[i]).ToNot(HaveOccurred())
			Expect(results[i]).ToNot(BeNil())
		}

		// All should be the same instance
		firstInstance := results[0]
		for i := 1; i < goroutines; i++ {
			Expect(results[i]).To(BeIdenticalTo(firstInstance))
		}
	})

	It("should shutdown idempotently - multiple shutdowns should not error", func() {
		setup, err := NewMetricsSetup()
		Expect(err).ToNot(HaveOccurred())
		Expect(setup).ToNot(BeNil())

		ctx := context.Background()

		// First shutdown
		err = setup.Shutdown(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Second shutdown should not error (idempotent)
		err = setup.Shutdown(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Third shutdown should also not error
		err = setup.Shutdown(ctx)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should handle concurrent shutdown calls", func() {
		setup, err := NewMetricsSetup()
		Expect(err).ToNot(HaveOccurred())

		const goroutines = 10
		errors := make([]error, goroutines)
		done := make(chan bool)

		ctx := context.Background()

		// Multiple goroutines try to shutdown simultaneously
		for i := 0; i < goroutines; i++ {
			go func(index int) {
				errors[index] = setup.Shutdown(ctx)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < goroutines; i++ {
			<-done
		}

		// None should error (idempotent shutdown)
		for i := 0; i < goroutines; i++ {
			Expect(errors[i]).ToNot(HaveOccurred())
		}
	})
})
