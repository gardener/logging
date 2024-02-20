/*
This file was copied from the grafana/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/config.go

Modifications Copyright SAP SE or an SAP affiliate company and Gardener contributors
*/

package config

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/credativ/vali/pkg/logql"
	valiflag "github.com/credativ/vali/pkg/util/flagext"
	"github.com/credativ/vali/pkg/valitail/client"
	"github.com/prometheus/common/model"
)

// ClientConfig holds configuration for the clients
type ClientConfig struct {
	// CredativValiConfig holds the configuration for the grafana/vali client
	CredativValiConfig client.Config
	// BufferConfig holds the configuration for the buffered client
	BufferConfig BufferConfig
	// SortByTimestamp indicates whether the logs should be sorted ot not
	SortByTimestamp bool
	// NumberOfBatchIDs is number of id per batch.
	// This increase the number of vali label streams
	NumberOfBatchIDs uint64
	// IdLabelName is the name of the batch id label key.
	IdLabelName model.LabelName
	// TestingClient is mocked grafana/vali client used for testing purposes
	TestingClient client.Client
}

// BufferConfig contains the buffer settings
type BufferConfig struct {
	Buffer     bool
	BufferType string
	DqueConfig DqueConfig
}

// DqueConfig contains the dqueue settings
type DqueConfig struct {
	QueueDir         string
	QueueSegmentSize int
	QueueSync        bool
	QueueName        string
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

func initClientConfig(cfg Getter, res *Config) error {
	res.ClientConfig.CredativValiConfig = DefaultClientCfg
	res.ClientConfig.BufferConfig = DefaultBufferConfig

	url := cfg.Get("URL")
	var clientURL flagext.URLValue
	if url == "" {
		url = "http://localhost:3100/vali/api/v1/push"
	}
	err := clientURL.Set(url)
	if err != nil {
		return errors.New("failed to parse client URL")
	}
	res.ClientConfig.CredativValiConfig.URL = clientURL

	// cfg.Get will return empty string if not set, which is handled by the client library as no tenant
	res.ClientConfig.CredativValiConfig.TenantID = cfg.Get("TenantID")

	batchWait := cfg.Get("BatchWait")
	if batchWait != "" {
		res.ClientConfig.CredativValiConfig.BatchWait, err = time.ParseDuration(batchWait)
		if err != nil {
			return fmt.Errorf("failed to parse BatchWait: %s :%v", batchWait, err)
		}
	}

	batchSize := cfg.Get("BatchSize")
	if batchSize != "" {
		batchSizeValue, err := strconv.Atoi(batchSize)
		if err != nil {
			return fmt.Errorf("failed to parse BatchSize: %s", batchSize)
		}
		res.ClientConfig.CredativValiConfig.BatchSize = batchSizeValue
	}

	labels := cfg.Get("Labels")
	if labels == "" {
		labels = `{job="fluent-bit"}`
	}
	matchers, err := logql.ParseMatchers(labels)
	if err != nil {
		return err
	}
	labelSet := make(model.LabelSet)
	for _, m := range matchers {
		labelSet[model.LabelName(m.Name)] = model.LabelValue(m.Value)
	}
	res.ClientConfig.CredativValiConfig.ExternalLabels = valiflag.LabelSet{LabelSet: labelSet}

	maxRetries := cfg.Get("MaxRetries")
	if maxRetries != "" {
		res.ClientConfig.CredativValiConfig.BackoffConfig.MaxRetries, err = strconv.Atoi(maxRetries)
		if err != nil {
			return fmt.Errorf("failed to parse MaxRetries: %s", maxRetries)
		}
	}

	timeout := cfg.Get("Timeout")
	if timeout != "" {
		res.ClientConfig.CredativValiConfig.Timeout, err = time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("failed to parse Timeout: %s : %v", timeout, err)
		}
	}

	minBackoff := cfg.Get("MinBackoff")
	if minBackoff != "" {
		res.ClientConfig.CredativValiConfig.BackoffConfig.MinBackoff, err = time.ParseDuration(minBackoff)
		if err != nil {
			return fmt.Errorf("failed to parse MinBackoff: %s : %v", minBackoff, err)
		}
	}

	maxBackoff := cfg.Get("MaxBackoff")
	if maxBackoff != "" {
		res.ClientConfig.CredativValiConfig.BackoffConfig.MaxBackoff, err = time.ParseDuration(maxBackoff)
		if err != nil {
			return fmt.Errorf("failed to parse MaxBackoff: %s : %v", maxBackoff, err)
		}
	}

	// enable vali plugin buffering
	buffer := cfg.Get("Buffer")
	if buffer != "" {
		res.ClientConfig.BufferConfig.Buffer, err = strconv.ParseBool(buffer)
		if err != nil {
			return fmt.Errorf("invalid value for Buffer, error: %v", err)
		}
	}

	// buffering type
	bufferType := cfg.Get("BufferType")
	if bufferType != "" {
		res.ClientConfig.BufferConfig.BufferType = bufferType
	}

	// dque directory
	queueDir := cfg.Get("QueueDir")
	if queueDir != "" {
		res.ClientConfig.BufferConfig.DqueConfig.QueueDir = queueDir
	}

	// dque segment size (queueEntry unit)
	queueSegmentSize := cfg.Get("QueueSegmentSize")
	if queueSegmentSize != "" {
		res.ClientConfig.BufferConfig.DqueConfig.QueueSegmentSize, err = strconv.Atoi(queueSegmentSize)
		if err != nil {
			return fmt.Errorf("cannot convert QueueSegmentSize %v to integer, error: %v", queueSegmentSize, err)
		}
	}

	// queueSync control file change sync to disk as they happen aka dque.turbo mode
	queueSync := cfg.Get("QueueSync")
	switch queueSync {
	case "normal", "":
		res.ClientConfig.BufferConfig.DqueConfig.QueueSync = false
	case "full":
		res.ClientConfig.BufferConfig.DqueConfig.QueueSync = true
	default:
		return fmt.Errorf("invalid string queueSync: %v", queueSync)
	}

	queueName := cfg.Get("QueueName")
	if queueName != "" {
		res.ClientConfig.BufferConfig.DqueConfig.QueueName = queueName
	}

	sortByTimestamp := cfg.Get("SortByTimestamp")
	if sortByTimestamp != "" {
		res.ClientConfig.SortByTimestamp, err = strconv.ParseBool(sortByTimestamp)
		if err != nil {
			return fmt.Errorf("invalid string SortByTimestamp: %v", err)
		}
	}

	numberOfBatchIDs := cfg.Get("NumberOfBatchIDs")
	if numberOfBatchIDs != "" {
		numberOfBatchIDsValue, err := strconv.Atoi(numberOfBatchIDs)
		if err != nil {
			return fmt.Errorf("failed to parse NumberOfBatchIDs: %s", numberOfBatchIDs)
		}
		if numberOfBatchIDsValue <= 0 {
			return fmt.Errorf("NumberOfBatchIDs can't be zero or negative value: %s", numberOfBatchIDs)
		} else {
			res.ClientConfig.NumberOfBatchIDs = uint64(numberOfBatchIDsValue)
		}
	} else {
		res.ClientConfig.NumberOfBatchIDs = 10
	}

	idLabelNameStr := cfg.Get("IdLabelName")
	if idLabelNameStr == "" {
		idLabelNameStr = "id"
	}
	idLabelName := model.LabelName(idLabelNameStr)
	if !idLabelName.IsValid() {
		return fmt.Errorf("invalid IdLabelName: %s", idLabelNameStr)
	}
	res.ClientConfig.IdLabelName = idLabelName

	return nil
}
