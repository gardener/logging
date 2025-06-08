// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package curator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type file struct {
	name    string
	modTime int64
}

// DeleteFiles deletes a files until the free capacity reaches the target free capacity
func DeleteFiles(dirPath string, targetFreeSpace uint64, pageSize int, freeSpace func() (uint64, error), logger log.Logger) (int, error) {
	allDeletedFiles := 0
	currentFreeSpace, err := freeSpace()
	if err != nil {
		return allDeletedFiles, err
	}
	_ = level.Debug(logger).Log("msg", "current free space", "bytes", currentFreeSpace)

	for currentFreeSpace < targetFreeSpace {
		currDeletedFiles, err := deleteNOldestFiles(dirPath, pageSize)
		_ = level.Debug(logger).Log("msg", "current deleted files", "count", currDeletedFiles)

		if err != nil {
			return allDeletedFiles, err
		}
		allDeletedFiles += currDeletedFiles

		if currentFreeSpace, err = freeSpace(); err != nil {
			return allDeletedFiles, err
		}
		_ = level.Debug(logger).Log("msg", "current free space", "bytes", currentFreeSpace)
	}

	return allDeletedFiles, nil
}

func deleteNOldestFiles(dirPath string, filePageSize int) (int, error) {
	var deletedFiles int
	oldestFiles, err := getNOldestFiles(dirPath, filePageSize)
	if err != nil {
		return deletedFiles, err
	}
	for _, fileForDeletion := range oldestFiles {
		if err = os.Remove(dirPath + "/" + fileForDeletion.name); err != nil {
			return deletedFiles, err
		}
		deletedFiles++
	}

	return deletedFiles, err
}

// GetNOldestFiles returns the N oldest files on success or empty file slice and an error.
func getNOldestFiles(dirPath string, filePageSize int) ([]file, error) {
	var filePage []file
	openedDir, err := os.Open(filepath.Clean(dirPath))
	if err != nil {
		return []file{}, err
	}

	defer func() { _ = openedDir.Close() }()

	oldestNFiles, err := getNextNFiles(nil, openedDir, filePageSize)
	if err != nil {
		return []file{}, err
	}

	sortFilesByModTime(oldestNFiles)

	tempBuffer := make([]file, 0, filePageSize)

	for filePage, err = getNextNFiles(nil, openedDir, filePageSize); err == nil && len(filePage) > 0; filePage, err = getNextNFiles(filePage[:0], openedDir, filePageSize) {
		sortFilesByModTime(filePage)
		var oldestNFilesIndex, filePageIndex int

		for oldestNFilesIndex+filePageIndex < filePageSize {
			if filePageIndex >= len(filePage) {
				tempBuffer = append(tempBuffer, oldestNFiles[oldestNFilesIndex:]...)

				break
			}
			if oldestNFilesIndex >= len(oldestNFiles) {
				tempBuffer = append(tempBuffer, filePage[filePageIndex:]...)

				break
			}

			if filePage[filePageIndex].modTime < oldestNFiles[oldestNFilesIndex].modTime {
				tempBuffer = append(tempBuffer, filePage[filePageIndex])
				filePageIndex++
			} else {
				tempBuffer = append(tempBuffer, oldestNFiles[oldestNFilesIndex])
				oldestNFilesIndex++
			}
		}
		oldestNFiles, tempBuffer = tempBuffer, oldestNFiles[:0]
	}

	return oldestNFiles, err
}

func getNextNFiles(nextNFiles []file, openedDir *os.File, count int) ([]file, error) {
	nextNFilesInfo, err := openedDir.Readdir(count)
	if err != nil && err != io.EOF {
		return []file{}, fmt.Errorf("failed to read dir %s, err: %v", openedDir.Name(), err)
	}
	if nextNFiles == nil {
		nextNFiles = make([]file, 0, len(nextNFilesInfo))
	}

	for _, fileInfo := range nextNFilesInfo {
		nextNFiles = append(nextNFiles, file{name: fileInfo.Name(), modTime: fileInfo.ModTime().Unix()})
	}

	return nextNFiles, nil
}

func sortFilesByModTime(files []file) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime < files[j].modTime
	})
}

// MemStat holds information about the current memory.
type MemStat struct {
	sb strings.Builder
}

func (ms MemStat) String() string {
	var m runtime.MemStats
	defer ms.sb.Reset()
	runtime.ReadMemStats(&m)
	_, _ = fmt.Fprintf(&ms.sb, "%+v\n", m)
	memoryWastedFragmentation := m.HeapInuse - m.HeapAlloc
	_, _ = fmt.Fprintf(&ms.sb, "Fragmentation Memory Waste: %d\n", memoryWastedFragmentation)
	memoryThatCouldBeReturnedToOS := m.HeapIdle - m.HeapReleased
	_, _ = fmt.Fprintf(&ms.sb, "Memory That Could Be Returned To OS: %d\n", memoryThatCouldBeReturnedToOS)

	return ms.sb.String()
}
