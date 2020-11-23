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
	outputPluginNS = "output_plugin"

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
)
