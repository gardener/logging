// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package curator

import (
	"fmt"
	"syscall"

	"github.com/gardener/logging/pkg/vali/curator/metrics"
	"github.com/gardener/logging/pkg/vali/curator/utils"

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

	_ = level.Debug(c.logger).Log("msg", "current inode free capacity", "percentages", freeInodePercs)
	if freeInodePercs < c.config.InodeConfig.MinFreePercentages {
		metrics.TriggeredInodeDeletion.Inc()
		_ = level.Info(c.logger).Log("msg", "inodes cleanup started...")
		targetFreeInodes := stat.Files / 100 * uint64(c.config.InodeConfig.TargetFreePercentages)
		_ = level.Debug(c.logger).Log("msg", "target free inodes", "inodes", targetFreeInodes)

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

		_ = level.Info(c.logger).Log("msg", "inodes cleanup completed", "deleted chunks", deletedCount)
	}

	return nil
}
