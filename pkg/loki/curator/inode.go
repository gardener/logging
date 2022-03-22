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
	"fmt"
	"syscall"

	"github.com/gardener/logging/pkg/loki/curator/metrics"
	"github.com/gardener/logging/pkg/loki/curator/utils"

	"github.com/go-kit/kit/log/level"
)

// freeUpInodeCapacityIfNeeded checks the current inode usage and runs cleanup if needed
func (c *Curator) freeUpInodeCapacityIfNeeded() error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(c.config.DiskPath, &stat); err != nil {
		return err
	}

	freeInodePercs := int((stat.Ffree * 100) / stat.Files)
	metrics.FreeInodePercentages.Set(float64(freeInodePercs))

	level.Debug(c.logger).Log("msg", "current inode free capacity", "percentages", freeInodePercs)
	if freeInodePercs < c.config.InodeConfig.MinFreePercentages {
		metrics.TriggeredInodeDeletion.Inc()
		level.Info(c.logger).Log("msg", "inodes cleanup started...")
		targetFreeInodes := stat.Files / 100 * uint64(c.config.InodeConfig.TargetFreePercentages)
		level.Debug(c.logger).Log("msg", "target free inodes", "inodes", targetFreeInodes)

		currFreeSpaceFunc := func() (uint64, error) {
			var stat syscall.Statfs_t
			if err := syscall.Statfs(c.config.DiskPath, &stat); err != nil {
				return 0, err
			}

			return stat.Ffree, nil
		}

		pageSize := int(stat.Files/100) * c.config.InodeConfig.PageSizeForDeletionPercentages
		deletedCount, err := utils.DeleteFiles(c.config.DiskPath, targetFreeInodes, pageSize, currFreeSpaceFunc, c.logger)
		metrics.DeletedFilesDueToInodes.Add(float64(deletedCount))
		if err != nil {
			return fmt.Errorf("%s; Failed to clean the needed inodes. DeletedInodes: %d", err.Error(), deletedCount)
		}

		level.Info(c.logger).Log("msg", "inodes cleanup completed", "deleted chunks", deletedCount)
	}

	return nil
}
