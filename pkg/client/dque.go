// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/joncrlsn/dque"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

const componentNameDque = "dque"
const syncTimeout = 30 * time.Second

type dqueEntry struct {
	types.OutputEntry
}

func init() {
	gob.Register(map[string]any{})
	gob.Register(types.OutputRecord{})
}

func dqueEntryBuilder() any {
	return &dqueEntry{}
}

type dqueClient struct {
	logger  logr.Logger
	queue   *dque.DQue
	client  OutputClient
	wg      sync.WaitGroup
	stopped bool
	turboOn bool
	lock    sync.Mutex
}

func (c *dqueClient) GetEndPoint() string {
	return c.client.GetEndPoint()
}

var _ OutputClient = &dqueClient{}

// NewDque makes a new dque client
func NewDque(cfg config.Config, logger logr.Logger, newClientFunc NewClientFunc) (OutputClient, error) {
	var err error

	qDir := cfg.ClientConfig.BufferConfig.DqueConfig.QueueDir
	qName := cfg.ClientConfig.BufferConfig.DqueConfig.QueueName
	qSync := cfg.ClientConfig.BufferConfig.DqueConfig.QueueSync
	qSize := cfg.ClientConfig.BufferConfig.DqueConfig.QueueSegmentSize

	q := &dqueClient{
		logger: logger.WithValues(
			"name", qName,
		),
	}

	if err = os.MkdirAll(qDir, fs.FileMode(0644)); err != nil {
		return nil, fmt.Errorf("cannot create directory %s: %v", qDir, err)
	}

	q.queue, err = dque.NewOrOpen(qName, qDir, qSize, dqueEntryBuilder)
	if err != nil {
		return nil, fmt.Errorf("cannot create queue %s: %v", qName, err)
	}

	if !qSync {
		q.turboOn = true
		if err = q.queue.TurboOn(); err != nil {
			q.turboOn = false
			q.logger.Error(err, "cannot enable turbo mode for queue")
		}
	}

	// Create the upstream client
	if q.client, err = newClientFunc(cfg, logger); err != nil {
		return nil, err
	}

	q.wg.Go(q.dequeuer)

	q.logger.Info(fmt.Sprintf("%s created", componentNameDque))

	return q, nil
}

func (c *dqueClient) dequeuer() {
	c.logger.V(2).Info("starting dequeuer")

	timer := time.NewTicker(syncTimeout)
	defer timer.Stop()

	for {
		// Dequeue the next item in the queue
		entry, err := c.queue.DequeueBlock()
		if err != nil {
			switch {
			case errors.Is(err, dque.ErrQueueClosed):
				// Queue closed is expected during shutdown, log at info level
				c.logger.V(1).Info("dequeuer stopped gracefully, queue closed")

				return
			default:
				metrics.Errors.WithLabelValues(metrics.ErrorDequeuer).Inc()
				c.logger.Error(err, "error dequeue record")

				continue
			}
		}

		select {
		case <-timer.C:
			size := c.queue.Size()
			metrics.DqueSize.WithLabelValues(c.queue.Name).Set(float64(size))
			if c.turboOn {
				if err = c.queue.TurboSync(); err != nil {
					c.logger.Error(err, "error turbo sync")
				}
			}

		default:
			// Do nothing and continue
		}

		// Assert type of the response to an Item pointer so we can work with it
		record, ok := entry.(*dqueEntry)
		if !ok {
			metrics.Errors.WithLabelValues(metrics.ErrorDequeuerNotValidType).Inc()
			c.logger.Error(nil, "error record is not a valid type")

			continue
		}

		if err = c.client.Handle(record.OutputEntry); err != nil {
			metrics.Errors.WithLabelValues(metrics.ErrorDequeuerSendRecord).Inc()
			c.logger.Error(err, "error sending record to upstream client")
		}

		c.lock.Lock()
		if c.stopped && c.queue.Size() <= 0 {
			c.lock.Unlock()

			return
		}
		c.lock.Unlock()
	}
}

// Handle implement EntryHandler; adds a new line to the next batch; send is async.
func (c *dqueClient) Handle(log types.OutputEntry) error {
	// Here we don't need any synchronization because the worst thing is to
	// receive some more logs which would be dropped anyway.
	if c.stopped {
		return nil
	}

	entry := &dqueEntry{
		OutputEntry: log,
	}

	if err := c.queue.Enqueue(entry); err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorEnqueuer).Inc()

		return fmt.Errorf("failed to enqueue log entry: %w", err)
	}

	return nil
}

// Stop the client
func (c *dqueClient) Stop() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s", componentNameDque))
	if err := c.queue.Close(); err != nil {
		c.logger.Error(err, "error closing queue")
	}
	c.client.Stop()
}

// StopWait the client waiting all saved logs to be sent.
func (c *dqueClient) StopWait() {
	c.logger.V(2).Info(fmt.Sprintf("stopping %s with wait", componentNameDque))
	if err := c.stopQueWithTimeout(); err != nil {
		c.logger.Error(err, "error stopping client")
	}
	if err := c.closeQueWithClean(); err != nil {
		c.logger.Error(err, "error closing client")
	}
	c.client.StopWait() // Stop the underlying client
}

func (c *dqueClient) stopQueWithTimeout() error {
	c.lock.Lock()
	c.stopped = true
	// In case the dequeuer is blocked on empty queue.
	if c.queue.Size() == 0 {
		c.lock.Unlock() // Nothing to wait for

		return nil
	}
	c.lock.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	done := make(chan struct{})
	go func() {
		c.wg.Wait() // Wait for dequeuer to finish
		close(done)
	}()

	select {
	case <-ctx.Done():
		// Force close the queue to unblock the dequeuer goroutine and prevent leak
		if err := c.queue.Close(); err != nil {
			c.logger.Error(err, "error force closing queue after timeout %v", syncTimeout)
		}

		return nil
	case <-done:
		return nil
	}
}

func (c *dqueClient) closeQueWithClean() error {
	return os.RemoveAll(path.Join(c.queue.DirPath, c.queue.Name))
}
