// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package plugin // nolint:revive // var-naming the plugin package is the main entry point

import (
	"fmt"
	"regexp"
)

const (
	podName                           = "pod_name"
	namespaceName                     = "namespace_name"
	containerName                     = "container_name"
	containerID                       = "container_id"
	subExpressionNumber               = 5
	inCaseKubernetesMetadataIsMissing = 1
)

// extractKubernetesMetadataFromTag extracts kubernetes metadata from a tag and adds it to the records map.
// The tag should be in the format: pod_name.namespace_name.container_name.container_id
// This is required since the fluent-bit does not use the kubernetes filter plugin, reason for it is to avoid querying
// the kubernetes API server for the metadata.
func extractKubernetesMetadataFromTag(record map[string]any, tagKey string, re *regexp.Regexp) error {
	tag, ok := record[tagKey].(string)
	if !ok {
		// Collect available keys for debugging
		availableKeys := make([]string, 0, len(record))
		for k := range record {
			availableKeys = append(availableKeys, k)
		}

		return fmt.Errorf("the tag entry for key %q is missing or not a string, available keys: %v, value type: %T", tagKey, availableKeys, record[tagKey])
	}

	kubernetesMetaData := re.FindStringSubmatch(tag)
	if len(kubernetesMetaData) != subExpressionNumber {
		return fmt.Errorf("invalid format for tag %v. The tag should be in format: %s", tag, re.String())
	}

	record["kubernetes"] = map[string]any{
		podName:       kubernetesMetaData[1],
		namespaceName: kubernetesMetaData[2],
		containerName: kubernetesMetaData[3],
		containerID:   kubernetesMetaData[4],
	}

	return nil
}

func getDynamicHostName(records map[string]any, mapping map[string]any) string {
	for k, v := range mapping {
		//nolint:revive // enforce-switch-style: default-case is omitted on purpose since we are expecting only map[string]any or string as values in the mapping
		switch nextKey := v.(type) {
		// if the next level is a map we are expecting we need to move deeper in the tree
		case map[string]any:
			// FluentBit always sends map[string]any for nested structures
			if nextValue, ok := records[k].(map[string]any); ok {
				return getDynamicHostName(nextValue, nextKey)
			}
		case string:
			if value, ok := getRecordValue(k, records); ok {
				return value
			}
		}
	}

	return ""
}

func getRecordValue(key string, records map[string]any) (string, bool) {
	if value, ok := records[key]; ok {
		switch typedVal := value.(type) {
		case string:
			return typedVal, true
		case []byte:
			return string(typedVal), true
		default:
			return fmt.Sprintf("%v", typedVal), true
		}
	}

	return "", false
}
