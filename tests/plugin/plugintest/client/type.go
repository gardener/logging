// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

// Client is an interface that defines the methods for running, shutting down, and getting logs count.
type Client interface {
	Run()
	Shutdown()
	GetLogsCount() int
}

// BlackBoxTestingValiClient is a struct that implements the EndClient interface.
type BlackBoxTestingValiClient struct{}
