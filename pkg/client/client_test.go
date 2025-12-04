// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	"github.com/go-logr/logr"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/gardener/logging/v1/pkg/config"
)

var _ = ginkgov2.Describe("Client", func() {
	conf := config.Config{
		ClientConfig: config.ClientConfig{
			BufferConfig: config.BufferConfig{
				Buffer: false,
				DqueConfig: config.DqueConfig{
					QueueDir:         config.DefaultDqueConfig.QueueDir,
					QueueSegmentSize: config.DefaultDqueConfig.QueueSegmentSize,
					QueueSync:        config.DefaultDqueConfig.QueueSync,
					QueueName:        config.DefaultDqueConfig.QueueName,
				},
			},
		},
		PluginConfig: config.PluginConfig{
			DynamicHostPath: map[string]any{
				"kubernetes": map[string]any{
					"namespace_name": "namespace",
				},
			},
			DynamicHostRegex: "shoot--",
		},
		LogLevel: "info",
		ControllerConfig: config.ControllerConfig{
			DynamicHostPrefix: "localhost",
			DynamicHostSuffix: ":4317",
		},
	}

	logger := logr.Discard()

	ginkgov2.Describe("NewClient", func() {
		ginkgov2.It("should create a client", func() {
			c, err := NewClient(
				context.Background(),
				conf,
				WithLogger(logger),
			)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(c).ToNot(gomega.BeNil())
		})
	})
})
