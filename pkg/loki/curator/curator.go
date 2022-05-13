// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package curator

import (
	"io/ioutil"
	"strconv"
	"time"

	config "github.com/gardener/logging/pkg/loki/curator/config"
	"github.com/gardener/logging/pkg/loki/curator/metrics"
	"github.com/gardener/logging/pkg/loki/curator/utils"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Curator holds needed propperties for a curator
type Curator struct {
	closed           chan struct{}
	curateTicker     *time.Ticker
	cleanCacheTicker *time.Ticker
	config           config.CuratorConfig
	logger           log.Logger
}

// NewCurator creates new curator object
func NewCurator(conf config.CuratorConfig, logger log.Logger) *Curator {
	return &Curator{
		closed:           make(chan struct{}),
		curateTicker:     time.NewTicker(conf.TriggerInterval),
		cleanCacheTicker: time.NewTicker(conf.DropCacheConfig.TriggerInterval),
		config:           conf,
		logger:           logger,
	}
}

// Run the ticker
func (c *Curator) Run() {
	ms := utils.MemStat{}
	for {
		select {
		case <-c.closed:
			return
		case <-c.curateTicker.C:
			_ = level.Debug(c.logger).Log("mem_status", ms)
			c.curate()
		case <-c.cleanCacheTicker.C:
			if c.config.DropCacheConfig.Enabled {
				c.cleanCache()
			}
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

func (c *Curator) cleanCache() {
	_ = level.Info(c.logger).Log("msg", "cache cleanup started", "filePath", c.config.DropCacheConfig.DropCacheFilePath, "dropOption", c.config.DropCacheConfig.ResetCacheOption)
	err := ioutil.WriteFile(c.config.DropCacheConfig.DropCacheFilePath, []byte(strconv.Itoa(c.config.DropCacheConfig.ResetCacheOption)), 0644)
	if err != nil {
		_ = level.Error(c.logger).Log("msg", "Error in cache cleaning", "error", err)
	}
	_ = level.Info(c.logger).Log("msg", "cache cleanup completed")
}
