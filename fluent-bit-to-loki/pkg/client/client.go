package client

import (
	"github.com/go-kit/kit/log"

	"github.com/gardener/logging/fluent-bit-to-loki/pkg/buffer"
	"github.com/gardener/logging/fluent-bit-to-loki/pkg/config"
	"github.com/grafana/loki/pkg/promtail/client"
)

// NewClient creates a new client based on the fluentbit configuration.
func NewClient(cfg *config.Config, logger log.Logger) (client.Client, error) {
	if cfg.BufferConfig.Buffer {
		return buffer.NewBuffer(cfg, logger)
	}
	return client.New(cfg.ClientConfig, logger)
}
