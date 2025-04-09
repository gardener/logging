// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"io/ioutil"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/weaveworks/common/logging"

	"github.com/gardener/logging/pkg/vali/curator/utils"
)

var _ = ginkgov2.Describe("CuratorUtils", func() {

	var (
		numOfFiles = 10
		logLevel   logging.Level
	)

	ginkgov2.It("Test DeleteFiles", func() {
		files := []*os.File{}
		var tmpFile *os.File
		var err error
		for i := 0; i < numOfFiles; i++ {
			tmpFile, err = ioutil.TempFile(testDir, "temp-file")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
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

		_ = logLevel.Set("info")
		logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		logger = level.NewFilter(logger, logLevel.Gokit)
		deletedFiles, err := utils.DeleteFiles(testDir, uint64(numOfFiles/2), 1, freeSpaceFunc, logger)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(deletedFiles).To(gomega.Equal(numOfFiles / 2))
		newDeletedFiles, err := utils.DeleteFiles(testDir, uint64(numOfFiles-deletedFiles), 1, freeSpaceFunc, logger)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(newDeletedFiles).To(gomega.Equal(0))
	},
	)
})
