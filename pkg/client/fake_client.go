// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package client

import (
	"fmt"
	"time"

	"github.com/grafana/vali/pkg/logproto"
	"github.com/prometheus/common/model"
)

// FakeLokiClient mocks LokiClient
type FakeLokiClient struct {
	// IsStopped show whether the client is stopped or not
	IsStopped bool
	// IsGracefullyStopped show whether the client is gracefully topped or not
	IsGracefullyStopped bool
	// Entries is slice of all received entries
	Entries []Entry
}

// Handle processes and stores the received entries.
func (c *FakeLokiClient) Handle(labels model.LabelSet, timestamp time.Time, line string) error {
	if c.IsStopped || c.IsGracefullyStopped {
		return fmt.Errorf("client has been stopped")
	}

	c.Entries = append(c.Entries, Entry{
		Labels: labels.Clone(),
		Entry:  logproto.Entry{Timestamp: timestamp, Line: line},
	})
	return nil
}

// Stop stops the client
func (c *FakeLokiClient) Stop() {
	c.IsStopped = true
}

// StopWait gracefully stops the client
func (c *FakeLokiClient) StopWait() {
	c.IsGracefullyStopped = true
}
