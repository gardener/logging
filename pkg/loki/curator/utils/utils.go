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

package utils

import (
	"fmt"
	"os"
	"sort"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// DeleteFiles deletes a files until the free capacity reaches the target free capacity
func DeleteFiles(dirPath string, targetFreeSpace uint64, pageSize int, freeSpace func() (uint64, error), logger log.Logger) (int, error) {
	allDeletedFiles := 0
	currFreeSpace, err := freeSpace()
	if err != nil {
		return allDeletedFiles, err
	}
	level.Debug(logger).Log("msg", fmt.Sprintf("current free space: %d.", currFreeSpace))

	for currFreeSpace < targetFreeSpace {
		currDeletedFiles, err := deleteFilesCount(dirPath, pageSize)
		level.Debug(logger).Log("msg", fmt.Sprintf("current deleted files: %d.", currDeletedFiles))

		if err != nil {
			return allDeletedFiles, err
		}
		allDeletedFiles += currDeletedFiles

		if currFreeSpace, err = freeSpace(); err != nil {
			return allDeletedFiles, err
		}
		level.Debug(logger).Log("msg", fmt.Sprintf("current free space: %d.", currFreeSpace))
	}

	return allDeletedFiles, nil
}

func deleteFilesCount(dirPath string, count int) (int, error) {
	f, err := os.Open(dirPath)
	if err != nil {
		return 0, err
	}

	var initArray []os.FileInfo
	temp := make([]os.FileInfo, 0, count)
	listPage, err := f.Readdir(count)
	if err != nil {
		return 0, fmt.Errorf("failed to read dir %s, err: %v", dirPath, err)
	}
	if len(listPage) < count {
		return len(listPage), fmt.Errorf("found %d which is less than %d count", len(listPage), count)
	}

	initArray = listPage
	sort.Slice(initArray, func(i, j int) bool {
		return initArray[i].ModTime().Before(initArray[j].ModTime())
	})
	for listPage, err = f.Readdir(int(count)); err == nil; listPage, err = f.Readdir(int(count)) {
		sort.Slice(listPage, func(i, j int) bool {
			return listPage[i].ModTime().Before(listPage[j].ModTime())
		})
		var initArrayIndex int
		var listPageIndex int

		for initArrayIndex+listPageIndex < count {
			if listPageIndex >= len(listPage) || listPage[listPageIndex].ModTime().After(initArray[initArrayIndex].ModTime()) {
				temp = append(temp, initArray[initArrayIndex])
				initArrayIndex++
			} else {
				temp = append(temp, listPage[listPageIndex])
				listPageIndex++
			}
		}
		initArray, temp = temp, initArray[:0]
	}

	var deletedCount int
	for i := 0; i < len(initArray); i++ {
		if err = os.Remove(dirPath + "/" + initArray[i].Name()); err != nil {
			return deletedCount, err
		}
		deletedCount++
	}

	return deletedCount, nil
}
