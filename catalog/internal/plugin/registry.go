package plugin

import (
	"flag"
	"fmt"
	"sync"
)

var globalRegistry = &Registry{
	plugins: make(map[string]CatalogPlugin),
}

// Registry holds all registered catalog plugins.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]CatalogPlugin
	order   []string
}

// Register adds a plugin to the global registry.
// This is typically called from a plugin's init() function.
// Panics if a plugin with the same name is already registered.
func Register(p CatalogPlugin) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	name := p.Name()
	if _, exists := globalRegistry.plugins[name]; exists {
		panic(fmt.Sprintf("plugin %q already registered", name))
	}

	globalRegistry.plugins[name] = p
	globalRegistry.order = append(globalRegistry.order, name)
}

// All returns all registered plugins in registration order.
func All() []CatalogPlugin {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	result := make([]CatalogPlugin, 0, len(globalRegistry.order))
	for _, name := range globalRegistry.order {
		result = append(result, globalRegistry.plugins[name])
	}
	return result
}

// Get returns a plugin by name. The returned boolean is true if the plugin is found, false otherwise.
func Get(name string) (CatalogPlugin, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	p, ok := globalRegistry.plugins[name]
	return p, ok
}

// Names returns all registered plugin names in registration order.
func Names() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	result := make([]string, len(globalRegistry.order))
	copy(result, globalRegistry.order)
	return result
}

// Count returns the number of registered plugins.
func Count() int {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	return len(globalRegistry.plugins)
}

// RegisterAllFlags iterates all registered plugins and calls RegisterFlags
// on those implementing FlagProvider. Call this before flag.Parse().
func RegisterAllFlags(fs *flag.FlagSet) {
	for _, p := range All() {
		if fp, ok := p.(FlagProvider); ok {
			fp.RegisterFlags(fs)
		}
	}
}

// Reset clears the global registry. For testing only.
func Reset() {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	globalRegistry.plugins = make(map[string]CatalogPlugin)
	globalRegistry.order = nil
}
