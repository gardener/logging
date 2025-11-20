/*
This file was copied from the credativ/client project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/dque.go

Modifications Copyright SAP SE or an SAP affiliate company and Gardener contributors
*/
package client

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/joncrlsn/dque"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
)

const componentNameDque = "dque"

// OutputEntry is a single log entry with timestamp
type OutputEntry struct {
	Timestamp time.Time
	Line      string
}

type dqueEntry struct {
	LabelSet map[string]string
	OutputEntry
}

func dqueEntryBuilder() any {
	return &dqueEntry{}
}

type dqueClient struct {
	logger    logr.Logger
	queue     *dque.DQue
	client    OutputClient
	wg        sync.WaitGroup
	isStooped bool
	lock      sync.Mutex
	turboOn   bool
}

func (c *dqueClient) GetEndPoint() string {
	return c.client.GetEndPoint()
}

var _ OutputClient = &dqueClient{}

// NewDque makes a new dque client
func NewDque(cfg config.Config, logger logr.Logger, newClientFunc NewClientFunc) (OutputClient, error) {
	var err error

	q := &dqueClient{
		logger: logger.WithValues("component", componentNameDque, "name", cfg.ClientConfig.BufferConfig.DqueConfig.QueueName),
	}

	if err = os.MkdirAll(cfg.ClientConfig.BufferConfig.DqueConfig.QueueDir, fs.FileMode(0644)); err != nil {
		return nil, fmt.Errorf("cannot create directory %s: %v", cfg.ClientConfig.BufferConfig.DqueConfig.QueueDir, err)
	}

	q.queue, err = dque.NewOrOpen(cfg.ClientConfig.BufferConfig.DqueConfig.QueueName, cfg.ClientConfig.BufferConfig.DqueConfig.QueueDir, cfg.ClientConfig.BufferConfig.DqueConfig.QueueSegmentSize, dqueEntryBuilder)
	if err != nil {
		return nil, fmt.Errorf("cannot create queue %s: %v", cfg.ClientConfig.BufferConfig.DqueConfig.QueueName, err)
	}

	if !cfg.ClientConfig.BufferConfig.DqueConfig.QueueSync {
		q.turboOn = true
		if err = q.queue.TurboOn(); err != nil {
			q.logger.Error(err, "cannot enable turbo mode for queue")
		}
	}

	q.client, err = newClientFunc(cfg, logger)
	if err != nil {
		return nil, err
	}

	q.wg.Add(1)
	go q.dequeuer()

	q.logger.V(1).Info("client created")

	return q, nil
}

func (c *dqueClient) dequeuer() {
	defer c.wg.Done()
	c.logger.V(2).Info("dequeuer started")

	timer := time.NewTicker(30 * time.Second)
	defer timer.Stop()

	for {
		// Dequeue the next item in the queue
		entry, err := c.queue.DequeueBlock()
		if err != nil {
			switch err {
			case dque.ErrQueueClosed:
				return
			default:
				metrics.Errors.WithLabelValues(metrics.ErrorDequeuer).Inc()
				c.logger.Error(err, "error dequeue record")

				continue
			}
		}

		// Update queue size metric

		select {
		case <-timer.C:
			size := c.queue.Size()
			metrics.DqueSize.WithLabelValues(c.queue.Name).Set(float64(size))
			if c.turboOn {
				if err = c.queue.TurboSync(); err != nil {
					_ = c.logger.Log("msg", "turbo sync", "err", err)
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

		if err := c.client.Handle(record.Timestamp, record.Line); err != nil {
			metrics.Errors.WithLabelValues(metrics.ErrorDequeuerSendRecord).Inc()
			c.logger.Error(err, "error sending record to Vali")
		}

		c.lock.Lock()
		if c.isStopped && c.queue.Size() <= 0 {
			c.lock.Unlock()

			return
		}
		c.lock.Unlock()
	}
}

// Stop the client
func (c *dqueClient) Stop() {
	if err := c.closeQue(); err != nil {
		c.logger.Error(err, "error closing buffered client")
	}
	c.client.Stop()
	c.logger.V(1).Info("client stopped, without waiting")
}

// StopWait the client waiting all saved logs to be sent.
func (c *dqueClient) StopWait() {
	if err := c.stopQue(); err != nil {
		c.logger.Error(err, "error stopping buffered client")
	}
	if err := c.closeQueWithClean(); err != nil {
		c.logger.Error(err, "error closing buffered client")
	}
	c.client.StopWait()

	c.logger.V(1).Info("client stopped")
}

// Handle implement EntryHandler; adds a new line to the next batch; send is async.
func (c *dqueClient) Handle(t time.Time, line string) error {
	// Here we don't need any synchronization because the worst thing is to
	// receive some more logs which would be dropped anyway.
	if c.isStopped {
		return nil
	}

	entry := &dqueEntry{
		OutputEntry: OutputEntry{
			Timestamp: t,
			Line:      line,
		},
	}

	if err := c.queue.Enqueue(entry); err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorEnqueuer).Inc()

		return fmt.Errorf("failed to enqueue log entry: %w", err)
	}

	return nil
}

func (e *dqueEntry) String() string {
	return fmt.Sprintf("labels: %+v timestamp: %+v line: %+v", e.LabelSet, e.Timestamp, e.Line)
}

func (c *dqueClient) stopQue() error {
	c.lock.Lock()
	c.isStopped = true
	// In case the dequeuer is blocked on empty queue.
	if c.queue.Size() == 0 {
		c.lock.Unlock() // Nothing to wait for

		return nil
	}
	c.lock.Unlock()
	// TODO: Make time group waiter
	c.wg.Wait()

	return nil
}

func (c *dqueClient) closeQue() error {
	if err := c.queue.Close(); err != nil {
		return fmt.Errorf("cannot close %s buffer: %v", c.queue.Name, err)
	}

	return nil
}

func (c *dqueClient) closeQueWithClean() error {
	if err := c.closeQue(); err != nil {
		return fmt.Errorf("cannot close %s buffer: %v", c.queue.Name, err)
	}
	if err := os.RemoveAll(path.Join(c.queue.DirPath, c.queue.Name)); err != nil {
		return fmt.Errorf("cannot clean %s buffer: %v", c.queue.Name, err)
	}

	return nil
}
