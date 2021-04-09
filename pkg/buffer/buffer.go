/*
This file was copied from the grafana/loki project
https://github.com/grafana/loki/blob/v1.6.0/cmd/fluent-bit/buffer.go

Modifications Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
*/
package buffer

import (
	"fmt"

	"github.com/gardener/logging/pkg/config"

	"github.com/go-kit/kit/log"
	"github.com/grafana/loki/pkg/promtail/client"
)

// NewBuffer makes a new buffered Client.
func NewBuffer(cfg *config.Config, logger log.Logger, newClientFunc func(cfg client.Config, logger log.Logger) (client.Client, error)) (client.Client, error) {
	switch cfg.ClientConfig.BufferConfig.BufferType {
	case "dque":
		return newDque(cfg, logger, newClientFunc)
	default:
		return nil, fmt.Errorf("failed to parse bufferType: %s", cfg.ClientConfig.BufferConfig.BufferType)
	}
}
