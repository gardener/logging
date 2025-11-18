// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/logging/pkg/config"
)

var _ = ginkgov2.Describe("Client", func() {
	var infoLogLevel logging.Level
	_ = infoLogLevel.Set("info")
	conf := config.Config{
		ClientConfig: config.ClientConfig{
			BufferConfig: config.BufferConfig{
				Buffer:     false,
				BufferType: config.DefaultBufferConfig.BufferType,
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
		LogLevel: infoLogLevel,
		ControllerConfig: config.ControllerConfig{
			DynamicHostPrefix: "http://vali.",
			DynamicHostSuffix: ".svc:3100/client/api/v1/push",
		},
	}

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, infoLogLevel.Gokit)

	ginkgov2.Describe("NewClient", func() {
		ginkgov2.It("should create a client", func() {
			c, err := NewClient(
				conf,
				WithLogger(logger),
			)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(c).ToNot(gomega.BeNil())
		})
	})
})
