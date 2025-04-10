// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"time"

	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	. "github.com/gardener/logging/pkg/vali/curator/config"
)

var _ = ginkgov2.Describe("CuratorConfig", func() {
	type testArgs struct {
		conf    map[string]interface{}
		want    *CuratorConfig
		wantErr bool
	}

	ginkgov2.DescribeTable("Test CuratorConfig",
		func(args testArgs) {
			testConfigFile, err := os.CreateTemp(testDir, "curator-config")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			defer func() { _ = testConfigFile.Close() }()

			out, err := yaml.Marshal(args.conf)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			_, err = testConfigFile.Write(out)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			got, err := ParseConfigurations(testConfigFile.Name())
			if args.wantErr {
				gomega.Expect(err).To(gomega.HaveOccurred())
			} else {
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(args.want).To(gomega.Equal(got))
			}
		},
		ginkgov2.Entry("default values", testArgs{
			map[string]interface{}{},
			&DefaultCuratorConfig,
			false},
		),
		ginkgov2.Entry("overwrite values with the configuration ones", testArgs{
			map[string]interface{}{
				"LogLevel":        "debug",
				"DiskPath":        "/test",
				"TriggerInterval": "1s",
				"InodeConfig": map[string]interface{}{
					"MinFreePercentages":             2,
					"TargetFreePercentages":          3,
					"PageSizeForDeletionPercentages": 4,
				},
				"StorageConfig": map[string]interface{}{
					"MinFreePercentages":             5,
					"TargetFreePercentages":          6,
					"PageSizeForDeletionPercentages": 7,
				},
			},
			&CuratorConfig{
				LogLevel:        "debug",
				DiskPath:        "/test",
				TriggerInterval: 1 * time.Second,
				InodeConfig: Config{
					MinFreePercentages:             2,
					TargetFreePercentages:          3,
					PageSizeForDeletionPercentages: 4,
				},
				StorageConfig: Config{
					MinFreePercentages:             5,
					TargetFreePercentages:          6,
					PageSizeForDeletionPercentages: 7,
				},
			},
			false},
		),

		ginkgov2.Entry("bad TriggerInterval", testArgs{map[string]interface{}{"TriggerInterval": "0s"}, nil, true}),
		ginkgov2.Entry("bad MinFreeInodesPercentages", testArgs{map[string]interface{}{
			"InodeConfig": map[string]interface{}{
				"MinFreePercentages": 101,
			}}, nil, true}),
		ginkgov2.Entry("bad TargetFreeInodesPercentages", testArgs{map[string]interface{}{
			"InodeConfig": map[string]interface{}{
				"TargetFreePercentages": -1,
			}}, nil, true}),
		ginkgov2.Entry("bad InodesPageSizeForDeletionPercentages", testArgs{map[string]interface{}{
			"InodeConfig": map[string]interface{}{
				"PageSizeForDeletionPercentages": 101,
			}}, nil, true}),
		ginkgov2.Entry("bad MinFreeStoragePercentages", testArgs{map[string]interface{}{
			"StorageConfig": map[string]interface{}{
				"MinFreePercentages": -1,
			}}, nil, true}),
		ginkgov2.Entry("bad TargetFreeStoragePercentages", testArgs{map[string]interface{}{
			"StorageConfig": map[string]interface{}{
				"TargetFreePercentages": 101,
			}}, nil, true}),
		ginkgov2.Entry("bad CapacityPageSizeForDeletionPercentages", testArgs{map[string]interface{}{
			"StorageConfig": map[string]interface{}{
				"PageSizeForDeletionPercentages": -1,
			}}, nil, true}),
	)
})
