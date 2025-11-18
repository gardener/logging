package client

import (
	"fmt"

	"github.com/go-kit/log"

	"github.com/gardener/logging/pkg/config"
)

// NewBufferDecorator makes a new buffered Client.
func NewBufferDecorator(cfg config.Config, logger log.Logger, newClientFunc NewClientFunc) (OutputClient, error) {
	switch cfg.ClientConfig.BufferConfig.BufferType {
	case "dque":
		return NewDque(cfg, logger, newClientFunc)
	default:
		return nil, fmt.Errorf("failed to parse bufferType: %s", cfg.ClientConfig.BufferConfig.BufferType)
	}
}
