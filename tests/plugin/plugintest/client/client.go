// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

// NewBlackBoxTestingValiClient creates a new instance of BlackBoxTestingValiClient.
func NewBlackBoxTestingValiClient() *BlackBoxTestingValiClient {
	return &BlackBoxTestingValiClient{}
}

// Run starts the BlackBoxTestingValiClient and processes entries from the channel.
func (*BlackBoxTestingValiClient) Run() {

}

// Shutdown is used to close the entries channel.
func (*BlackBoxTestingValiClient) Shutdown() {

}

// GetLogsCount returns the count of logs for a given label set.
func (*BlackBoxTestingValiClient) GetLogsCount() int {
	return 0
}
