/*
This file was copied from the grafana/vali project
https://github.com/credativ/vali/blob/v1.6.0/cmd/fluent-bit/buffer.go

Modifications Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved.
*/
package client

import (
	"fmt"

	"github.com/gardener/logging/pkg/buffer"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/types"

	"github.com/go-kit/kit/log"
)

// NewBufferDecorator makes a new buffered Client.
func NewBufferDecorator(cfg config.Config, newClientFunc NewLokiClientFunc, logger log.Logger) (types.LokiClient, error) {
	switch cfg.ClientConfig.BufferConfig.BufferType {
	case "dque":
		return buffer.NewDque(cfg, logger, newClientFunc)
	default:
		return nil, fmt.Errorf("failed to parse bufferType: %s", cfg.ClientConfig.BufferConfig.BufferType)
	}
}
