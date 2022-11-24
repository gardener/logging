/*
This file was copied from the grafana/loki project
https://github.com/grafana/loki/blob/v1.6.0/cmd/fluent-bit/config.go

Modifications Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
*/

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/prometheus/common/model"
	"github.com/weaveworks/common/logging"

	"github.com/grafana/loki/pkg/logql"
	"github.com/grafana/loki/pkg/promtail/client"
	lokiflag "github.com/grafana/loki/pkg/util/flagext"
)

// DefaultClientCfg is the default gardener lokiplugin client configuration.
var DefaultClientCfg = client.Config{}

func init() {
	// Init everything with default values.
	flagext.RegisterFlags(&DefaultClientCfg)
}

// Getter get a configuration settings base on the passed key.
type Getter interface {
	Get(key string) string
}

// Format is the log line format
type Format int

const (
	// JSONFormat represents json format for log line
	JSONFormat Format = iota
	// KvPairFormat represents key-value format for log line
	KvPairFormat
	// DefaultKubernetesMetadataTagExpression for extracting the kubernetes metadata from tag
	DefaultKubernetesMetadataTagExpression = "\\.([^_]+)_([^_]+)_(.+)-([a-z0-9]{64})\\.log$"

	// DefaultKubernetesMetadataTagKey represents the key for the tag in the entry
	DefaultKubernetesMetadataTagKey = "tag"

	// DefaultKubernetesMetadataTagPrefix represents the prefix of the entry's tag
	DefaultKubernetesMetadataTagPrefix = "kubernetes\\.var\\.log\\.containers"
)

// Config holds all of the needet properties of the loki output plugin
type Config struct {
	ClientConfig     ClientConfig
	ControllerConfig ControllerConfig
	PluginConfig     PluginConfig
	LogLevel         logging.Level
	Pprof            bool
}

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

// ControllerConfig hold the configuration fot the Loki client controller
type ControllerConfig struct {
	// CtlSyncTimeout for resource synchronization
	CtlSyncTimeout time.Duration
	// DynamicHostPrefix is the prefix of the dynamic host endpoint
	DynamicHostPrefix string
	// DynamicHostSuffix is the suffix of the dynamic host endpoint
	DynamicHostSuffix string
	// DeletedClientTimeExpiration is the time after a client for
	// deleted shoot should be cosidered for removal
	DeletedClientTimeExpiration time.Duration
	// MainControllerClientConfig configure to whether to send or not the log to the shoot
	// Loki for a particular shoot state.
	MainControllerClientConfig ControllerClientConfiguration
	// DefaultControllerClientConfig configure to whether to send or not the log to the shoot
	// Loki for a particular shoot state.
	DefaultControllerClientConfig ControllerClientConfiguration
}

// ControllerClientConfiguration contains flags which
// mutes/unmutes Shoot's and Seed Loki for a given Shoot state.
type ControllerClientConfiguration struct {
	SendLogsWhenIsInCreationState    bool
	SendLogsWhenIsInReadyState       bool
	SendLogsWhenIsInHibernatingState bool
	SendLogsWhenIsInHibernatedState  bool
	SendLogsWhenIsInWakingState      bool
	SendLogsWhenIsInDeletionState    bool
	SendLogsWhenIsInDeletedState     bool
	SendLogsWhenIsInRestoreState     bool
	SendLogsWhenIsInMigrationState   bool
}

// PluginConfig contains the configuration mostly related to the Loki plugin.
type PluginConfig struct {
	// AutoKubernetesLabels extact all key/values from the kubernetes field.
	AutoKubernetesLabels bool
	// RemoveKeys specify removing keys.
	RemoveKeys []string
	// LabelKeys is comma separated list of keys to use as stream labels.
	LabelKeys []string
	// LineFormat is the format to use when flattening the record to a log line.
	LineFormat Format
	// DropSingleKey if set to true and after extracting label_keys a record only
	// has a single key remaining, the log line sent to Loki will just be
	// the value of the record key.
	DropSingleKey bool
	// LabelMap is path to a json file defining how to transform nested records.
	LabelMap map[string]interface{}
	// DynamicHostPath is jsonpath in the log labels to the dynamic host.
	DynamicHostPath map[string]interface{}
	// DynamicHostRegex is regex to check if the dynamic host is valid.
	DynamicHostRegex string
	// KubernetesMetadata holds the configurations for retrieving the meta data from a tag.
	KubernetesMetadata KubernetesMetadataExtraction
	//DynamicTenant holds the configurations for retrieving the tenant from a record key.
	DynamicTenant DynamicTenant
	//LabelSetInitCapacity the initial capacity of the labelset stream.
	LabelSetInitCapacity int
	//HostnameKey is the key name of the hostname key/value pair.
	HostnameKey *string
	//HostnameValue is the value name of the hostname key/value pair.
	HostnameValue *string
	//PreservedLabels is the set of label which will be preserved after packing the handled logs.
	PreservedLabels model.LabelSet
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

// KubernetesMetadataExtraction holds the configurations for retrieving the meta data from a tag
type KubernetesMetadataExtraction struct {
	FallbackToTagWhenMetadataIsMissing bool
	DropLogEntryWithoutK8sMetadata     bool
	TagKey                             string
	TagPrefix                          string
	TagExpression                      string
}

// DynamicTenant contains specs for the lokiplugin dynamic functionality
type DynamicTenant struct {
	Tenant                                string
	Field                                 string
	Regex                                 string
	RemoveTenantIdWhenSendingToDefaultURL bool
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

// DefaultControllerClientConfig is the default controller client configuration
var DefaultControllerClientConfig = ControllerClientConfiguration{
	SendLogsWhenIsInCreationState:    true,
	SendLogsWhenIsInReadyState:       false,
	SendLogsWhenIsInHibernatingState: false,
	SendLogsWhenIsInHibernatedState:  false,
	SendLogsWhenIsInWakingState:      false,
	SendLogsWhenIsInDeletionState:    true,
	SendLogsWhenIsInDeletedState:     true,
	SendLogsWhenIsInRestoreState:     true,
	SendLogsWhenIsInMigrationState:   true,
}

// MainControllerClientConfig is the main controller client configuration
var MainControllerClientConfig = ControllerClientConfiguration{
	SendLogsWhenIsInCreationState:    true,
	SendLogsWhenIsInReadyState:       true,
	SendLogsWhenIsInHibernatingState: false,
	SendLogsWhenIsInHibernatedState:  false,
	SendLogsWhenIsInWakingState:      true,
	SendLogsWhenIsInDeletionState:    true,
	SendLogsWhenIsInDeletedState:     true,
	SendLogsWhenIsInRestoreState:     true,
	SendLogsWhenIsInMigrationState:   true,
}

// ParseConfig parse a Loki plugin configuration
func ParseConfig(cfg Getter) (*Config, error) {
	var err error
	res := &Config{}

	logLevel := cfg.Get("LogLevel")
	if logLevel == "" {
		logLevel = "info"
	}
	var level logging.Level
	if err := level.Set(logLevel); err != nil {
		return nil, err
	}
	res.LogLevel = level

	pprof := cfg.Get("Pprof")
	if pprof != "" {
		res.Pprof, err = strconv.ParseBool(pprof)
		if err != nil {
			return nil, fmt.Errorf("invalid value for Pprof, error: %v", err)
		}
	}

	if err := initClientConfig(cfg, res); err != nil {
		return nil, err
	}
	if err := initControllerConfig(cfg, res); err != nil {
		return nil, err
	}
	if err := initPluginConfig(cfg, res); err != nil {
		return nil, err
	}

	return res, nil
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

func initControllerConfig(cfg Getter, res *Config) error {
	var err error
	ctlSyncTimeout := cfg.Get("ControllerSyncTimeout")
	if ctlSyncTimeout != "" {
		res.ControllerConfig.CtlSyncTimeout, err = time.ParseDuration(ctlSyncTimeout)
		if err != nil {
			return fmt.Errorf("failed to parse ControllerSyncTimeout: %s : %v", ctlSyncTimeout, err)
		}
	} else {
		res.ControllerConfig.CtlSyncTimeout = 60 * time.Second
	}

	res.ControllerConfig.DynamicHostPrefix = cfg.Get("DynamicHostPrefix")
	res.ControllerConfig.DynamicHostSuffix = cfg.Get("DynamicHostSuffix")

	deletedClientTimeExpiration := cfg.Get("DeletedClientTimeExpiration")
	if deletedClientTimeExpiration != "" {
		res.ControllerConfig.DeletedClientTimeExpiration, err = time.ParseDuration(deletedClientTimeExpiration)
		if err != nil {
			return fmt.Errorf("failed to parse DeletedClientTimeExpiration: %s", deletedClientTimeExpiration)
		}
	} else {
		res.ControllerConfig.DeletedClientTimeExpiration = time.Hour
	}

	return initControllerClientConfig(cfg, res)
}

func initPluginConfig(cfg Getter, res *Config) error {
	var err error
	autoKubernetesLabels := cfg.Get("AutoKubernetesLabels")
	if autoKubernetesLabels != "" {
		res.PluginConfig.AutoKubernetesLabels, err = strconv.ParseBool(autoKubernetesLabels)
		if err != nil {
			return fmt.Errorf("invalid boolean for AutoKubernetesLabels, error: %v", err)
		}
	}

	dropSingleKey := cfg.Get("DropSingleKey")
	if dropSingleKey != "" {
		res.PluginConfig.DropSingleKey, err = strconv.ParseBool(dropSingleKey)
		if err != nil {
			return fmt.Errorf("invalid boolean DropSingleKey: %v", dropSingleKey)
		}
	} else {
		res.PluginConfig.DropSingleKey = true
	}

	removeKey := cfg.Get("RemoveKeys")
	if removeKey != "" {
		res.PluginConfig.RemoveKeys = strings.Split(removeKey, ",")
	}

	labelKeys := cfg.Get("LabelKeys")
	if labelKeys != "" {
		res.PluginConfig.LabelKeys = strings.Split(labelKeys, ",")
	}

	lineFormat := cfg.Get("LineFormat")
	switch lineFormat {
	case "json", "":
		res.PluginConfig.LineFormat = JSONFormat
	case "key_value":
		res.PluginConfig.LineFormat = KvPairFormat
	default:
		return fmt.Errorf("invalid format: %s", lineFormat)
	}

	labelMapPath := cfg.Get("LabelMapPath")
	if labelMapPath != "" {
		var content []byte
		if _, err := os.Stat(labelMapPath); err == nil {
			content, err = ioutil.ReadFile(labelMapPath)
			if err != nil {
				return fmt.Errorf("failed to open LabelMap file: %s", err)
			}
		} else if errors.Is(err, os.ErrNotExist) {
			content = []byte(labelMapPath)
		}
		if err := json.Unmarshal(content, &res.PluginConfig.LabelMap); err != nil {
			return fmt.Errorf("failed to Unmarshal LabelMap file: %s", err)
		}
		res.PluginConfig.LabelKeys = nil
	}

	dynamicHostPath := cfg.Get("DynamicHostPath")
	if dynamicHostPath != "" {
		if err := json.Unmarshal([]byte(dynamicHostPath), &res.PluginConfig.DynamicHostPath); err != nil {
			return fmt.Errorf("failed to Unmarshal DynamicHostPath json: %s", err)
		}
	}

	res.PluginConfig.DynamicHostRegex = cfg.Get("DynamicHostRegex")
	if res.PluginConfig.DynamicHostRegex == "" {
		res.PluginConfig.DynamicHostRegex = "*"
	}

	fallbackToTagWhenMetadataIsMissing := cfg.Get("FallbackToTagWhenMetadataIsMissing")
	if fallbackToTagWhenMetadataIsMissing != "" {
		res.PluginConfig.KubernetesMetadata.FallbackToTagWhenMetadataIsMissing, err = strconv.ParseBool(fallbackToTagWhenMetadataIsMissing)
		if err != nil {
			return fmt.Errorf("invalid value for FallbackToTagWhenMetadataIsMissing, error: %v", err)
		}
	}

	tagKey := cfg.Get("TagKey")
	if tagKey != "" {
		res.PluginConfig.KubernetesMetadata.TagKey = tagKey
	} else {
		res.PluginConfig.KubernetesMetadata.TagKey = DefaultKubernetesMetadataTagKey
	}

	tagPrefix := cfg.Get("TagPrefix")
	if tagPrefix != "" {
		res.PluginConfig.KubernetesMetadata.TagPrefix = tagPrefix
	} else {
		res.PluginConfig.KubernetesMetadata.TagPrefix = DefaultKubernetesMetadataTagPrefix
	}

	tagExpression := cfg.Get("TagExpression")
	if tagExpression != "" {
		res.PluginConfig.KubernetesMetadata.TagExpression = tagExpression
	} else {
		res.PluginConfig.KubernetesMetadata.TagExpression = DefaultKubernetesMetadataTagExpression
	}

	dropLogEntryWithoutK8sMetadata := cfg.Get("DropLogEntryWithoutK8sMetadata")
	if dropLogEntryWithoutK8sMetadata != "" {
		res.PluginConfig.KubernetesMetadata.DropLogEntryWithoutK8sMetadata, err = strconv.ParseBool(dropLogEntryWithoutK8sMetadata)
		if err != nil {
			return fmt.Errorf("invalid string DropLogEntryWithoutK8sMetadata: %v", err)
		}
	}

	dynamicTenant := cfg.Get("DynamicTenant")
	dynamicTenant = strings.Trim(dynamicTenant, " ")
	if dynamicTenant != "" {
		dynamicTenantValues := strings.SplitN(dynamicTenant, " ", 3)
		if len(dynamicTenantValues) != 3 {
			return fmt.Errorf("failed to parse DynamicTenant. Should consist of <tenant-name>\" \"<field-for-regex>\" \"<regex>. Found %d elements", len(dynamicTenantValues))
		}
		res.PluginConfig.DynamicTenant.Tenant = dynamicTenantValues[0]
		res.PluginConfig.DynamicTenant.Field = dynamicTenantValues[1]
		res.PluginConfig.DynamicTenant.Regex = dynamicTenantValues[2]
		removeTenantIdWhenSendingToDefaultURL := cfg.Get("RemoveTenantIdWhenSendingToDefaultURL")
		if removeTenantIdWhenSendingToDefaultURL != "" {
			res.PluginConfig.DynamicTenant.RemoveTenantIdWhenSendingToDefaultURL, err = strconv.ParseBool(removeTenantIdWhenSendingToDefaultURL)
			if err != nil {
				return fmt.Errorf("invalid value for RemoveTenantIdWhenSendingToDefaultURL, error: %v", err)
			}
		} else {
			res.PluginConfig.DynamicTenant.RemoveTenantIdWhenSendingToDefaultURL = true
		}
	}

	labelSetInitCapacity := cfg.Get("LabelSetInitCapacity")
	if labelSetInitCapacity != "" {
		labelSetInitCapacityValue, err := strconv.Atoi(labelSetInitCapacity)
		if err != nil {
			return fmt.Errorf("failed to parse LabelSetInitCapacity: %s", labelSetInitCapacity)
		}
		if labelSetInitCapacityValue <= 0 {
			return fmt.Errorf("LabelSetInitCapacity can't be zero or negative value: %s", labelSetInitCapacity)
		} else {
			res.PluginConfig.LabelSetInitCapacity = labelSetInitCapacityValue
		}
	} else {
		res.PluginConfig.LabelSetInitCapacity = 12
	}

	hostnameKeyValue := cfg.Get("HostnameKeyValue")
	if hostnameKeyValue != "" {
		hostnameKeyValueTokens := strings.SplitN(hostnameKeyValue, " ", 2)
		switch len(hostnameKeyValueTokens) {
		case 1:
			res.PluginConfig.HostnameKey = &hostnameKeyValueTokens[0]
		case 2:
			res.PluginConfig.HostnameKey = &hostnameKeyValueTokens[0]
			res.PluginConfig.HostnameValue = &hostnameKeyValueTokens[1]
		default:
			return fmt.Errorf("failed to parse HostnameKeyValue. Should consist of <hostname-key>\" \"<optional hostname-value>\". Found %d elements", len(hostnameKeyValueTokens))
		}
	}

	res.PluginConfig.PreservedLabels = model.LabelSet{}
	preservedLabelsStr := cfg.Get("PreservedLabels")
	if preservedLabelsStr != "" {
		for _, label := range strings.Split(preservedLabelsStr, ",") {
			res.PluginConfig.PreservedLabels[model.LabelName(strings.TrimSpace(label))] = ""
		}
	}

	return nil
}

func initControllerClientConfig(cfg Getter, res *Config) error {
	var err error

	res.ControllerConfig.MainControllerClientConfig = MainControllerClientConfig
	res.ControllerConfig.DefaultControllerClientConfig = DefaultControllerClientConfig

	sendLogsToMainClusterWhenIsInCreationState := cfg.Get("SendLogsToMainClusterWhenIsInCreationState")
	if sendLogsToMainClusterWhenIsInCreationState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInCreationState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInCreationState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInCreationState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInReadyState := cfg.Get("SendLogsToMainClusterWhenIsInReadyState")
	if sendLogsToMainClusterWhenIsInReadyState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInReadyState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInReadyState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInReadyState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInHibernatingState := cfg.Get("SendLogsToMainClusterWhenIsInHibernatingState")
	if sendLogsToMainClusterWhenIsInHibernatingState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInHibernatingState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInHibernatingState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInHibernatingState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInHibernatedState := cfg.Get("SendLogsToMainClusterWhenIsInHibernatedState")
	if sendLogsToMainClusterWhenIsInHibernatedState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInHibernatedState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInHibernatedState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInHibernatedState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInDeletionState := cfg.Get("SendLogsToMainClusterWhenIsInDeletionState")
	if sendLogsToMainClusterWhenIsInDeletionState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInDeletionState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInDeletionState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInDeletionState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInDeletedState := cfg.Get("SendLogsToMainClusterWhenIsInDeletedState")
	if sendLogsToMainClusterWhenIsInDeletedState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInDeletedState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInDeletedState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInDeletedState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInRestoreState := cfg.Get("SendLogsToMainClusterWhenIsInRestoreState")
	if sendLogsToMainClusterWhenIsInRestoreState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInRestoreState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInRestoreState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInRestoreState, error: %v", err)
		}
	}

	sendLogsToMainClusterWhenIsInMigrationState := cfg.Get("SendLogsToMainClusterWhenIsInMigrationState")
	if sendLogsToMainClusterWhenIsInMigrationState != "" {
		res.ControllerConfig.MainControllerClientConfig.SendLogsWhenIsInMigrationState, err = strconv.ParseBool(sendLogsToMainClusterWhenIsInMigrationState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToMainClusterWhenIsInMigrationState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInCreationState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInCreationState")
	if sendLogsToDefaultClientWhenClusterIsInCreationState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInCreationState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInCreationState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInCreationState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInReadyState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInReadyState")
	if sendLogsToDefaultClientWhenClusterIsInReadyState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInReadyState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInReadyState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInReadyState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInHibernatingState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInHibernatingState")
	if sendLogsToDefaultClientWhenClusterIsInHibernatingState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInHibernatingState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInHibernatingState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInHibernatingState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInHibernatedState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInHibernatedState")
	if sendLogsToDefaultClientWhenClusterIsInHibernatedState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInHibernatedState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInHibernatedState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInHibernatedState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInDeletionState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInDeletionState")
	if sendLogsToDefaultClientWhenClusterIsInDeletionState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInDeletionState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInDeletionState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInDeletionState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInDeletedState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInDeletedState")
	if sendLogsToDefaultClientWhenClusterIsInDeletedState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInDeletedState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInDeletedState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInDeletedState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInRestoreState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInRestoreState")
	if sendLogsToDefaultClientWhenClusterIsInRestoreState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInRestoreState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInRestoreState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInRestoreState, error: %v", err)
		}
	}

	sendLogsToDefaultClientWhenClusterIsInMigrationState := cfg.Get("SendLogsToDefaultClientWhenClusterIsInMigrationState")
	if sendLogsToDefaultClientWhenClusterIsInMigrationState != "" {
		res.ControllerConfig.DefaultControllerClientConfig.SendLogsWhenIsInMigrationState, err = strconv.ParseBool(sendLogsToDefaultClientWhenClusterIsInMigrationState)
		if err != nil {
			return fmt.Errorf("invalid value for SendLogsToDefaultClientWhenClusterIsInMigrationState, error: %v", err)
		}
	}

	return nil
}
