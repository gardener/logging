// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package buffer

import (
	"os"
	"time"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/types"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/loki/pkg/promtail/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"
)

var _ = Describe("Buffer", func() {
	var infoLogLevel logging.Level
	_ = infoLogLevel.Set("info")
	var conf *config.Config

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = level.NewFilter(logger, infoLogLevel.Gokit)

	Describe("NewBuffer", func() {
		BeforeEach(func() {
			conf = &config.Config{
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
			c, err := NewBuffer(conf, logger, newFakeLokiClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(c).ToNot(BeNil())
		})

		It("should not create a buffered client when buffer type is wrong", func() {
			conf := conf
			conf.ClientConfig.BufferConfig.BufferType = "wrong-buffer"
			c, err := NewBuffer(conf, logger, newFakeLokiClient)
			Expect(err).To(HaveOccurred())
			Expect(c).To(BeNil())
		})
	})

	Describe("newDque", func() {
		var lokiclient types.LokiClient

		BeforeEach(func() {
			var err error
			conf = &config.Config{
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
			lokiclient, err = newDque(conf, logger, newFakeLokiClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(lokiclient).ToNot(BeNil())
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
			err := lokiclient.Handle(ls, ts, line)
			Expect(err).ToNot(HaveOccurred())
			dQueCleint, ok := lokiclient.(*dqueClient)
			Expect(ok).To(BeTrue())
			fakeLoki, ok := dQueCleint.loki.(*fakeLokiclient)
			Expect(ok).To(BeTrue())
			time.Sleep(2 * time.Second)
			log := fakeLoki.sentLogs[0]
			Expect(log.labelSet).To(Equal(ls))
			Expect(log.timestamp).To(Equal(ts))
			Expect(log.line).To(Equal(line))
			lokiclient.StopWait()
		})
		It("should gracefully stop correctly", func() {
			lokiclient.StopWait()
			dQueCleint, ok := lokiclient.(*dqueClient)
			Expect(ok).To(BeTrue())
			fakeLoki, ok := dQueCleint.loki.(*fakeLokiclient)
			Expect(ok).To(BeTrue())
			time.Sleep(2 * time.Second)
			Expect(fakeLoki.stopped).To(BeTrue())
			_, err := os.Stat("/tmp/gardener")
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		It("should stop correctly", func() {
			lokiclient.Stop()
			dQueCleint, ok := lokiclient.(*dqueClient)
			Expect(ok).To(BeTrue())
			fakeLoki, ok := dQueCleint.loki.(*fakeLokiclient)
			Expect(ok).To(BeTrue())
			time.Sleep(2 * time.Second)
			Expect(fakeLoki.stopped).To(BeTrue())
			_, err := os.Stat("/tmp/gardener")
			Expect(os.IsNotExist(err)).To(BeFalse())
		})
	})

})

type fakeLokiclient struct {
	stopped  bool
	sentLogs []logEntry
}

func newFakeLokiClient(c client.Config, logger log.Logger) (types.LokiClient, error) {
	return &fakeLokiclient{}, nil
}

func (c *fakeLokiclient) Handle(labels model.LabelSet, time time.Time, entry string) error {
	c.sentLogs = append(c.sentLogs, logEntry{time, labels, entry})
	return nil
}

func (c *fakeLokiclient) Stop() {
	c.stopped = true
}

func (c *fakeLokiclient) StopWait() {
	c.stopped = true
}

type logEntry struct {
	timestamp time.Time
	labelSet  model.LabelSet
	line      string
}
