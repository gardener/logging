// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/logging/v1/pkg/log"
	"github.com/gardener/logging/v1/pkg/types"
)

// fakePlugin is a minimal OutputPlugin implementation for testing.
type fakePlugin struct {
	closed bool
}

//nolint:revive // receiver-naming
func (f *fakePlugin) SendRecord(_ types.OutputEntry) error {
	return nil
}

func (f *fakePlugin) Close() {
	f.closed = true
}

var _ = Describe("Plugin Registry", func() {
	var registry *Registry

	BeforeEach(func() {
		registry = &Registry{
			logger: log.NewNopLogger(),
		}
	})

	Describe("Contains", func() {
		It("should return false for a non-existent plugin", func() {
			Expect(registry.Contains("unknown")).To(BeFalse())
		})

		It("should return true for an existing plugin", func() {
			registry.Set("plugin-1", &fakePlugin{})
			Expect(registry.Contains("plugin-1")).To(BeTrue())
		})
	})

	Describe("Get", func() {
		It("should return nil and false for a non-existent plugin", func() {
			p, ok := registry.Get("missing")
			Expect(ok).To(BeFalse())
			Expect(p).To(BeNil())
		})

		It("should return the plugin and true for an existing plugin", func() {
			expected := &fakePlugin{}
			registry.Set("plugin-1", expected)

			p, ok := registry.Get("plugin-1")
			Expect(ok).To(BeTrue())
			Expect(p).To(Equal(expected))
		})

		It("should return nil and false when the stored value is not an OutputPlugin", func() {
			// Store a non-OutputPlugin value directly via sync.Map
			registry.Store("bad", "not-a-plugin")

			p, ok := registry.Get("bad")
			Expect(ok).To(BeFalse())
			Expect(p).To(BeNil())
		})
	})

	Describe("Set", func() {
		It("should store a plugin that can be retrieved", func() {
			plugin := &fakePlugin{}
			registry.Set("p1", plugin)

			retrieved, ok := registry.Get("p1")
			Expect(ok).To(BeTrue())
			Expect(retrieved).To(Equal(plugin))
		})

		It("should overwrite an existing plugin", func() {
			first := &fakePlugin{}
			second := &fakePlugin{}

			registry.Set("p1", first)
			registry.Set("p1", second)

			retrieved, ok := registry.Get("p1")
			Expect(ok).To(BeTrue())
			Expect(retrieved).To(Equal(second))
		})
	})

	Describe("Remove", func() {
		It("should remove an existing plugin", func() {
			registry.Set("p1", &fakePlugin{})
			registry.Remove("p1")

			Expect(registry.Contains("p1")).To(BeFalse())
		})

		It("should not panic when removing a non-existent plugin", func() {
			Expect(func() { registry.Remove("non-existent") }).NotTo(Panic())
		})
	})

	Describe("Len", func() {
		It("should return 0 for an empty registry", func() {
			Expect(registry.Len()).To(Equal(0))
		})

		It("should return the correct count", func() {
			registry.Set("p1", &fakePlugin{})
			registry.Set("p2", &fakePlugin{})
			registry.Set("p3", &fakePlugin{})

			Expect(registry.Len()).To(Equal(3))
		})
	})

	Describe("CleanupAll", func() {
		It("should close and remove all plugins", func() {
			p1 := &fakePlugin{}
			p2 := &fakePlugin{}

			registry.Set("p1", p1)
			registry.Set("p2", p2)

			registry.CleanupAll()

			Expect(p1.closed).To(BeTrue())
			Expect(p2.closed).To(BeTrue())
			Expect(registry.Len()).To(Equal(0))
		})

		It("should handle an empty registry without error", func() {
			Expect(func() { registry.CleanupAll() }).NotTo(Panic())
			Expect(registry.Len()).To(Equal(0))
		})
	})
})
