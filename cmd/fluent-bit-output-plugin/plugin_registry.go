// Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
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

// pluginsCleanupAll closes and removes all plugins from the plugins map.
// This is used during fluent-bit shutdown (FLBPluginExit) to ensure all resources are properly released.
// Each plugin's Close method is called to properly shutdown controllers and clients.
func pluginsCleanupAll() {
	var idsToDelete []string

	// First, collect all plugin IDs and close them
	plugins.Range(func(key, value any) bool {
		id, ok := key.(string)
		if !ok {
			return true
		}

		p, ok := value.(plugin.OutputPlugin)
		if ok && p != nil {
			p.Close()
		}

		idsToDelete = append(idsToDelete, id)

		return true
	})

	// Then delete all entries
	for _, id := range idsToDelete {
		plugins.Delete(id)
	}

	if len(idsToDelete) > 0 {
		logger.Info("[flb-go] cleaned up plugins during shutdown", "count", len(idsToDelete))
	}
}
