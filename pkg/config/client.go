/*
This file was copied from the credativ/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/config.go

Modifications Copyright SAP SE or an SAP affiliate company and Gardener contributors
*/

package config

import (
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/credativ/vali/pkg/valitail/client"
	"github.com/prometheus/common/model"
)

// ClientConfig holds configuration for the chain of clients.
type ClientConfig struct {
	// URL for the Vali instance
	URL flagext.URLValue `mapstructure:"-"`
	// ProxyURL for proxy configuration
	ProxyURL string `mapstructure:"ProxyURL"`
	// TenantID for multi-tenant setups
	TenantID string `mapstructure:"TenantID"`
	// BatchWait time before sending a batch
	BatchWait string `mapstructure:"BatchWait"`
	// BatchSize maximum size of a batch
	BatchSize int `mapstructure:"BatchSize"`
	// Labels to attach to logs
	Labels string `mapstructure:"-"`
	// MaxRetries for failed requests
	MaxRetries int `mapstructure:"MaxRetries"`
	// Timeout for requests
	Timeout string `mapstructure:"Timeout"`
	// MinBackoff time for retries
	MinBackoff string `mapstructure:"MinBackoff"`
	// MaxBackoff time for retries
	MaxBackoff string `mapstructure:"MaxBackoff"`

	// CredativValiConfig holds the configuration for the credativ/vali client
	CredativValiConfig client.Config `mapstructure:"-"`
	// BufferConfig holds the configuration for the buffered client
	BufferConfig BufferConfig `mapstructure:",squash"`
	// SortByTimestamp indicates whether the logs should be sorted ot not
	SortByTimestamp bool `mapstructure:"SortByTimestamp"`
	// NumberOfBatchIDs is number of id per batch.
	// This increase the number of vali label streams
	NumberOfBatchIDs uint64 `mapstructure:"NumberOfBatchIDs"`
	// IDLabelName is the name of the batch id label key.
	IDLabelName model.LabelName `mapstructure:"IdLabelName"`
	// TestingClient is mocked credativ/vali client used for testing purposes
	TestingClient client.Client `mapstructure:"-"`
}

// BufferConfig contains the buffer settings
type BufferConfig struct {
	Buffer     bool       `mapstructure:"Buffer"`
	BufferType string     `mapstructure:"BufferType"`
	DqueConfig DqueConfig `mapstructure:",squash"`
}

// DqueConfig contains the dqueue settings
type DqueConfig struct {
	QueueDir         string `mapstructure:"QueueDir"`
	QueueSegmentSize int    `mapstructure:"QueueSegmentSize"`
	QueueSync        bool   `mapstructure:"-"` // Handled specially in postProcessConfig
	QueueName        string `mapstructure:"QueueName"`
}

// DefaultBufferConfig holds the configurations for using output buffer
var DefaultBufferConfig = BufferConfig{
	Buffer:     false,
	BufferType: "dque",
	DqueConfig: DefaultDqueConfig,
}

// DefaultDqueConfig holds dque configurations for the buffer
var DefaultDqueConfig = DqueConfig{
	QueueDir:         "/tmp/flb-storage/vali",
	QueueSegmentSize: 500,
	QueueSync:        false,
	QueueName:        "dque",
}
