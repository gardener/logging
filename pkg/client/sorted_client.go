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
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/batch"
	"github.com/gardener/logging/pkg/config"
)

type sortedClient struct {
	logger           log.Logger
	valiclient       multiTenantClient
	batch            *batch.Batch
	batchWait        time.Duration
	batchLock        sync.Mutex
	batchSize        int
	batchID          uint64
	numberOfBatchIDs uint64
	idLabelName      model.LabelName
	quit             chan struct{}
	once             sync.Once
	entries          chan Entry
	wg               sync.WaitGroup
}

var _ ValiClient = &sortedClient{}

func (c *sortedClient) GetEndPoint() string {
	return c.valiclient.GetEndPoint()
}

// NewSortedClientDecorator returns client which sorts the logs based their timestamp.
func NewSortedClientDecorator(cfg config.Config, newClient NewValiClientFunc, logger log.Logger) (ValiClient, error) {
	var err error
	batchWait := cfg.ClientConfig.CredativValiConfig.BatchWait
	cfg.ClientConfig.CredativValiConfig.BatchWait = batchWait + (5 * time.Second)

	client, err := newValiClient(cfg, newClient, logger)
	if err != nil {
		return nil, err
	}

	c := &sortedClient{
		logger:           log.With(logger, "component", "client", "host", cfg.ClientConfig.CredativValiConfig.URL.Host),
		valiclient:       multiTenantClient{valiclient: client},
		batchWait:        batchWait,
		batchSize:        cfg.ClientConfig.CredativValiConfig.BatchSize,
		batchID:          0,
		numberOfBatchIDs: cfg.ClientConfig.NumberOfBatchIDs,
		batch:            batch.NewBatch(cfg.ClientConfig.IdLabelName, 0),
		idLabelName:      cfg.ClientConfig.IdLabelName,
		quit:             make(chan struct{}),
		entries:          make(chan Entry),
	}

	c.wg.Add(1)
	go c.run()
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
			if c.batch.SizeBytesAfter(e.Entry.Line) > c.batchSize {
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
		if err := c.valiclient.handleStream(*stream); err != nil {
			_ = level.Error(c.logger).Log("msg", "error sending stream", "stream", stream.Labels.String(), "error", err.Error())
		}
	}
	c.batch = nil
}

func (c *sortedClient) newBatch(e Entry) {
	c.batchLock.Lock()
	defer c.batchLock.Unlock()
	if c.batch == nil {
		c.batchID++
		c.batch = batch.NewBatch(c.idLabelName, c.batchID%c.numberOfBatchIDs)
	}

	c.batch.Add(e.Labels.Clone(), e.Entry.Timestamp, e.Entry.Line)
}

func (c *sortedClient) addToBatch(e Entry) {
	c.newBatch(e)
}

// Stop the client.
func (c *sortedClient) Stop() {
	c.once.Do(func() {
		close(c.quit)
		c.wg.Wait()
		c.valiclient.Stop()
	})

}

func (c *sortedClient) StopWait() {
	c.once.Do(func() {
		close(c.quit)
		c.wg.Wait()
		if c.batch != nil {
			c.sendBatch()
		}
		c.valiclient.StopWait()
	})

}

// Handle implement EntryHandler; adds a new line to the next batch; send is async.
func (c *sortedClient) Handle(ls model.LabelSet, t time.Time, s string) error {
	c.entries <- Entry{ls, logproto.Entry{
		Timestamp: t,
		Line:      s,
	}}
	return nil
}
