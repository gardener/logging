// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"os"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/logging/pkg/config"
)

var _ = Describe("Buffer", func() {
	var infoLogLevel logging.Level
	_ = infoLogLevel.Set("info")
	var conf config.Config

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, infoLogLevel.Gokit)

	Describe("NewBuffer", func() {
		BeforeEach(func() {
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
		AfterEach(func() {
			os.RemoveAll("/tmp/dque")
		})
		It("should create a buffered client when buffer is set", func() {
			conf := conf
			conf.ClientConfig.BufferConfig.Buffer = true
			c, err := NewBuffer(conf, logger, newFakeValiClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).ToNot(BeNil())
		})

		It("should not create a buffered client when buffer type is wrong", func() {
			conf := conf
			conf.ClientConfig.BufferConfig.BufferType = "wrong-buffer"
			c, err := NewBuffer(conf, logger, newFakeValiClient)
			Expect(err).To(HaveOccurred())
			Expect(c).To(BeNil())
		})
	})

	Describe("newDque", func() {
		var valiclient ValiClient

		BeforeEach(func() {
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
			Expect(err).ToNot(HaveOccurred())
			Expect(valiclient).ToNot(BeNil())
		})
		AfterEach(func() {
			err := os.RemoveAll("/tmp/gardener")
			Expect(err).ToNot(HaveOccurred())
		})
		It("should sent log successfully", func() {
			ls := model.LabelSet{
				"foo": "bar",
			}
			ts := time.Now()
			line := "this is the message"
			err := valiclient.Handle(ls, ts, line)
			Expect(err).ToNot(HaveOccurred())
			dQueCleint, ok := valiclient.(*dqueClient)
			Expect(ok).To(BeTrue())
			fakeVali, ok := dQueCleint.vali.(*fakeValiclient)
			Expect(ok).To(BeTrue())
			time.Sleep(2 * time.Second)
			fakeVali.mu.Lock()
			defer fakeVali.mu.Unlock()
			log := fakeVali.sentLogs[0]
			Expect(log.labelSet).To(Equal(ls))
			Expect(log.timestamp).To(Equal(ts))
			Expect(log.line).To(Equal(line))
		})
		It("should stop correctly", func() {
			valiclient.Stop()
			dQueCleint, ok := valiclient.(*dqueClient)
			Expect(ok).To(BeTrue())
			fakeVali, ok := dQueCleint.vali.(*fakeValiclient)
			Expect(ok).To(BeTrue())
			time.Sleep(2 * time.Second)
			fakeVali.mu.Lock()
			defer fakeVali.mu.Unlock()
			Expect(fakeVali.stopped).To(BeTrue())
			_, err := os.Stat("/tmp/gardener")
			Expect(os.IsNotExist(err)).To(BeFalse())
		})
		It("should gracefully stop correctly", func() {
			valiclient.StopWait()
			dQueCleint, ok := valiclient.(*dqueClient)
			Expect(ok).To(BeTrue())
			fakeVali, ok := dQueCleint.vali.(*fakeValiclient)
			Expect(ok).To(BeTrue())
			time.Sleep(2 * time.Second)
			fakeVali.mu.Lock()
			defer fakeVali.mu.Unlock()
			Expect(fakeVali.stopped).To(BeTrue())
			_, err := os.Stat("/tmp/gardener")
			Expect(os.IsNotExist(err)).To(BeTrue())
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

func newFakeValiClient(c config.Config, logger log.Logger) (ValiClient, error) {
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
