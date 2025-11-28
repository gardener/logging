// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	"github.com/gardener/logging/v1/pkg/metrics"
	"github.com/gardener/logging/v1/pkg/types"
)

// toOutputRecord converts fluent-bit's map[any]any to types.OutputRecord.
// It recursively processes nested structures and converts byte arrays to strings.
// Entries with non-string keys are dropped and logged as warnings with metrics.
func toOutputRecord(record map[any]any) types.OutputRecord {
	m := make(types.OutputRecord, len(record))
	for k, v := range record {
		key, ok := k.(string)
		if !ok {
			logger.V(2).Info("dropping record entry with non-string key", "keyType", fmt.Sprintf("%T", k))
			metrics.Errors.WithLabelValues(metrics.ErrorInvalidRecordKey).Inc()

			continue
		}

		switch t := v.(type) {
		case []byte:
			m[key] = string(t)
		case map[any]any:
			m[key] = toOutputRecord(t)
		case []any:
			m[key] = toSlice(t)
		default:
			m[key] = v
		}
	}

	return m
}

// toSlice recursively converts []any, handling nested structures and byte arrays.
// It maintains the same conversion logic as toOutputRecord for consistency.
func toSlice(slice []any) []any {
	if len(slice) == 0 {
		return slice
	}

	s := make([]any, 0, len(slice))
	for _, v := range slice {
		switch t := v.(type) {
		case []byte:
			s = append(s, string(t))
		case map[any]any:
			s = append(s, toOutputRecord(t))
		case []any:
			s = append(s, toSlice(t))
		default:
			s = append(s, t)
		}
	}

	return s
}
