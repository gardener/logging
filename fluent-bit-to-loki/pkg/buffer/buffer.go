package buffer

import (
	"fmt"

	"github.com/gardener/logging/fluent-bit-to-loki/pkg/config"

	"github.com/go-kit/kit/log"
	"github.com/grafana/loki/pkg/promtail/client"
)

// NewBuffer makes a new buffered Client.
func NewBuffer(cfg *config.Config, logger log.Logger) (client.Client, error) {
	switch cfg.BufferConfig.BufferType {
	case "dque":
		return newDque(cfg, logger)
	default:
		return nil, fmt.Errorf("failed to parse bufferType: %s", cfg.BufferConfig.BufferType)
	}
}
