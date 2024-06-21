// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	namespace = "fluentbit_vali_gardener"

	// Errors is a prometheus which keeps total number of the errors
	Errors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "errors_total",
		Help:      "Total number of the errors",
	}, []string{"type"})

	// LogsWithoutMetadata is a prometheus metric which keeps the number of logs without metadata
	LogsWithoutMetadata = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "logs_without_metadata_total",
		Help:      "Total numbers of logs without metadata in the Vali Gardener",
	}, []string{"type"})

	// IncomingLogs is a prometheus metric which keeps the number of incoming logs
	IncomingLogs = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "incoming_logs_total",
		Help:      "Total number of incoming logs in the Vali Gardener",
	}, []string{"host"})

	// IncomingLogsWithEndpoint is a prometheus metric which keeps the number of incoming logs with endpoint
	IncomingLogsWithEndpoint = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "incoming_logs_with_endpoint_total",
		Help:      "Total number of incoming logs with endpoint in the Vali Gardener",
	}, []string{"host"})

	// ForwardedLogs is a prometheus metric which keeps forwarded logs to the Promtail Client
	ForwardedLogs = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "forwarded_logs_total",
		Help:      "Total number of the forwarded logs to Promtail client",
	}, []string{"host"})

	// DroppedLogs is a prometheus metric which keeps the number of dropped logs by the output plugin
	DroppedLogs = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "dropped_logs_total",
		Help:      "Total number of dropped logs by the output plugin",
	}, []string{"host"})
)
