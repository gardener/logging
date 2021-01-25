/*
This file was copied from the grafana/loki project
https://github.com/grafana/loki/blob/v1.6.0/cmd/fluent-bit/dque.go

Modifications Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
*/
package buffer

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/loki/pkg/promtail/client"
	"github.com/joncrlsn/dque"
	"github.com/prometheus/common/model"
)

type dqueEntry struct {
	LabelSet  model.LabelSet
	TimeStamp time.Time
	Line      string
}

func dqueEntryBuilder() interface{} {
	return &dqueEntry{}
}

type dqueClient struct {
	logger log.Logger
	queue  *dque.DQue
	loki   client.Client
	once   sync.Once
	url    string
}

// newDque makes a new dque loki client
func newDque(cfg *config.Config, logger log.Logger, newClientFunc func(cfg client.Config, logger log.Logger) (client.Client, error)) (client.Client, error) {
	var err error

	q := &dqueClient{
		logger: log.With(logger, "component", "queue", "name", cfg.BufferConfig.DqueConfig.QueueName),
	}

	err = os.MkdirAll(cfg.BufferConfig.DqueConfig.QueueDir, 0644)
	if err != nil {
		return nil, fmt.Errorf("cannot create queue directory, error: %v", err)
	}

	q.queue, err = dque.NewOrOpen(cfg.BufferConfig.DqueConfig.QueueName, cfg.BufferConfig.DqueConfig.QueueDir, cfg.BufferConfig.DqueConfig.QueueSegmentSize, dqueEntryBuilder)
	if err != nil {
		return nil, err
	}

	q.url = cfg.ClientConfig.URL.String()

	if !cfg.BufferConfig.DqueConfig.QueueSync {
		_ = q.queue.TurboOn()
	}

	q.loki, err = newClientFunc(cfg.ClientConfig, logger)
	if err != nil {
		return nil, err
	}

	go q.dequeuer()
	return q, nil
}

func (c *dqueClient) dequeuer() {
	for {
		// Dequeue the next item in the queue
		entry, err := c.queue.DequeueBlock()
		if err != nil {
			switch err {
			case dque.ErrQueueClosed:
				return
			default:
				metrics.Errors.WithLabelValues(metrics.ErrorDequeuer).Inc()
				level.Error(c.logger).Log("msg", "error dequeuing record", "error", err, "queue", c.queue.Name)
				continue
			}
		}

		// Assert type of the response to an Item pointer so we can work with it
		record, ok := entry.(*dqueEntry)
		if !ok {
			metrics.Errors.WithLabelValues(metrics.ErrorDequeuerNotValidType).Inc()
			level.Error(c.logger).Log("msg", "error dequeued record is not an valid type", "queue", c.queue.Name)
			continue
		}

		level.Debug(c.logger).Log("msg", "sending record to Loki", "url", c.url, "record", record.String())
		if err := c.loki.Handle(record.LabelSet, record.TimeStamp, record.Line); err != nil {
			metrics.Errors.WithLabelValues(metrics.ErrorDequeuerSendRecord).Inc()
			level.Error(c.logger).Log("msg", fmt.Sprintf("error sending record to Loki %s", c.url), "error", err)
		}
		level.Debug(c.logger).Log("msg", "successful sent record to Loki", "host", c.url, "record", record.String())
	}
}

// Stop the client
func (c *dqueClient) Stop() {
	c.once.Do(func() {
		c.queue.Close()
		c.loki.Stop()
	})

}

// Handle implement EntryHandler; adds a new line to the next batch; send is async.
func (c *dqueClient) Handle(ls model.LabelSet, t time.Time, s string) error {
	record := &dqueEntry{ls, t, s}
	if err := c.queue.Enqueue(record); err != nil {
		return fmt.Errorf("cannot enqueue record %s: %v", record.String(), err)
	}

	return nil
}

func (e *dqueEntry) String() string {
	return fmt.Sprintf("labels: %+v timestamp: %+v line: %+v", e.LabelSet, e.TimeStamp, e.Line)
}
