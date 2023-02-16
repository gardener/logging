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

	"github.com/prometheus/common/model"
)

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
	//EnableMultiTenancy switch on and off the parsing of __gardener_multitenancy_id__ label
	EnableMultiTenancy bool
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

	enableMultiTenancy := cfg.Get("EnableMultiTenancy")
	if enableMultiTenancy != "" {
		res.PluginConfig.EnableMultiTenancy, err = strconv.ParseBool(enableMultiTenancy)
		if err != nil {
			return fmt.Errorf("invalid boolean EnableMultiTenancy: %v", enableMultiTenancy)
		}
	}

	return nil
}
