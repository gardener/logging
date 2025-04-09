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
	"github.com/prometheus/common/model"
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
			conf := conf
			conf.ClientConfig.BufferConfig.Buffer = true
			c, err := NewBuffer(conf, logger, newFakeValiClient)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(c).ToNot(gomega.BeNil())
		})

		ginkgov2.It("should not create a buffered client when buffer type is wrong", func() {
			conf := conf
			conf.ClientConfig.BufferConfig.BufferType = "wrong-buffer"
			c, err := NewBuffer(conf, logger, newFakeValiClient)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(c).To(gomega.BeNil())
		})
	})

	ginkgov2.Describe("newDque", func() {
		var valiclient ValiClient

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
			valiclient, err = NewDque(conf, logger, newFakeValiClient)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(valiclient).ToNot(gomega.BeNil())
		})
		ginkgov2.AfterEach(func() {
			err := os.RemoveAll("/tmp/gardener")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		})
		ginkgov2.It("should sent log successfully", func() {
			ls := model.LabelSet{
				"foo": "bar",
			}
			ts := time.Now()
			line := "this is the message"
			err := valiclient.Handle(ls, ts, line)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			dQueCleint, ok := valiclient.(*dqueClient)
			gomega.Expect(ok).To(gomega.BeTrue())
			fakeVali, ok := dQueCleint.vali.(*fakeValiclient)
			gomega.Expect(ok).To(gomega.BeTrue())
			time.Sleep(2 * time.Second)
			fakeVali.mu.Lock()
			defer fakeVali.mu.Unlock()
			log := fakeVali.sentLogs[0]
			gomega.Expect(log.labelSet).To(gomega.Equal(ls))
			gomega.Expect(log.timestamp).To(gomega.Equal(ts))
			gomega.Expect(log.line).To(gomega.Equal(line))
		})
		ginkgov2.It("should stop correctly", func() {
			valiclient.Stop()
			dQueCleint, ok := valiclient.(*dqueClient)
			gomega.Expect(ok).To(gomega.BeTrue())
			fakeVali, ok := dQueCleint.vali.(*fakeValiclient)
			gomega.Expect(ok).To(gomega.BeTrue())
			time.Sleep(2 * time.Second)
			fakeVali.mu.Lock()
			defer fakeVali.mu.Unlock()
			gomega.Expect(fakeVali.stopped).To(gomega.BeTrue())
			_, err := os.Stat("/tmp/gardener")
			gomega.Expect(os.IsNotExist(err)).To(gomega.BeFalse())
		})
		ginkgov2.It("should gracefully stop correctly", func() {
			valiclient.StopWait()
			dQueCleint, ok := valiclient.(*dqueClient)
			gomega.Expect(ok).To(gomega.BeTrue())
			fakeVali, ok := dQueCleint.vali.(*fakeValiclient)
			gomega.Expect(ok).To(gomega.BeTrue())
			time.Sleep(2 * time.Second)
			fakeVali.mu.Lock()
			defer fakeVali.mu.Unlock()
			gomega.Expect(fakeVali.stopped).To(gomega.BeTrue())
			_, err := os.Stat("/tmp/gardener")
			gomega.Expect(os.IsNotExist(err)).To(gomega.BeTrue())
		})
	})

})

type fakeValiclient struct {
	stopped  bool
	sentLogs []logEntry
	mu       sync.Mutex
}

func (c *fakeValiclient) GetEndPoint() string {
	return "http://localhost"
}

var _ ValiClient = &fakeValiclient{}

func newFakeValiClient(_ config.Config, _ log.Logger) (ValiClient, error) {
	return &fakeValiclient{}, nil
}

func (c *fakeValiclient) Handle(labels model.LabelSet, time time.Time, entry string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sentLogs = append(c.sentLogs, logEntry{time, labels, entry})
	return nil
}

func (c *fakeValiclient) Stop() {
	c.stopped = true
}

func (c *fakeValiclient) StopWait() {
	c.stopped = true
}

type logEntry struct {
	timestamp time.Time
	labelSet  model.LabelSet
	line      string
}
