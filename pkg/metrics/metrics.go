// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// FluentBitGardenerMetrics contains all prometheus metrics for the fluent-bit gardener output plugin.
type FluentBitGardenerMetrics struct {
	// Clients is a prometheus metric which keeps total number of the output clients
	Clients *prometheus.GaugeVec
	// Errors is a prometheus which keeps total number of the errors
	Errors *prometheus.CounterVec
	// LogsWithoutMetadata is a prometheus metric which keeps the number of logs without metadata
	LogsWithoutMetadata *prometheus.CounterVec
	// IncomingLogs is a prometheus metric which keeps the number of incoming logs
	IncomingLogs *prometheus.CounterVec
	// OutputClientLogs is a prometheus metric which keeps logs to the Output Client
	OutputClientLogs *prometheus.CounterVec
	// ExportedClientLogs is a prometheus metric which keeps logs to the Output Client
	ExportedClientLogs *prometheus.CounterVec
	// DroppedLogs is a prometheus metric which keeps the number of dropped logs by the output plugin
	DroppedLogs *prometheus.CounterVec
	// ThrottledLogs is a prometheus metric which keeps the number of throttled logs by the output plugin
	ThrottledLogs *prometheus.CounterVec
	// BufferedLogs is a prometheus metric which keeps the number of logs buffered in the batch processor queue
	BufferedLogs *prometheus.GaugeVec
	// DqueSize is a prometheus metric which keeps the current size of the dque queue
	DqueSize *prometheus.GaugeVec
}

// NewFluentBitGardenerMetrics creates and registers all fluent-bit gardener metrics with the given registerer.
func NewFluentBitGardenerMetrics(reg prometheus.Registerer) *FluentBitGardenerMetrics {
	namespace := "fluentbit_gardener"

	return &FluentBitGardenerMetrics{
		Clients: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "clients_total",
			Help:      "Total number of the output clients",
		}, []string{"type"}),
		Errors: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "errors_total",
			Help:      "Total number of the errors",
		}, []string{"type"}),
		LogsWithoutMetadata: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "logs_without_metadata_total",
			Help:      "Total numbers of logs without metadata in the gardener output plugin",
		}, []string{"type"}),
		IncomingLogs: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "incoming_logs_total",
			Help:      "Total number of incoming logs in the gardener output plugin",
		}, []string{"host"}),
		OutputClientLogs: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "output_client_logs_total",
			Help:      "Total number of the forwarded logs to the output client",
		}, []string{"host"}),
		ExportedClientLogs: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "exported_client_logs_total",
			Help:      "Total number of the exported logs to the output client",
		}, []string{"host"}),
		DroppedLogs: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "dropped_logs_total",
			Help:      "Total number of dropped logs by the output plugin",
		}, []string{"host", "reason"}),
		ThrottledLogs: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "throttled_logs_total",
			Help:      "Total number of throttled logs by the output plugin",
		}, []string{"host"}),
		BufferedLogs: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "buffered_logs",
			Help:      "Current number of logs buffered in the batch processor queue",
		}, []string{"host"}),
		DqueSize: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "dque_size",
			Help:      "Current size of the dque queue",
		}, []string{"name"}),
	}
}
