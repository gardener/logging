// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"io/ioutil"
	"time"

	. "github.com/gardener/logging/pkg/vali/curator/config"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var _ = Describe("CuratorConfig", func() {
	type testArgs struct {
		conf    map[string]interface{}
		want    *CuratorConfig
		wantErr bool
	}

	DescribeTable("Test CuratorConfig",
		func(args testArgs) {
			testConfigFile, err := ioutil.TempFile(testDir, "curator-config")
			Expect(err).ToNot(HaveOccurred())
			defer testConfigFile.Close()

			out, err := yaml.Marshal(args.conf)
			Expect(err).ToNot(HaveOccurred())
			_, err = testConfigFile.Write(out)
			Expect(err).ToNot(HaveOccurred())

			got, err := ParseConfigurations(testConfigFile.Name())
			if args.wantErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(args.want).To(Equal(got))
			}
		},
		Entry("default values", testArgs{
			map[string]interface{}{},
			&DefaultCuratorConfig,
			false},
		),
		Entry("overwrite values with the configuration ones", testArgs{
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

		Entry("bad TriggerInterval", testArgs{map[string]interface{}{"TriggerInterval": "0s"}, nil, true}),
		Entry("bad MinFreeInodesPercentages", testArgs{map[string]interface{}{
			"InodeConfig": map[string]interface{}{
				"MinFreePercentages": 101,
			}}, nil, true}),
		Entry("bad TargetFreeInodesPercentages", testArgs{map[string]interface{}{
			"InodeConfig": map[string]interface{}{
				"TargetFreePercentages": -1,
			}}, nil, true}),
		Entry("bad InodesPageSizeForDeletionPercentages", testArgs{map[string]interface{}{
			"InodeConfig": map[string]interface{}{
				"PageSizeForDeletionPercentages": 101,
			}}, nil, true}),
		Entry("bad MinFreeStoragePercentages", testArgs{map[string]interface{}{
			"StorageConfig": map[string]interface{}{
				"MinFreePercentages": -1,
			}}, nil, true}),
		Entry("bad TargetFreeStoragePercentages", testArgs{map[string]interface{}{
			"StorageConfig": map[string]interface{}{
				"TargetFreePercentages": 101,
			}}, nil, true}),
		Entry("bad CapacityPageSizeForDeletionPercentages", testArgs{map[string]interface{}{
			"StorageConfig": map[string]interface{}{
				"PageSizeForDeletionPercentages": -1,
			}}, nil, true}),
	)
})
