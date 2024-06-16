// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/gardener/logging/pkg/metrics"
)

var (
	namespace = "vali_curator"

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
