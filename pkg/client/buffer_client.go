/*
This file was copied from the grafana/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/buffer.go

Modifications Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved.
*/
package client

import (
	"fmt"

	"github.com/go-kit/kit/log"

	"github.com/gardener/logging/pkg/buffer"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/types"
)

// NewBufferDecorator makes a new buffered Client.
func NewBufferDecorator(cfg config.Config, newClientFunc NewValiClientFunc, logger log.Logger) (types.ValiClient, error) {
	switch cfg.ClientConfig.BufferConfig.BufferType {
	case "dque":
		return buffer.NewDque(cfg, logger, newClientFunc)
	default:
		return nil, fmt.Errorf("failed to parse bufferType: %s", cfg.ClientConfig.BufferConfig.BufferType)
	}
}
