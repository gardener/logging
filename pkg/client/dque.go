/*
This file was copied from the credativ/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/dque.go

Modifications Copyright SAP SE or an SAP affiliate company and Gardener contributors
*/
package client

import (
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/credativ/vali/pkg/logproto"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/joncrlsn/dque"
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
)

const componentNameDque = "dque"

type dqueEntry struct {
	LabelSet model.LabelSet
	logproto.Entry
}

func dqueEntryBuilder() interface{} {
	return &dqueEntry{}
}

type dqueClient struct {
	logger    log.Logger
	queue     *dque.DQue
	vali      ValiClient
	wg        sync.WaitGroup
	url       string
	isStooped bool
	lock      sync.Mutex
}

func (c *dqueClient) GetEndPoint() string {
	return c.vali.GetEndPoint()
}

var _ ValiClient = &dqueClient{}

// NewDque makes a new dque vali client
func NewDque(cfg config.Config, logger log.Logger, newClientFunc func(cfg config.Config,
	logger log.Logger) (ValiClient, error)) (ValiClient, error) {
	var err error

	if logger == nil {
		logger = log.NewNopLogger()
	}

	q := &dqueClient{
		logger: log.With(logger, "component", componentNameDque, "name", cfg.ClientConfig.BufferConfig.DqueConfig.QueueName),
	}

	if err = os.MkdirAll(cfg.ClientConfig.BufferConfig.DqueConfig.QueueDir, 0644); err != nil {
		return nil, fmt.Errorf("cannot create directory %s: %v", cfg.ClientConfig.BufferConfig.DqueConfig.QueueDir, err)
	}

	q.queue, err = dque.NewOrOpen(cfg.ClientConfig.BufferConfig.DqueConfig.QueueName, cfg.ClientConfig.BufferConfig.DqueConfig.QueueDir, cfg.ClientConfig.BufferConfig.DqueConfig.QueueSegmentSize, dqueEntryBuilder)
	if err != nil {
		return nil, fmt.Errorf("cannot create queue %s: %v", cfg.ClientConfig.BufferConfig.DqueConfig.QueueName, err)
	}

	q.url = cfg.ClientConfig.CredativValiConfig.URL.String()

	if !cfg.ClientConfig.BufferConfig.DqueConfig.QueueSync {
		if err = q.queue.TurboOn(); err != nil {
			_ = level.Error(q.logger).Log("msg", "cannot enable turbo mode for queue", "err", err)
		}
	}

	q.vali, err = newClientFunc(cfg, logger)
	if err != nil {
		return nil, err
	}

	q.wg.Add(1)
	go q.dequeuer()

	_ = level.Debug(q.logger).Log("msg", "client created", "url", q.url)
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
				_ = level.Error(c.logger).Log("msg", "error dequeue record", "err", err)
				continue
			}
		}

		// Assert type of the response to an Item pointer so we can work with it
		record, ok := entry.(*dqueEntry)
		if !ok {
			metrics.Errors.WithLabelValues(metrics.ErrorDequeuerNotValidType).Inc()
			_ = level.Error(c.logger).Log("msg", "error record is not a valid type")
			continue
		}

		if err := c.vali.Handle(record.LabelSet, record.Timestamp, record.Line); err != nil {
			metrics.Errors.WithLabelValues(metrics.ErrorDequeuerSendRecord).Inc()
			_ = level.Error(c.logger).Log("msg", "error sending record to Vali", "err", err, "url", c.url)
		}

		c.lock.Lock()
		if c.isStooped && c.queue.Size() <= 0 {
			c.lock.Unlock()
			return
		}
		c.lock.Unlock()
	}
}

// Stop the client
func (c *dqueClient) Stop() {

	if err := c.closeQue(false); err != nil {
		_ = level.Error(c.logger).Log("msg", "error closing buffered client", "err", err.Error())
	}
	c.vali.Stop()
	_ = level.Debug(c.logger).Log("msg", "client stopped, without waiting")

}

// StopWait the client waiting all saved logs to be sent.
func (c *dqueClient) StopWait() {

	if err := c.stopQue(true); err != nil {
		_ = level.Error(c.logger).Log("msg", "error stopping buffered client", "err", err.Error())
	}
	if err := c.closeQue(true); err != nil {
		_ = level.Error(c.logger).Log("msg", "error closing buffered client", "err", err.Error())
	}
	c.vali.StopWait()

	_ = level.Debug(c.logger).Log("msg", "client stopped")
}

// Handle implement EntryHandler; adds a new line to the next batch; send is async.
func (c *dqueClient) Handle(ls model.LabelSet, t time.Time, s string) error {
	// Here we don't need any synchronization because the worst thing is to
	// receive some more logs which would be dropped anyway.
	if c.isStooped {
		return nil
	}

	record := &dqueEntry{LabelSet: ls, Entry: logproto.Entry{Timestamp: t, Line: s}}
	if err := c.queue.Enqueue(record); err != nil {
		return fmt.Errorf("cannot enqueue record %s: %v", record.String(), err)
	}

	return nil
}

func (e *dqueEntry) String() string {
	return fmt.Sprintf("labels: %+v timestamp: %+v line: %+v", e.LabelSet, e.Entry.Timestamp, e.Entry.Line)
}

func (c *dqueClient) stopQue(wait bool) error {
	if !wait {
		return nil
	}
	c.lock.Lock()
	c.isStooped = true
	// In case the dequeuer is blocked on empty queue.
	if c.queue.Size() == 0 {
		c.lock.Unlock() //Nothing to wait for
		return nil
	}
	c.lock.Unlock()
	//TODO: Make time group waiter
	c.wg.Wait()
	return nil
}

func (c *dqueClient) closeQue(cleanUnderlyingFileBuffer bool) error {
	if err := c.queue.Close(); err != nil {
		return fmt.Errorf("cannot close %s buffer: %v", c.queue.Name, err)
	}

	if cleanUnderlyingFileBuffer {
		if err := os.RemoveAll(path.Join(c.queue.DirPath, c.queue.Name)); err != nil {
			return fmt.Errorf("cannot close %s buffer: %v", c.queue.Name, err)
		}
	}

	return nil
}
