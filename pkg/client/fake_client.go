// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"errors"
	"sync"
	"time"

	"github.com/credativ/vali/pkg/logproto"
	"github.com/prometheus/common/model"
)

var _ OutputClient = &FakeValiClient{}

// FakeValiClient mocks OutputClient
type FakeValiClient struct {
	// IsStopped show whether the client is stopped or not
	IsStopped bool
	// IsGracefullyStopped show whether the client is gracefully topped or not
	IsGracefullyStopped bool
	// Entries is slice of all received entries
	Entries []Entry
	Mu      sync.Mutex
}

// GetEndPoint returns the target logging backend endpoint
func (*FakeValiClient) GetEndPoint() string {
	return "http://localhost"
}

// Handle processes and stores the received entries.
func (c *FakeValiClient) Handle(ls any, timestamp time.Time, line string) error {
	if c.IsStopped || c.IsGracefullyStopped {
		return errors.New("client has been stopped")
	}

	_ls, ok := ls.(model.LabelSet)
	if !ok {
		return ErrInvalidLabelType
	}

	c.Mu.Lock()
	c.Entries = append(c.Entries, Entry{
		Labels: _ls.Clone(),
		Entry:  logproto.Entry{Timestamp: timestamp, Line: line},
	})
	c.Mu.Unlock()

	return nil
}

// Stop stops the client
func (c *FakeValiClient) Stop() {
	c.IsStopped = true
}

// StopWait gracefully stops the client
func (c *FakeValiClient) StopWait() {
	c.IsGracefullyStopped = true
}
