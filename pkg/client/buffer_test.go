// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"os"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/logging/pkg/config"
)

var _ = ginkgov2.Describe("Buffer", func() {
	var infoLogLevel logging.Level
	_ = infoLogLevel.Set("info")
	var conf config.Config

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, infoLogLevel.Gokit)

	ginkgov2.Describe("NewBuffer", func() {
		ginkgov2.BeforeEach(func() {
			conf = config.Config{
				ClientConfig: config.ClientConfig{
					BufferConfig: config.BufferConfig{
						Buffer:     false,
						BufferType: "dque",
						DqueConfig: config.DqueConfig{
							QueueDir:         "/tmp/",
							QueueSegmentSize: 500,
							QueueSync:        false,
							QueueName:        "dque",
						},
					},
				},
			}
		})
		ginkgov2.AfterEach(func() {
			_ = os.RemoveAll("/tmp/dque")
		})

		ginkgov2.It("should create a buffered client when buffer is set", func() {

		})

		ginkgov2.It("should not create a buffered client when buffer type is wrong", func() {

		})
	})

	ginkgov2.Describe("newDque", func() {
		var valiclient OutputClient

		ginkgov2.BeforeEach(func() {
			var err error
			conf = config.Config{
				ClientConfig: config.ClientConfig{
					BufferConfig: config.BufferConfig{
						Buffer:     false,
						BufferType: "dque",
						DqueConfig: config.DqueConfig{
							QueueDir:         "/tmp/",
							QueueSegmentSize: 500,
							QueueSync:        false,
							QueueName:        "gardener",
						},
					},
				},
			}
			valiclient, err = NewDque(conf, logger, newFakeClient)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(valiclient).ToNot(gomega.BeNil())
		})
		ginkgov2.AfterEach(func() {
			err := os.RemoveAll("/tmp/gardener")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})
		ginkgov2.It("should sent log successfully", func() {})
		ginkgov2.It("should stop correctly", func() {})
		ginkgov2.It("should gracefully stop correctly", func() {})
	})
})

type fakeClient struct {
	stopped bool
	mu      sync.Mutex
}

func (*fakeClient) GetEndPoint() string {
	return "http://localhost"
}

var _ OutputClient = &fakeClient{}

func newFakeClient(_ config.Config, _ log.Logger) (OutputClient, error) {
	return &fakeClient{}, nil
}

func (c *fakeClient) Handle(_ time.Time, _ string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return nil
}

func (c *fakeClient) Stop() {
	c.stopped = true
}

func (c *fakeClient) StopWait() {
	c.stopped = true
}
