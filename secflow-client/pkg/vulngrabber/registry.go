package vulngrabber

import (
	"fmt"
	"sync"
)

// ── Registry ──────────────────────────────────────────────────────────────

var (
	mu       sync.RWMutex
	registry = map[string]func() Grabber{}
)

// Register adds a grabber constructor to the global registry.
// It is intended to be called from init() functions.
func Register(name string, constructor func() Grabber) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("grabber %q already registered", name))
	}
	registry[name] = constructor
}

// ByName returns a new Grabber instance for the given source name.
func ByName(name string) (Grabber, error) {
	mu.RLock()
	ctor, ok := registry[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("grabber %q not registered", name)
	}
	return ctor(), nil
}

// Available returns the sorted list of registered grabber names.
func Available() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	return names
}

// GetAll returns all registered grabbers.
func GetAll() map[string]Grabber {
	mu.RLock()
	defer mu.RUnlock()
	result := make(map[string]Grabber, len(registry))
	for name, ctor := range registry {
		result[name] = ctor()
	}
	return result
}
