// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"github.com/credativ/vali/pkg/valitail/api"
	"github.com/prometheus/common/model"
)

// EndClient is an interface that defines the methods for running, shutting down, and getting logs count.
type EndClient interface {
	Run()
	Shutdown()
	GetLogsCount(ls model.LabelSet) int
}

// BlackBoxTestingValiClient is a struct that implements the EndClient interface.
type BlackBoxTestingValiClient struct {
	entries         chan api.Entry
	receivedEntries []api.Entry
	localStreams    map[string]*localStream
	stopped         int
}
