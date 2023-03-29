/*
This file was copied from the grafana/vali project
https://github.com/grafana/vali/blob/v1.6.0/cmd/fluent-bit/buffer.go

Modifications Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
*/
package buffer

import (
	"fmt"

	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/types"

	"github.com/go-kit/kit/log"
)

// NewBuffer makes a new buffered Client.
func NewBuffer(cfg config.Config, logger log.Logger, newClientFunc func(cfg config.Config, logger log.Logger) (types.LokiClient, error)) (types.LokiClient, error) {
	switch cfg.ClientConfig.BufferConfig.BufferType {
	case "dque":
		return NewDque(cfg, logger, newClientFunc)
	default:
		return nil, fmt.Errorf("failed to parse bufferType: %s", cfg.ClientConfig.BufferConfig.BufferType)
	}
}
