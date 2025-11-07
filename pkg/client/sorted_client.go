// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"sync"
	"time"

	"github.com/credativ/vali/pkg/logproto"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	giterrors "github.com/pkg/errors"
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/batch"
	"github.com/gardener/logging/pkg/config"
)

const componentNameSort = "sort"

type sortedClient struct {
	logger           log.Logger
	valiclient       OutputClient
	batch            *batch.Batch
	batchWait        time.Duration
	batchLock        sync.Mutex
	batchSize        int
	batchID          uint64
	numberOfBatchIDs uint64
	idLabelName      model.LabelName
	quit             chan struct{}
	entries          chan Entry
	wg               sync.WaitGroup
}

var _ OutputClient = &sortedClient{}

func (c *sortedClient) GetEndPoint() string {
	return c.valiclient.GetEndPoint()
}

// NewSortedClientDecorator returns client which sorts the logs based their timestamp.
func NewSortedClientDecorator(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (OutputClient, error) {
	var err error
	batchWait := cfg.ClientConfig.CredativValiConfig.BatchWait
	cfg.ClientConfig.CredativValiConfig.BatchWait = batchWait + (5 * time.Second)

	if logger == nil {
		logger = log.NewNopLogger()
	}

	client, err := newValiTailClient(cfg, newClient, logger)
	if err != nil {
		return nil, err
	}

	c := &sortedClient{
		logger:           log.With(logger, "component", componentNameSort, "host", cfg.ClientConfig.CredativValiConfig.URL.Host),
		valiclient:       client,
		batchWait:        batchWait,
		batchSize:        cfg.ClientConfig.CredativValiConfig.BatchSize,
		batchID:          0,
		numberOfBatchIDs: cfg.ClientConfig.NumberOfBatchIDs,
		batch:            batch.NewBatch(cfg.ClientConfig.IDLabelName, 0),
		idLabelName:      cfg.ClientConfig.IDLabelName,
		quit:             make(chan struct{}),
		entries:          make(chan Entry),
	}

	c.wg.Add(1)
	go c.run()
	_ = level.Debug(c.logger).Log("msg", "client started")

	return c, nil
}

func (c *sortedClient) run() {
	maxWaitCheckFrequency := c.batchWait / waitCheckFrequencyDelimiter
	if maxWaitCheckFrequency < minWaitCheckFrequency {
		maxWaitCheckFrequency = minWaitCheckFrequency
	}

	maxWaitCheck := time.NewTicker(maxWaitCheckFrequency)

	defer func() {
		maxWaitCheck.Stop()
		c.wg.Done()
	}()

	for {
		select {
		case <-c.quit:
			return

		case e := <-c.entries:

			// If the batch doesn't exist yet, we create a new one with the entry
			if c.batch == nil {
				c.newBatch(e)

				break
			}

			// If adding the entry to the batch will increase the size over the max
			// size allowed, we do send the current batch and then create a new one
			if c.batch.SizeBytesAfter(e.Line) > c.batchSize {
				c.sendBatch()
				c.newBatch(e)

				break
			}

			// The max size of the batch isn't reached, so we can add the entry
			c.addToBatch(e)

		case <-maxWaitCheck.C:
			// Send batche if max wait time has been reached

			if !c.isBatchWaitExceeded() {
				continue
			}

			c.sendBatch()
		}
	}
}

func (c *sortedClient) isBatchWaitExceeded() bool {
	c.batchLock.Lock()
	defer c.batchLock.Unlock()

	return c.batch != nil && c.batch.Age() > c.batchWait
}

func (c *sortedClient) sendBatch() {
	c.batchLock.Lock()
	defer c.batchLock.Unlock()

	if c.batch == nil {
		return
	}

	c.batch.Sort()

	for _, stream := range c.batch.GetStreams() {
		if err := c.handleEntries(stream.Labels, stream.Entries); err != nil {
			_ = level.Error(c.logger).Log("msg", "error sending stream", "stream", stream.Labels.String(), "error", err.Error())
		}
	}
	c.batch = nil
}

func (c *sortedClient) handleEntries(ls model.LabelSet, entries []batch.Entry) error {
	var combineErr error
	for _, entry := range entries {
		err := c.valiclient.Handle(ls, entry.Timestamp, entry.Line)
		if err != nil {
			combineErr = giterrors.Wrap(combineErr, err.Error())
		}
	}

	return combineErr
}

func (c *sortedClient) newBatch(e Entry) {
	c.batchLock.Lock()
	defer c.batchLock.Unlock()
	if c.batch == nil {
		c.batchID++
		c.batch = batch.NewBatch(c.idLabelName, c.batchID%c.numberOfBatchIDs)
	}

	c.batch.Add(e.Labels.Clone(), e.Timestamp, e.Line)
}

func (c *sortedClient) addToBatch(e Entry) {
	c.newBatch(e)
}

// Stop the client.
func (c *sortedClient) Stop() {
	close(c.quit)
	c.wg.Wait()
	c.valiclient.Stop()
	_ = level.Debug(c.logger).Log("msg", "client stopped without waiting")
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *sortedClient) StopWait() {
	close(c.quit)
	c.wg.Wait()
	if c.batch != nil {
		c.sendBatch()
	}
	c.valiclient.StopWait()
	_ = level.Debug(c.logger).Log("msg", "client stopped")
}

// Handle implement EntryHandler; adds a new line to the next batch; send is async.
func (c *sortedClient) Handle(ls any, t time.Time, s string) error {
	_ls, ok := ls.(model.LabelSet)
	if !ok {
		return ErrInvalidLabelType
	}
	c.entries <- Entry{_ls, logproto.Entry{
		Timestamp: t,
		Line:      s,
	}}

	return nil
}
