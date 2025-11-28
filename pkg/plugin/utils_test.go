// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/logging/v1/pkg/config"
	"github.com/gardener/logging/v1/pkg/types"
)

type getDynamicHostNameArgs struct {
	records map[string]any
	mapping map[string]any
	want    string
}

type fallbackToTagWhenMetadataIsMissing struct {
	records   map[string]any
	tagKey    string
	tagPrefix string
	tagRegexp string
	want      map[string]any
	err       error
}

var _ = Describe("OutputPlugin plugin utils", func() {
	DescribeTable("#getDynamicHostName",
		func(args getDynamicHostNameArgs) {
			got := getDynamicHostName(args.records, args.mapping)
			Expect(got).To(Equal(args.want))
		},
		Entry("empty record",
			getDynamicHostNameArgs{
				records: types.OutputRecord{},
				mapping: map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": "namespace",
					},
				},
				want: "",
			},
		),
		Entry("empty mapping",
			getDynamicHostNameArgs{
				records: types.OutputRecord{
					"kubernetes": map[string]any{
						"foo":            []byte("buzz"),
						"namespace_name": []byte("garden"),
					},
				},
				mapping: map[string]any{},
				want:    "",
			},
		),
		Entry("empty subrecord",
			getDynamicHostNameArgs{
				records: types.OutputRecord{
					"kubernetes": map[string]any{
						"foo": []byte("buzz"),
					},
				},
				mapping: map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": "namespace",
					},
				},
				want: "",
			},
		),
		Entry("subrecord",
			getDynamicHostNameArgs{
				records: types.OutputRecord{
					"kubernetes": map[string]any{
						"foo":            []byte("buzz"),
						"namespace_name": []byte("garden"),
					},
				},
				mapping: map[string]any{
					"kubernetes": map[string]any{
						"namespace_name": "namespace",
					},
				},
				want: "garden",
			},
		),
		Entry("deep string",
			getDynamicHostNameArgs{
				records: types.OutputRecord{
					"int":   "42",
					"float": "42.42",
					"array": `[42,42.42,"foo"]`,
					"kubernetes": map[string]any{
						"label": map[string]any{
							"component": map[string]any{
								"buzz": "value",
							},
						},
					},
				},
				mapping: map[string]any{
					"kubernetes": map[string]any{
						"label": map[string]any{
							"component": map[string]any{
								"buzz": "label",
							},
						},
					},
				},
				want: "value",
			}),
	)

	DescribeTable("#fallbackToTagWhenMetadataIsMissing",
		func(args fallbackToTagWhenMetadataIsMissing) {
			re := regexp.MustCompile(args.tagPrefix + args.tagRegexp)
			err := extractKubernetesMetadataFromTag(args.records, args.tagKey, re)
			if args.err != nil {
				Expect(err.Error()).To(Equal(args.err.Error()))

				return
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(args.records).To(Equal(args.want))
		},
		Entry("records with correct tag",
			fallbackToTagWhenMetadataIsMissing{
				records: map[string]any{
					config.DefaultKubernetesMetadataTagKey: "kubernetes.var.log.containers.cluster-autoscaler-65d4ccbb7d-w5kd2_shoot--i355448--local-shoot_cluster-autoscaler-a8bba03512b5dd378c620ab3707aec013f83bdb9abae08d347e1644b064ed35f.log",
				},
				tagKey:    config.DefaultKubernetesMetadataTagKey,
				tagPrefix: config.DefaultKubernetesMetadataTagPrefix,
				tagRegexp: config.DefaultKubernetesMetadataTagExpression,
				want: map[string]any{
					config.DefaultKubernetesMetadataTagKey: "kubernetes.var.log.containers.cluster-autoscaler-65d4ccbb7d-w5kd2_shoot--i355448--local-shoot_cluster-autoscaler-a8bba03512b5dd378c620ab3707aec013f83bdb9abae08d347e1644b064ed35f.log",
					"kubernetes": map[string]any{
						podName:       "cluster-autoscaler-65d4ccbb7d-w5kd2",
						containerName: "cluster-autoscaler",
						namespaceName: "shoot--i355448--local-shoot",
						containerID:   "a8bba03512b5dd378c620ab3707aec013f83bdb9abae08d347e1644b064ed35f",
					},
				},
				err: nil,
			},
		),
		Entry("records with incorrect tag",
			fallbackToTagWhenMetadataIsMissing{
				records: map[string]any{
					config.DefaultKubernetesMetadataTagKey: "kubernetes.var.log.containers.cluster-autoscaler-65d4ccbb7d-w5kd2_shoot--i355448--local-shoot-cluster-autoscaler-a8bba03512b5dd378c620ab3707aec013f83bdb9abae08d347e1644b064ed35f.log",
				},
				tagKey:    config.DefaultKubernetesMetadataTagKey,
				tagPrefix: config.DefaultKubernetesMetadataTagPrefix,
				tagRegexp: config.DefaultKubernetesMetadataTagExpression,
				err:       fmt.Errorf("invalid format for tag %v. The tag should be in format: %s", "kubernetes.var.log.containers.cluster-autoscaler-65d4ccbb7d-w5kd2_shoot--i355448--local-shoot-cluster-autoscaler-a8bba03512b5dd378c620ab3707aec013f83bdb9abae08d347e1644b064ed35f.log", "kubernetes\\.var\\.log\\.containers"+config.DefaultKubernetesMetadataTagExpression),
			},
		),
		Entry("records with missing tag",
			fallbackToTagWhenMetadataIsMissing{
				records: map[string]any{
					"missing_tag": "kubernetes.var.log.containers.cluster-autoscaler-65d4ccbb7d-w5kd2_shoot--i355448--local-shoot-cluster-autoscaler-a8bba03512b5dd378c620ab3707aec013f83bdb9abae08d347e1644b064ed35f.log",
				},
				tagKey:    config.DefaultKubernetesMetadataTagKey,
				tagPrefix: config.DefaultKubernetesMetadataTagPrefix,
				tagRegexp: config.DefaultKubernetesMetadataTagExpression,
				err:       fmt.Errorf("the tag entry for key %q is missing", config.DefaultKubernetesMetadataTagKey),
			},
		),
	)
})
