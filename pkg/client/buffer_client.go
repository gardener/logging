/*
This file was copied from the credativ/vali project
https://github.com/credativ/vali/blob/v2.2.4/cmd/fluent-bit/buffer.go

Modifications Copyright SAP SE or an SAP affiliate company and Gardener contributors
*/
package client

import (
	"fmt"

	"github.com/go-kit/log"

	"github.com/gardener/logging/pkg/config"
)

// NewBufferDecorator makes a new buffered Client.
func NewBufferDecorator(cfg config.Config, newClientFunc NewValiClientFunc, logger log.Logger) (OutputClient, error) {
	switch cfg.ClientConfig.BufferConfig.BufferType {
	case "dque":
		return NewDque(cfg, logger, newClientFunc)
	default:
		return nil, fmt.Errorf("failed to parse bufferType: %s", cfg.ClientConfig.BufferConfig.BufferType)
	}
}
