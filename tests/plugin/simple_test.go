// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/log"
	"github.com/gardener/logging/pkg/types"
)

var _ = Describe("Simple Plugin Test", func() {
	It("should create a NoopClient with logr logger", func() {
		logger := log.NewNopLogger()
		cfg := config.Config{
			ClientConfig: config.ClientConfig{
				SeedType: types.NOOP,
			},
			OTLPConfig: config.OTLPConfig{
				Endpoint: "http://test:4318",
			},
		}

		c, err := client.NewNoopClient(cfg, logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(c).NotTo(BeNil())

		// Test handle
		err = c.Handle(time.Now(), "test log")
		Expect(err).NotTo(HaveOccurred())

		// Test cleanup
		c.Stop()
		c.StopWait()
	})

	It("should create a logger with slog backend", func() {
		logger := log.NewLogger("info")
		Expect(logger).NotTo(BeNil())

		// Test logging calls
		logger.Info("test message", "key", "value")
		logger.V(1).Info("debug message")
		logger.Error(nil, "error message")
	})
})
