/*
This file was copied from the grafana/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/buffer.go

Modifications Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.
*/
package client

import (
	"fmt"
	"github.com/go-kit/kit/log"

	"github.com/gardener/logging/pkg/config"
)

// NewBuffer makes a new buffered Client.
func NewBuffer(cfg config.Config, logger log.Logger, newClientFunc func(cfg config.Config, logger log.Logger) (ValiClient, error)) (ValiClient, error) {
	switch cfg.ClientConfig.BufferConfig.BufferType {
	case "dque":
		return NewDque(cfg, logger, newClientFunc)
	default:
		return nil, fmt.Errorf("failed to parse bufferType: %s", cfg.ClientConfig.BufferConfig.BufferType)
	}
}
