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

package utils_test

import (
	"io/ioutil"
	"os"

	"github.com/gardener/logging/pkg/loki/curator/utils"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/common/logging"
)

var _ = Describe("CuratorUtils", func() {

	var (
		numOfFiles int = 10
		logLevel   logging.Level
	)

	It("Test DeleteFiles", func() {
		files := []*os.File{}
		var tmpFile *os.File
		var err error
		for i := 0; i < numOfFiles; i++ {
			tmpFile, err = ioutil.TempFile(testDir, "temp-file")
			Expect(err).ToNot(HaveOccurred())
			files = append(files, tmpFile)
			defer tmpFile.Close()
		}

		freeSpaceFunc := func() (uint64, error) {
			currentFiles := 0
			for _, file := range files {
				if _, err := os.Stat(file.Name()); !os.IsNotExist(err) {
					currentFiles++
				}
			}

			return uint64(numOfFiles - currentFiles), nil
		}

		logLevel.Set("info")
		logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		logger = level.NewFilter(logger, logLevel.Gokit)
		deletedFiles, err := utils.DeleteFiles(testDir, uint64(numOfFiles/2), 1, freeSpaceFunc, logger)
		Expect(err).ToNot(HaveOccurred())
		Expect(deletedFiles).To(Equal(numOfFiles / 2))
		newDeletedFiles, err := utils.DeleteFiles(testDir, uint64(numOfFiles-deletedFiles), 1, freeSpaceFunc, logger)
		Expect(err).ToNot(HaveOccurred())
		Expect(newDeletedFiles).To(Equal(0))
	},
	)
})
