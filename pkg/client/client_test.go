// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"net/url"
	"os"
	"time"

	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	valiflag "github.com/credativ/vali/pkg/util/flagext"
	"github.com/credativ/vali/pkg/valitail/client"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/logging/pkg/config"
)

var _ = ginkgov2.Describe("Client", func() {
	defaultURL, _ := parseURL("http://localhost:3100/vali/api/v1/push")
	var infoLogLevel logging.Level
	_ = infoLogLevel.Set("info")
	conf := config.Config{
		ClientConfig: config.ClientConfig{
			CredativValiConfig: client.Config{
				URL:            defaultURL,
				TenantID:       "", // empty as not set in fluent-bit plugin config map
				BatchSize:      100,
				BatchWait:      30 * time.Second,
				ExternalLabels: valiflag.LabelSet{LabelSet: model.LabelSet{"app": "foo"}},
				BackoffConfig: util.BackoffConfig{
					MinBackoff: (1 * time.Second),
					MaxBackoff: 300 * time.Second,
					MaxRetries: 10,
				},
				Timeout: 10 * time.Second,
			},
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
			LabelKeys:     []string{"foo", "bar"},
			RemoveKeys:    []string{"buzz", "fuzz"},
			DropSingleKey: false,
			DynamicHostPath: map[string]any{
				"kubernetes": map[string]any{
					"namespace_name": "namespace",
				},
			},
			DynamicHostRegex: "shoot--",
			LineFormat:       config.KvPairFormat,
		},
		LogLevel: infoLogLevel,
		ControllerConfig: config.ControllerConfig{
			DynamicHostPrefix: "http://vali.",
			DynamicHostSuffix: ".svc:3100/vali/api/v1/push",
		},
	}

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, infoLogLevel.Gokit)

	ginkgov2.Describe("NewClient", func() {
		ginkgov2.It("should create a client", func() {
			c, err := NewClient(conf, logger, Options{})
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(c).ToNot(gomega.BeNil())
		})
	})
})

func parseURL(u string) (flagext.URLValue, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return flagext.URLValue{}, err
	}

	return flagext.URLValue{URL: parsed}, nil
}
