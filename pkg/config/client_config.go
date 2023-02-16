/*
This file was copied from the grafana/loki project
https://github.com/grafana/loki/blob/v1.6.0/cmd/fluent-bit/config.go

Modifications Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
*/

package config

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/prometheus/common/model"

	"github.com/grafana/loki/pkg/logql"
	"github.com/grafana/loki/pkg/promtail/client"
	lokiflag "github.com/grafana/loki/pkg/util/flagext"
)

// ClientConfig holds configuration for the clients
type ClientConfig struct {
	// GrafanaLokiConfig holds the configuration for the grafana/loki client
	GrafanaLokiConfig client.Config
	// BufferConfig holds the configuration for the buffered client
	BufferConfig BufferConfig
	// SortByTimestamp indicates whether the logs should be sorted ot not
	SortByTimestamp bool
	// NumberOfBatchIDs is number of id per batch.
	// This increase the number of loki label streams
	NumberOfBatchIDs uint64
	// IdLabelName is the name of the batch id label key.
	IdLabelName model.LabelName
	// TestingClient is mocked grafana/loki client used for testing purposes
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
	QueueDir:         "/tmp/flb-storage/loki",
	QueueSegmentSize: 500,
	QueueSync:        false,
	QueueName:        "dque",
}

func initClientConfig(cfg Getter, res *Config) error {
	res.ClientConfig.GrafanaLokiConfig = DefaultClientCfg
	res.ClientConfig.BufferConfig = DefaultBufferConfig

	url := cfg.Get("URL")
	var clientURL flagext.URLValue
	if url == "" {
		url = "http://localhost:3100/loki/api/v1/push"
	}
	err := clientURL.Set(url)
	if err != nil {
		return errors.New("failed to parse client URL")
	}
	res.ClientConfig.GrafanaLokiConfig.URL = clientURL

	// cfg.Get will return empty string if not set, which is handled by the client library as no tenant
	res.ClientConfig.GrafanaLokiConfig.TenantID = cfg.Get("TenantID")

	batchWait := cfg.Get("BatchWait")
	if batchWait != "" {
		res.ClientConfig.GrafanaLokiConfig.BatchWait, err = time.ParseDuration(batchWait)
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
		res.ClientConfig.GrafanaLokiConfig.BatchSize = batchSizeValue
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
	res.ClientConfig.GrafanaLokiConfig.ExternalLabels = lokiflag.LabelSet{LabelSet: labelSet}

	maxRetries := cfg.Get("MaxRetries")
	if maxRetries != "" {
		res.ClientConfig.GrafanaLokiConfig.BackoffConfig.MaxRetries, err = strconv.Atoi(maxRetries)
		if err != nil {
			return fmt.Errorf("failed to parse MaxRetries: %s", maxRetries)
		}
	}

	timeout := cfg.Get("Timeout")
	if timeout != "" {
		res.ClientConfig.GrafanaLokiConfig.Timeout, err = time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("failed to parse Timeout: %s : %v", timeout, err)
		}
	}

	minBackoff := cfg.Get("MinBackoff")
	if minBackoff != "" {
		res.ClientConfig.GrafanaLokiConfig.BackoffConfig.MinBackoff, err = time.ParseDuration(minBackoff)
		if err != nil {
			return fmt.Errorf("failed to parse MinBackoff: %s : %v", minBackoff, err)
		}
	}

	maxBackoff := cfg.Get("MaxBackoff")
	if maxBackoff != "" {
		res.ClientConfig.GrafanaLokiConfig.BackoffConfig.MaxBackoff, err = time.ParseDuration(maxBackoff)
		if err != nil {
			return fmt.Errorf("failed to parse MaxBackoff: %s : %v", maxBackoff, err)
		}
	}

	// enable loki plugin buffering
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
