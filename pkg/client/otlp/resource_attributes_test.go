// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package otlp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	. "github.com/gardener/logging/v1/pkg/client/otlp"
	"github.com/gardener/logging/v1/pkg/config"
)

var _ = Describe("ResourceAttributesBuilder", func() {
	Describe("NewResourceAttributesBuilder", func() {
		It("should create a new builder", func() {
			builder := NewResourceAttributesBuilder()
			Expect(builder).NotTo(BeNil())
		})
	})

	Describe("WithHostname", func() {
		It("should add host.name attribute when hostname value is set", func() {
			cfg := config.Config{
				PluginConfig: config.PluginConfig{
					HostnameValue: "node-1",
				},
			}

			builder := NewResourceAttributesBuilder()
			result := builder.WithHostname(cfg)
			Expect(result).To(Equal(builder))

			resource := builder.Build()
			Expect(resource.Attributes()).To(ContainElement(semconv.HostName("node-1")))
		})

		It("should not add host.name attribute when hostname value is empty", func() {
			cfg := config.Config{
				PluginConfig: config.PluginConfig{
					HostnameValue: "",
				},
			}

			builder := NewResourceAttributesBuilder()
			result := builder.WithHostname(cfg)
			Expect(result).To(Equal(builder))

			resource := builder.Build()
			Expect(resource.Attributes()).To(BeEmpty())
		})
	})

	Describe("Build", func() {
		It("should build resource with configured attributes and schema URL", func() {
			cfg := config.Config{
				PluginConfig: config.PluginConfig{
					HostnameValue: "node-1",
				},
			}

			resource := NewResourceAttributesBuilder().
				WithHostname(cfg).
				Build()

			Expect(resource).NotTo(BeNil())
			Expect(resource.SchemaURL()).To(Equal(semconv.SchemaURL))
			Expect(resource.Attributes()).To(HaveLen(1))
		})

		It("should build empty resource when no attributes set", func() {
			resource := NewResourceAttributesBuilder().Build()
			Expect(resource).NotTo(BeNil())
			Expect(resource.SchemaURL()).To(Equal(semconv.SchemaURL))
			Expect(resource.Attributes()).To(BeEmpty())
		})
	})
})
