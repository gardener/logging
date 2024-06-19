// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package curator

import (
	"runtime"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	config "github.com/gardener/logging/pkg/vali/curator/config"
	"github.com/gardener/logging/pkg/vali/curator/metrics"
	"github.com/gardener/logging/pkg/vali/curator/utils"
)

// Curator holds needed propperties for a curator
type Curator struct {
	closed chan struct{}
	ticker *time.Ticker
	config config.CuratorConfig
	logger log.Logger
}

// NewCurator creates new curator object
func NewCurator(conf config.CuratorConfig, logger log.Logger) *Curator {
	return &Curator{
		closed: make(chan struct{}),
		ticker: time.NewTicker(conf.TriggerInterval),
		config: conf,
		logger: logger,
	}
}

// Run the ticker
func (c *Curator) Run() {
	ms := utils.MemStat{}
	for {
		select {
		case <-c.closed:
			return
		case <-c.ticker.C:
			_ = level.Debug(c.logger).Log("mem_status", ms)
			c.curate()
			runtime.GC()
		}
	}
}

// Stop the ticker
func (c *Curator) Stop() {
	close(c.closed)
}

func (c *Curator) curate() {
	if err := c.freeUpDiskCapacityIfNeeded(); err != nil {
		_ = level.Error(c.logger).Log("msg", "Error in checking storage capacity", "error", err)
		metrics.Errors.WithLabelValues(metrics.ErrorWithDiskCurator).Inc()
	}

	if err := c.freeUpInodeCapacityIfNeeded(); err != nil {
		_ = level.Error(c.logger).Log("msg", "Error in checking Inodes capacity", "error", err)
		metrics.Errors.WithLabelValues(metrics.ErrorWithInodeCurator).Inc()
	}
}
