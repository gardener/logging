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

package client

import (
	"time"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/types"

	"github.com/go-kit/kit/log"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/prometheus/common/model"
)

// LokiClient represents an instance which sends logs to Loki ingester
type LokiClient interface {
	// Handle processes logs and then sends them to Loki ingester
	Handle(labels model.LabelSet, time time.Time, entry string) error
	// Stop shut down the client immediately without waiting to send the saved logs
	Stop()
	// StopWait stops the client of receiving new logs and waits all saved logs to be sent until shuting down
	StopWait()
}

type Entry struct {
	Labels model.LabelSet
	logproto.Entry
}

// NewLokiClientFunc returns a LokiClient on success.
type NewLokiClientFunc func(cfg config.Config, logger log.Logger) (types.LokiClient, error)

// NewLokiClientDecoratorFunc return LokiClient which wraps another LokiClient
type NewLokiClientDecoratorFunc func(cfg config.Config, client LokiClient, logger log.Logger) (types.LokiClient, error)
