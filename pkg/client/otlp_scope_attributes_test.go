// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/gardener/logging/v1/pkg/client"
)

var _ = Describe("ScopeAttributesBuilder", func() {
	Describe("NewScopeAttributesBuilder", func() {
		It("should create a new builder", func() {
			builder := NewScopeAttributesBuilder()
			Expect(builder).NotTo(BeNil())
		})
	})

	Describe("WithVersion", func() {
		It("should add version to scope options", func() {
			builder := NewScopeAttributesBuilder()
			result := builder.WithVersion(PluginVersion())
			Expect(result).To(Equal(builder))

			options := builder.Build()
			Expect(options).To(HaveLen(1))
		})
	})

	Describe("WithSchemaURL", func() {
		It("should add schema URL to scope options", func() {
			builder := NewScopeAttributesBuilder()
			result := builder.WithSchemaURL(SchemaURL)
			Expect(result).To(Equal(builder))

			options := builder.Build()
			Expect(options).To(HaveLen(1))
		})
	})

	Describe("Build", func() {
		It("should build complete scope options with version and schema", func() {
			builder := NewScopeAttributesBuilder().
				WithVersion(PluginVersion()).
				WithSchemaURL(SchemaURL)

			options := builder.Build()
			Expect(options).To(HaveLen(2))
		})

		It("should build empty options when no attributes set", func() {
			builder := NewScopeAttributesBuilder()
			options := builder.Build()
			Expect(options).To(HaveLen(0))
		})
	})

	Describe("Constants", func() {
		It("should have proper plugin name", func() {
			Expect(PluginName).To(Equal("fluent-bit-output-plugin"))
		})

		It("should have proper plugin version", func() {
			Expect(PluginVersion()).NotTo(BeEmpty())
		})

		It("should have proper schema URL", func() {
			Expect(SchemaURL).To(ContainSubstring("opentelemetry.io/schemas"))
		})
	})
})
