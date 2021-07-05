/*
This file was copied from the grafana/loki project
https://github.com/grafana/loki/blob/v1.6.0/cmd/fluent-bit/dque.go

Modifications Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved.
*/
package buffer

import (
	"fmt"
	"os"
	"path"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
	"github.com/gardener/logging/pkg/types"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/grafana/loki/pkg/promtail/client"
	"github.com/joncrlsn/dque"
	"github.com/prometheus/common/model"
)

var openDQueProfile = pprof.NewProfile("openedDQue")
var openDQueFilesProfile = pprof.NewProfile("openedDQueFiles")

type dqueEntry struct {
	LabelSet model.LabelSet
	logproto.Entry
	end bool
}

func dqueEntryBuilder() interface{} {
	return &dqueEntry{}
}

type dqueClient struct {
	logger                    log.Logger
	queue                     *dque.DQue
	loki                      client.Client
	once                      sync.Once
	wg                        sync.WaitGroup
	url                       string
	cleanUnderlyingFileBuffer bool
}

// newDque makes a new dque loki client
func newDque(cfg *config.Config, logger log.Logger, newClientFunc func(cfg client.Config, logger log.Logger) (types.LokiClient, error)) (types.LokiClient, error) {
	var err error

	q := &dqueClient{
		logger: log.With(logger, "component", "queue", "name", cfg.ClientConfig.BufferConfig.DqueConfig.QueueName),
	}

	err = os.MkdirAll(cfg.ClientConfig.BufferConfig.DqueConfig.QueueDir, 0644)
	if err != nil {
		return nil, fmt.Errorf("cannot create queue directory, error: %v", err)
	}

	q.queue, err = dque.NewOrOpen(cfg.ClientConfig.BufferConfig.DqueConfig.QueueName, cfg.ClientConfig.BufferConfig.DqueConfig.QueueDir, cfg.ClientConfig.BufferConfig.DqueConfig.QueueSegmentSize, dqueEntryBuilder)
	if err != nil {
		return nil, err
	}

	q.url = cfg.ClientConfig.GrafanaLokiConfig.URL.String()

	if !cfg.ClientConfig.BufferConfig.DqueConfig.QueueSync {
		_ = q.queue.TurboOn()
	}

	q.loki, err = newClientFunc(cfg.ClientConfig.GrafanaLokiConfig, logger)
	if err != nil {
		return nil, err
	}

	q.cleanUnderlyingFileBuffer = cfg.ClientConfig.BufferConfig.DqueConfig.CleanUnderlyingFileBuffer

	q.wg.Add(1)
	go q.dequeuer()
	openDQueProfile.Add(path.Join(q.queue.DirPath, q.queue.Name), 3)
	openDQueFilesProfile.Add(path.Join(q.queue.DirPath, q.queue.Name), 3)
	return q, nil
}

func (c *dqueClient) dequeuer() {
	defer c.wg.Done()

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

		if record.end {
			//TODO: What if the final ending record is malformed here
			return
		}

		level.Debug(c.logger).Log("msg", "sending record to Loki", "url", c.url, "record", record.String())
		if err := c.loki.Handle(record.LabelSet, record.Timestamp, record.Line); err != nil {
			metrics.Errors.WithLabelValues(metrics.ErrorDequeuerSendRecord).Inc()
			level.Error(c.logger).Log("msg", fmt.Sprintf("error sending record to Loki %s", c.url), "error", err)
		}
		level.Debug(c.logger).Log("msg", "successful sent record to Loki", "host", c.url, "record", record.String())
	}
}

// Stop the client
func (c *dqueClient) Stop() {
	c.once.Do(func() {
		if err := c.closeQue(false); err != nil {
			level.Error(c.logger).Log("msg", "error closing buffered client", "queue", c.queue.Name, "err", err.Error())
		}
		c.loki.Stop()
	})
}

// Stop the client
func (c *dqueClient) StopWait() {
	c.once.Do(func() {
		if err := c.sendQueStopSignal(); err != nil {
			level.Error(c.logger).Log("msg", "error closing buffered client", "queue", c.queue.Name, "err", err.Error())
		}
		if err := c.closeQue(true); err != nil {
			level.Error(c.logger).Log("msg", "error closing buffered client", "queue", c.queue.Name, "err", err.Error())
		}
		c.loki.Stop()
	})
}

// Handle implement EntryHandler; adds a new line to the next batch; send is async.
func (c *dqueClient) Handle(ls model.LabelSet, t time.Time, s string) error {

	record := &dqueEntry{LabelSet: ls, Entry: logproto.Entry{Timestamp: t, Line: s}}
	if err := c.queue.Enqueue(record); err != nil {
		return fmt.Errorf("cannot enqueue record %s: %v", record.String(), err)
	}

	return nil
}

func (e *dqueEntry) String() string {
	return fmt.Sprintf("labels: %+v timestamp: %+v line: %+v", e.LabelSet, e.Timestamp, e.Line)
}

func (c *dqueClient) sendQueStopSignal() error {

	record := &dqueEntry{end: true}
	//TODO: make more effective shutdown mechanism when the queue is full.
	if err := c.queue.Enqueue(record); err != nil {
		return fmt.Errorf("cannot enqueue end signal record to %s buffer: %v", c.queue.Name, err)
	}

	//TODO: Make time group waiter
	c.wg.Wait()
	return nil
}

func (c *dqueClient) closeQue(cleanUnderlyingFileBuffer bool) error {

	if err := c.queue.Close(); err != nil {
		return fmt.Errorf("cannot close %s buffer: %v", c.queue.Name, err)
	}
	openDQueProfile.Remove(path.Join(c.queue.DirPath, c.queue.Name))

	if cleanUnderlyingFileBuffer {
		if err := os.RemoveAll(path.Join(c.queue.DirPath, c.queue.Name)); err != nil {
			return fmt.Errorf("cannot close %s buffer: %v", c.queue.Name, err)
		}
		openDQueFilesProfile.Remove(path.Join(c.queue.DirPath, c.queue.Name))
	}

	return nil
}
