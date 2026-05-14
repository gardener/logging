package plugin

import (
	"sync"

	"github.com/go-logr/logr"
)

var (
	reg          *Registry
	registryOnce sync.Once
)

// Registry registeres plugin instances, required for disposal during shutdown.
// Safe for concurrent use.
type Registry struct {
	sync.Map // map[string]OutputPlugin
	logger   logr.Logger
}

// InitRegistry initializes the global plugin registry singleton.
// It is safe to call multiple times; only the first call takes effect.
func InitRegistry(logger logr.Logger) error {
	registryOnce.Do(func() {
		reg = &Registry{
			logger: logger,
		}
	})
	return nil
}

// RegistryInst returns the global plugin registry instance.
// InitRegistry must be called before this function; otherwise it returns nil.
func RegistryInst() *Registry {
	return reg
}

// Contains checks if a plugin with the given id exists in the plugins map.
func (r *Registry) Contains(id string) bool {
	_, ok := r.Load(id)
	return ok
}

// Get retrieves a plugin with the given id from the plugins map.
// Returns the plugin and a boolean indicating whether it was found.
func (r *Registry) Get(id string) (OutputPlugin, bool) {
	val, ok := r.Load(id)
	if !ok {
		return nil, false
	}

	p, ok := val.(OutputPlugin)
	if !ok {
		return nil, false
	}

	return p, ok
}

// Set stores a plugin with the given id in the plugins map.
func (r *Registry) Set(id string, p OutputPlugin) {
	r.Store(id, p)
}

// Remove removes a plugin with the given id from the plugins map.
func (r *Registry) Remove(id string) {
	r.Delete(id)
}

// Len returns the number of plugins in the plugins map.
func (r *Registry) Len() int {
	count := 0
	r.Range(func(_, _ any) bool {
		count++
		return true
	})

	return count
}

// CleanupAll closes and removes all plugins from the plugins map.
// This is used during fluent-bit shutdown (FLBPluginExit) to ensure all resources are properly released.
// Each plugin's Close method is called to properly shutdown controllers and clients.
func (r *Registry) CleanupAll() {
	var idsToDelete []string

	// First, collect all plugin IDs and close them
	r.Range(func(key, value any) bool {
		id, ok := key.(string)
		if !ok {
			return true
		}

		p, ok := value.(OutputPlugin)
		if ok && p != nil {
			p.Close()
		}

		idsToDelete = append(idsToDelete, id)

		return true
	})

	// Then delete all entries
	for _, id := range idsToDelete {
		r.Delete(id)
	}

	if len(idsToDelete) > 0 {
		r.logger.Info("[flb-go] cleaned up plugins during shutdown", "count", len(idsToDelete))
	}
}
