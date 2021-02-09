// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"github.com/gardener/logging/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	namespace = "loki_curator"

	// Errors is a prometheus metric which keeps total number of the errors
	Errors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "errors_total",
		Help:      "Total number of the errors",
	}, []string{"source"})

	// FreeStoragePercentages is a prometheus metric which keeps percentages of the free storage
	FreeStoragePercentages = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "free_storage_percentages",
		Help:      "Shows curret free storage percentages",
	})

	// FreeInodePercentages is a prometheus metric which keeps percentages of the free inodes
	FreeInodePercentages = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "free_inode_percentages",
		Help:      "Shows current free inode percentages",
	})

	// TriggeredStorageDeletion is a prometheus metric which keeps total triggered storage deletions
	TriggeredStorageDeletion = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "triggered_storage_deletion_total",
		Help:      "Shows the total triggers of the storage deletion logic",
	})

	// TriggeredInodeDeletion is a prometheus metric which keeps total triggered inode deletions
	TriggeredInodeDeletion = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "triggered_inode_deletion_total",
		Help:      "Shows the total triggers of the inode deletion logic",
	})

	// DeletedFilesDueToStorage is a prometheus metric which keeps total deleted files due to lack of storage
	DeletedFilesDueToStorage = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "deleted_files_due_to_storage_lack_total",
		Help:      "Shows the total amount of deleted storage percentages",
	})

	// DeletedFilesDueToInodes is a prometheus metric which keeps total deleted files due to lack of inodes
	DeletedFilesDueToInodes = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "deleted_files_due_to_inode_lack_total",
		Help:      "Shows the total amount of deleted inode percentages",
	})

	counters = []prometheus.Counter{
		FreeStoragePercentages, FreeInodePercentages, TriggeredStorageDeletion, TriggeredInodeDeletion, DeletedFilesDueToStorage, DeletedFilesDueToInodes,
	}
)

func init() {
	// Initialize counters to 0 so the metrics are exported before the first
	// occurrence of incrementing to avoid missing metrics.
	for _, counter := range counters {
		counter.Add(0)
	}
	for _, label := range []string{ErrorWithDiskCurator, ErrorWithInodeCurator} {
		metrics.Errors.WithLabelValues(label).Add(0)
	}
}
