// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	namespace = "fluentbit_loki_gardener"

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
		Help:      "Total numbers of logs without metadata in the Loki Gardener",
	}, []string{"type"})

	// IncomingLogs is a prometheus metric which keeps the number of incoming logs
	IncomingLogs = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "incoming_logs_total",
		Help:      "Total number of incoming logs in the Loki Gardener",
	}, []string{"host"})

	// ForwardedLogs is a prometheus metric which keeps forwarded logs to the Promtail Client
	ForwardedLogs = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "forwarded_logs_total",
		Help:      "Total number of the forwarded logs to Promtail client",
	}, []string{"host"})

	// DroppedLogs is a prometheus metric which keeps the number of dropt logs by the output plugin
	DroppedLogs = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "dropped_logs_total",
		Help:      "Total number of dropped logs by the output plugin",
	}, []string{"host"})
)
