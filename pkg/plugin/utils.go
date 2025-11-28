// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"
	"regexp"

	"github.com/gardener/logging/v1/pkg/types"
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
func extractKubernetesMetadataFromTag(records map[string]any, tagKey string, re *regexp.Regexp) error {
	tag, ok := records[tagKey].(string)
	if !ok {
		return fmt.Errorf("the tag entry for key %q is missing", tagKey)
	}

	kubernetesMetaData := re.FindStringSubmatch(tag)
	if len(kubernetesMetaData) != subExpressionNumber {
		return fmt.Errorf("invalid format for tag %v. The tag should be in format: %s", tag, re.String())
	}

	records["kubernetes"] = map[string]any{
		podName:       kubernetesMetaData[1],
		namespaceName: kubernetesMetaData[2],
		containerName: kubernetesMetaData[3],
		containerID:   kubernetesMetaData[4],
	}

	return nil
}

func getDynamicHostName(records types.OutputRecord, mapping map[string]any) string {
	for k, v := range mapping {
		switch nextKey := v.(type) {
		// if the next level is a map we are expecting we need to move deeper in the tree
		case map[string]any:
			// Try to get the nested value and convert it to OutputRecord
			if nextValue, ok := records[k].(types.OutputRecord); ok {
				return getDynamicHostName(nextValue, nextKey)
			}
			// Also try map[string]any directly since type aliases may not match in type assertions
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
