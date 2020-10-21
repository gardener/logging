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
	areMetricsRegistered bool
	outputPluginNS       = "output_plugin"

	// ErrorsCount is a prometheus which keeps number of the errors
	ErrorsCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: outputPluginNS,
		Name:      "errors",
		Help:      "Number of the errors",
	}, []string{"type"})

	// MissingMetadataLogs is a prometheus metric which keeps the number of logs without metadata
	MissingMetadataLogs = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: outputPluginNS,
		Name:      "no_metadata",
		Help:      "Number of logs without metadata",
	}, []string{"type"})

	// IncomingLogs is a prometheus metric which keeps the number of incoming logs
	IncomingLogs = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: outputPluginNS,
		Name:      "incoming_logs",
		Help:      "Number of incoming logs",
	}, []string{"host"})

	// PastSendRequests is a prometheus metric which keeps past send requests
	PastSendRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: outputPluginNS,
		Name:      "past_send_requests",
		Help:      "Number of past send requests",
	}, []string{"host"})

	// DroptLogs is a prometheus metric which keeps the number of dropt logs by the output plugin
	DroptLogs = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: outputPluginNS,
		Name:      "dropt_logs",
		Help:      "Number of dropt logs by the output plugin",
	}, []string{"host"})

	// ErrorsMetric is a custom metric which keeps the errors count
	ErrorsMetric OneLabelMetric

	// MissingMetadataMetric is a custom metric which keeps missing metadata logs count
	MissingMetadataMetric OneLabelMetric

	// IncomingLogMetric is a custom metric which keeps the incoming logs count
	IncomingLogMetric OneLabelMetric

	// PastSendRequestMetric is a custom metric which keeps the past send requests count
	PastSendRequestMetric OneLabelMetric

	// DroptLogMetric is a custom metric which keeps dropt logs count
	DroptLogMetric OneLabelMetric

	oneLabelMetrics Metrics
)

// RegisterMetrics is a singletion function which registers the metrics
func RegisterMetrics(intervals int) {

	if !areMetricsRegistered {
		lock.RLock()
		if !areMetricsRegistered {
			ErrorsMetric = OneLabelMetric{countVec: ErrorsCount, oneLabelMetric: make(map[string]*oneLabelMetricWrapper), intervals: intervals}
			MissingMetadataMetric = OneLabelMetric{countVec: MissingMetadataLogs, oneLabelMetric: make(map[string]*oneLabelMetricWrapper), intervals: intervals}
			IncomingLogMetric = OneLabelMetric{countVec: IncomingLogs, oneLabelMetric: make(map[string]*oneLabelMetricWrapper), intervals: intervals}
			PastSendRequestMetric = OneLabelMetric{countVec: PastSendRequests, oneLabelMetric: make(map[string]*oneLabelMetricWrapper), intervals: intervals}
			DroptLogMetric = OneLabelMetric{countVec: DroptLogs, oneLabelMetric: make(map[string]*oneLabelMetricWrapper), intervals: intervals}

			oneLabelMetrics = Metrics{&ErrorsMetric, &MissingMetadataMetric, &IncomingLogMetric, &PastSendRequestMetric, &DroptLogMetric}
		}
		areMetricsRegistered = true
		lock.RUnlock()
	}
}

// Metrics is a slice with all custom metrics
type Metrics []Metric

// Update updates all metrics
func (m Metrics) Update() {
	for _, metric := range m {
		metric.Update()
	}
}

// Metric is an interface which all custom metrics should implement
type Metric interface {
	Update()
	Add(logsCount int, labels ...string) error
}
