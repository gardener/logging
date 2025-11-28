// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/gardener/logging/v1/pkg/plugin"
)

// pluginsContains checks if a plugin with the given id exists in the plugins map.
// Uses sync.Map's Load method which is concurrent-safe.
func pluginsContains(id string) bool {
	_, ok := plugins.Load(id)

	return ok
}

// pluginsGet retrieves a plugin with the given id from the plugins map.
// Returns the plugin and a boolean indicating whether it was found.
// Uses sync.Map's Load method which is concurrent-safe.
func pluginsGet(id string) (plugin.OutputPlugin, bool) {
	val, ok := plugins.Load(id)
	if !ok {
		return nil, false
	}

	p, ok := val.(plugin.OutputPlugin)
	if !ok {
		return nil, false
	}

	return p, ok
}

// pluginsSet stores a plugin with the given id in the plugins map.
// Uses sync.Map's Store method which is concurrent-safe.
func pluginsSet(id string, p plugin.OutputPlugin) {
	plugins.Store(id, p)
}

// pluginsRemove removes a plugin with the given id from the plugins map.
// Uses sync.Map's Delete method which is concurrent-safe.
func pluginsRemove(id string) {
	plugins.Delete(id)
}

// pluginsLen returns the number of plugins in the plugins map.
// Uses sync.Map's Range method to count entries.
func pluginsLen() int {
	count := 0
	plugins.Range(func(_, _ any) bool {
		count++

		return true
	})

	return count
}
