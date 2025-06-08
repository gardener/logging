// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package curator

import (
	"fmt"
	"syscall"

	"github.com/go-kit/log/level"

	"github.com/gardener/logging/pkg/vali/curator/metrics"
)

// freeUpDiskCapacityIfNeeded checks the current disk usage and runs cleanup if needed
func (c *Curator) freeUpDiskCapacityIfNeeded() error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(c.config.DiskPath, &stat); err != nil {
		return err
	}

	// In bytes
	allCapacity := stat.Blocks * uint64(stat.Bsize)
	freeCapacity := stat.Bfree * uint64(stat.Bsize)
	freeCapacityPerc := int(freeCapacity * 100 / allCapacity)
	metrics.FreeStoragePercentages.Set(float64(freeCapacityPerc))

	_ = level.Debug(c.logger).Log("msg", "current storage free capacity", "percentages", freeCapacityPerc)
	if freeCapacityPerc < c.config.StorageConfig.MinFreePercentages {
		metrics.TriggeredStorageDeletion.Inc()
		_ = level.Info(c.logger).Log("msg", "storage cleanup started...")
		targetFreeCap := allCapacity / 100 * uint64(c.config.StorageConfig.TargetFreePercentages)
		_ = level.Debug(c.logger).Log("msg", "target free capacity", "bytes", targetFreeCap)

		currFreeSpaceFunc := func() (uint64, error) {
			var stat syscall.Statfs_t
			if err := syscall.Statfs(c.config.DiskPath, &stat); err != nil {
				return 0, err
			}

			return stat.Bfree * uint64(stat.Bsize), nil
		}

		pageSize := int(stat.Files/100) * c.config.StorageConfig.PageSizeForDeletionPercentages
		deletedCount, err := DeleteFiles(c.config.DiskPath, targetFreeCap, pageSize, currFreeSpaceFunc, c.logger)
		metrics.DeletedFilesDueToStorage.Add(float64(deletedCount))
		if err != nil {
			return fmt.Errorf("%s; Failed to clean the needed capacity. DeletedFiles: %d", err.Error(), deletedCount)
		}

		_ = level.Info(c.logger).Log("msg", "storage cleanup completed", "deleted chunks", deletedCount)
	}

	return nil
}
