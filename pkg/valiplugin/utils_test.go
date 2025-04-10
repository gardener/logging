// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package valiplugin

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"

	jsoniter "github.com/json-iterator/go"
	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/prometheus/common/model"

	"github.com/gardener/logging/pkg/config"
)

type createLineArgs struct {
	records map[string]interface{}
	f       config.Format
	want    string
	wantErr bool
}

type removeKeysArgs struct {
	records  map[string]interface{}
	expected map[string]interface{}
	keys     []string
}

type extractLabelsArgs struct {
	records map[string]interface{}
	keys    []string
	want    model.LabelSet
}

type toStringMapArgs struct {
	record map[interface{}]interface{}
	want   map[string]interface{}
}

type labelMappingArgs struct {
	records map[string]interface{}
	mapping map[string]interface{}
	want    model.LabelSet
}

type getDynamicHostNameArgs struct {
	records map[string]interface{}
	mapping map[string]interface{}
	want    string
}

type autoKubernetesLabelsArgs struct {
	records map[interface{}]interface{}
	want    model.LabelSet
	err     error
}
type fallbackToTagWhenMetadataIsMissing struct {
	records   map[string]interface{}
	tagKey    string
	tagPrefix string
	tagRegexp string
	want      map[string]interface{}
	err       error
}

var _ = ginkgov2.Describe("Vali plugin utils", func() {
	ginkgov2.DescribeTable("#createLine",
		func(args createLineArgs) {
			got, err := createLine(args.records, args.f)
			if args.wantErr {
				gomega.Expect(err).To(gomega.HaveOccurred())

				return
			}
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			if args.f == config.JSONFormat {
				result, err := compareJSON(got, args.want)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(result).To(gomega.BeTrue())

				return
			}
			gomega.Expect(got).To(gomega.Equal(args.want))
		},
		ginkgov2.Entry("json",
			createLineArgs{
				records: map[string]interface{}{"foo": "bar", "bar": map[string]interface{}{"bizz": "bazz"}},
				f:       config.JSONFormat,
				want:    `{"foo":"bar","bar":{"bizz":"bazz"}}`,
				wantErr: false,
			},
		),
		ginkgov2.Entry("json with number",
			createLineArgs{
				records: map[string]interface{}{"foo": "bar", "bar": map[string]interface{}{"bizz": 20}},
				f:       config.JSONFormat,
				want:    `{"foo":"bar","bar":{"bizz":20}}`,
				wantErr: false,
			},
		),
		ginkgov2.Entry("bad json",
			createLineArgs{
				records: map[string]interface{}{"foo": make(chan interface{})},
				f:       config.JSONFormat,
				want:    "",
				wantErr: true,
			},
		),
		ginkgov2.Entry("kv with space",
			createLineArgs{
				records: map[string]interface{}{"foo": "bar", "bar": "foo foo"},
				f:       config.KvPairFormat,
				want:    `bar="foo foo" foo=bar`,
				wantErr: false,
			},
		),
		ginkgov2.Entry("kv with number",
			createLineArgs{
				records: map[string]interface{}{"foo": "bar foo", "decimal": 12.2},
				f:       config.KvPairFormat,
				want:    `decimal=12.2 foo="bar foo"`,
				wantErr: false,
			},
		),
		ginkgov2.Entry("kv with nil",
			createLineArgs{
				records: map[string]interface{}{"foo": "bar", "null": nil},
				f:       config.KvPairFormat,
				want:    `foo=bar null=null`,
				wantErr: false,
			},
		),
		ginkgov2.Entry("kv with array",
			createLineArgs{
				records: map[string]interface{}{"foo": "bar", "array": []string{"foo", "bar"}},
				f:       config.KvPairFormat,
				want:    `array="[foo bar]" foo=bar`,
				wantErr: false,
			},
		),
		ginkgov2.Entry("kv with map",
			createLineArgs{
				records: map[string]interface{}{"foo": "bar", "map": map[string]interface{}{"foo": "bar", "bar": "foo "}},
				f:       config.KvPairFormat,
				want:    `foo=bar map="map[bar:foo  foo:bar]"`,
				wantErr: false,
			},
		),
		ginkgov2.Entry("kv empty",
			createLineArgs{
				records: map[string]interface{}{},
				f:       config.KvPairFormat,
				want:    ``,
				wantErr: false,
			},
		),
		ginkgov2.Entry("bad format",
			createLineArgs{
				records: map[string]interface{}{},
				f:       config.Format(3),
				want:    "",
				wantErr: true,
			},
		))

	ginkgov2.DescribeTable("#removeKeys",
		func(args removeKeysArgs) {
			removeKeys(args.records, args.keys)
			gomega.Expect(args.expected).To(gomega.Equal(args.records))
		},
		ginkgov2.Entry("remove all keys",
			removeKeysArgs{
				records:  map[string]interface{}{"foo": "bar", "bar": map[string]interface{}{"bizz": "bazz"}},
				expected: map[string]interface{}{},
				keys:     []string{"foo", "bar"},
			},
		),
		ginkgov2.Entry("remove none",
			removeKeysArgs{
				records:  map[string]interface{}{"foo": "bar"},
				expected: map[string]interface{}{"foo": "bar"},
				keys:     []string{},
			},
		),
		ginkgov2.Entry("remove not existing",
			removeKeysArgs{
				records:  map[string]interface{}{"foo": "bar"},
				expected: map[string]interface{}{"foo": "bar"},
				keys:     []string{"bar"},
			},
		),
		ginkgov2.Entry("remove one",
			removeKeysArgs{
				records:  map[string]interface{}{"foo": "bar", "bazz": "buzz"},
				expected: map[string]interface{}{"foo": "bar"},
				keys:     []string{"bazz"},
			},
		),
	)

	ginkgov2.DescribeTable("#extractLabels",
		func(args extractLabelsArgs) {
			got := extractLabels(args.records, args.keys)
			gomega.Expect(got).To(gomega.Equal(args.want))
		},
		ginkgov2.Entry("single string",
			extractLabelsArgs{
				records: map[string]interface{}{"foo": "bar", "bar": map[string]interface{}{"bizz": "bazz"}},
				keys:    []string{"foo"},
				want:    model.LabelSet{"foo": "bar"},
			},
		),
		ginkgov2.Entry("multiple",
			extractLabelsArgs{
				records: map[string]interface{}{"foo": "bar", "bar": map[string]interface{}{"bizz": "bazz"}},
				keys:    []string{"foo", "bar"},
				want:    model.LabelSet{"foo": "bar", "bar": "map[bizz:bazz]"},
			},
		),
		ginkgov2.Entry("nil",
			extractLabelsArgs{
				records: map[string]interface{}{"foo": nil},
				keys:    []string{"foo"},
				want:    model.LabelSet{"foo": "<nil>"},
			},
		),
		ginkgov2.Entry("none",
			extractLabelsArgs{
				records: map[string]interface{}{"foo": nil},
				keys:    []string{},
				want:    model.LabelSet{},
			},
		),
		ginkgov2.Entry("missing",
			extractLabelsArgs{
				records: map[string]interface{}{"foo": "bar"},
				keys:    []string{"foo", "buzz"},
				want:    model.LabelSet{"foo": "bar"},
			},
		),
		ginkgov2.Entry("skip invalid",
			extractLabelsArgs{
				records: map[string]interface{}{"foo.blah": "bar", "bar": "a\xc5z"},
				keys:    []string{"foo.blah", "bar"},
				want:    model.LabelSet{},
			},
		),
	)

	ginkgov2.DescribeTable("#extractLabels",
		func(args toStringMapArgs) {
			got := toStringMap(args.record)
			gomega.Expect(got).To(gomega.Equal(args.want))
		},
		ginkgov2.Entry("already string",
			toStringMapArgs{
				record: map[interface{}]interface{}{"string": "foo", "bar": []byte("buzz")},
				want:   map[string]interface{}{"string": "foo", "bar": "buzz"},
			},
		),
		ginkgov2.Entry("skip non string",
			toStringMapArgs{
				record: map[interface{}]interface{}{"string": "foo", 1.0: []byte("buzz")},
				want:   map[string]interface{}{"string": "foo"},
			},
		),
		ginkgov2.Entry("byteslice in array",
			toStringMapArgs{
				record: map[interface{}]interface{}{"string": "foo", "bar": []interface{}{map[interface{}]interface{}{"baz": []byte("quux")}}},
				want:   map[string]interface{}{"string": "foo", "bar": []interface{}{map[string]interface{}{"baz": "quux"}}},
			},
		),
	)

	ginkgov2.DescribeTable("labelMapping",
		func(args labelMappingArgs) {
			got := model.LabelSet{}
			mapLabels(args.records, args.mapping, got)
			gomega.Expect(got).To(gomega.Equal(args.want))
		},
		ginkgov2.Entry("empty record",
			labelMappingArgs{
				records: map[string]interface{}{},
				mapping: map[string]interface{}{},
				want:    model.LabelSet{},
			},
		),
		ginkgov2.Entry("empty subrecord",
			labelMappingArgs{
				records: map[string]interface{}{
					"kubernetes": map[interface{}]interface{}{
						"foo": []byte("buzz"),
					},
				},
				mapping: map[string]interface{}{},
				want:    model.LabelSet{},
			},
		),
		ginkgov2.Entry("deep string",
			labelMappingArgs{
				records: map[string]interface{}{
					"int":   "42",
					"float": "42.42",
					"array": `[42,42.42,"foo"]`,
					"kubernetes": map[string]interface{}{
						"label": map[string]interface{}{
							"component": map[string]interface{}{
								"buzz": "value",
							},
						},
					},
				},
				mapping: map[string]interface{}{
					"int":   "int",
					"float": "float",
					"array": "array",
					"kubernetes": map[string]interface{}{
						"label": map[string]interface{}{
							"component": map[string]interface{}{
								"buzz": "label",
							},
						},
					},
					"stream": "output",
					"nope":   "nope",
				},
				want: model.LabelSet{
					"int":   "42",
					"float": "42.42",
					"array": `[42,42.42,"foo"]`,
					"label": "value",
				},
			},
		))

	ginkgov2.DescribeTable("#getDynamicHostName",
		func(args getDynamicHostNameArgs) {
			got := getDynamicHostName(args.records, args.mapping)
			gomega.Expect(got).To(gomega.Equal(args.want))
		},
		ginkgov2.Entry("empty record",
			getDynamicHostNameArgs{
				records: map[string]interface{}{},
				mapping: map[string]interface{}{
					"kubernetes": map[string]interface{}{
						"namespace_name": "namespace",
					},
				},
				want: "",
			},
		),
		ginkgov2.Entry("empty mapping",
			getDynamicHostNameArgs{
				records: map[string]interface{}{
					"kubernetes": map[string]interface{}{
						"foo":            []byte("buzz"),
						"namespace_name": []byte("garden"),
					},
				},
				mapping: map[string]interface{}{},
				want:    "",
			},
		),
		ginkgov2.Entry("empty subrecord",
			getDynamicHostNameArgs{
				records: map[string]interface{}{
					"kubernetes": map[string]interface{}{
						"foo": []byte("buzz"),
					},
				},
				mapping: map[string]interface{}{
					"kubernetes": map[string]interface{}{
						"namespace_name": "namespace",
					},
				},
				want: "",
			},
		),
		ginkgov2.Entry("subrecord",
			getDynamicHostNameArgs{
				records: map[string]interface{}{
					"kubernetes": map[string]interface{}{
						"foo":            []byte("buzz"),
						"namespace_name": []byte("garden"),
					},
				},
				mapping: map[string]interface{}{
					"kubernetes": map[string]interface{}{
						"namespace_name": "namespace",
					},
				},
				want: "garden",
			},
		),
		ginkgov2.Entry("deep string",
			getDynamicHostNameArgs{
				records: map[string]interface{}{
					"int":   "42",
					"float": "42.42",
					"array": `[42,42.42,"foo"]`,
					"kubernetes": map[string]interface{}{
						"label": map[string]interface{}{
							"component": map[string]interface{}{
								"buzz": "value",
							},
						},
					},
				},
				mapping: map[string]interface{}{
					"kubernetes": map[string]interface{}{
						"label": map[string]interface{}{
							"component": map[string]interface{}{
								"buzz": "label",
							},
						},
					},
				},
				want: "value",
			}),
	)

	ginkgov2.DescribeTable("#autoKubernetesLabels",
		func(args autoKubernetesLabelsArgs) {
			m := toStringMap(args.records)
			lbs := model.LabelSet{}
			err := autoLabels(m, lbs)
			if args.err != nil {
				gomega.Expect(err.Error()).To(gomega.Equal(args.err.Error()))

				return
			}
			gomega.Expect(lbs).To(gomega.Equal(args.want))
		},
		ginkgov2.Entry("records without labels",
			autoKubernetesLabelsArgs{
				records: map[interface{}]interface{}{
					"kubernetes": map[interface{}]interface{}{
						"foo": []byte("buzz"),
					},
				},
				want: model.LabelSet{
					"foo": "buzz",
				},
				err: nil,
			},
		),
		ginkgov2.Entry("records without kubernetes labels",
			autoKubernetesLabelsArgs{
				records: map[interface{}]interface{}{
					"foo":   "bar",
					"label": "value",
				},
				want: model.LabelSet{},
				err:  errors.New("kubernetes labels not found, no labels will be added"),
			},
		),
	)

	ginkgov2.DescribeTable("#fallbackToTagWhenMetadataIsMissing",
		func(args fallbackToTagWhenMetadataIsMissing) {
			re := regexp.MustCompile(args.tagPrefix + args.tagRegexp)
			err := extractKubernetesMetadataFromTag(args.records, args.tagKey, re)
			if args.err != nil {
				gomega.Expect(err.Error()).To(gomega.Equal(args.err.Error()))

				return
			}
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(args.records).To(gomega.Equal(args.want))
		},
		ginkgov2.Entry("records with correct tag",
			fallbackToTagWhenMetadataIsMissing{
				records: map[string]interface{}{
					config.DefaultKubernetesMetadataTagKey: "kubernetes.var.log.containers.cluster-autoscaler-65d4ccbb7d-w5kd2_shoot--i355448--local-shoot_cluster-autoscaler-a8bba03512b5dd378c620ab3707aec013f83bdb9abae08d347e1644b064ed35f.log",
				},
				tagKey:    config.DefaultKubernetesMetadataTagKey,
				tagPrefix: config.DefaultKubernetesMetadataTagPrefix,
				tagRegexp: config.DefaultKubernetesMetadataTagExpression,
				want: map[string]interface{}{
					config.DefaultKubernetesMetadataTagKey: "kubernetes.var.log.containers.cluster-autoscaler-65d4ccbb7d-w5kd2_shoot--i355448--local-shoot_cluster-autoscaler-a8bba03512b5dd378c620ab3707aec013f83bdb9abae08d347e1644b064ed35f.log",
					"kubernetes": map[string]interface{}{
						podName:       "cluster-autoscaler-65d4ccbb7d-w5kd2",
						containerName: "cluster-autoscaler",
						namespaceName: "shoot--i355448--local-shoot",
						containerID:   "a8bba03512b5dd378c620ab3707aec013f83bdb9abae08d347e1644b064ed35f",
					},
				},
				err: nil,
			},
		),
		ginkgov2.Entry("records with incorrect tag",
			fallbackToTagWhenMetadataIsMissing{
				records: map[string]interface{}{
					config.DefaultKubernetesMetadataTagKey: "kubernetes.var.log.containers.cluster-autoscaler-65d4ccbb7d-w5kd2_shoot--i355448--local-shoot-cluster-autoscaler-a8bba03512b5dd378c620ab3707aec013f83bdb9abae08d347e1644b064ed35f.log",
				},
				tagKey:    config.DefaultKubernetesMetadataTagKey,
				tagPrefix: config.DefaultKubernetesMetadataTagPrefix,
				tagRegexp: config.DefaultKubernetesMetadataTagExpression,
				err:       fmt.Errorf("invalid format for tag %v. The tag should be in format: %s", "kubernetes.var.log.containers.cluster-autoscaler-65d4ccbb7d-w5kd2_shoot--i355448--local-shoot-cluster-autoscaler-a8bba03512b5dd378c620ab3707aec013f83bdb9abae08d347e1644b064ed35f.log", "kubernetes\\.var\\.log\\.containers"+config.DefaultKubernetesMetadataTagExpression),
			},
		),
		ginkgov2.Entry("records with missing tag",
			fallbackToTagWhenMetadataIsMissing{
				records: map[string]interface{}{
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

// compareJson unmarshal both string to map[string]interface compare json result.
// we can't compare string to string as jsoniter doesn't ensure field ordering.
func compareJSON(got, want string) (bool, error) {
	var w map[string]interface{}
	err := jsoniter.Unmarshal([]byte(want), &w)
	if err != nil {
		return false, err
	}
	var g map[string]interface{}
	err = jsoniter.Unmarshal([]byte(got), &g)
	if err != nil {
		return false, err
	}

	return reflect.DeepEqual(g, w), nil
}
