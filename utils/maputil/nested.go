// Package maputil provides utilities for working with nested maps.
package maputil

import (
	"fmt"
	"strings"
)

// SetNestedValue sets a value in a nested map using a dot-separated key path.
// If intermediate maps don't exist, they will be created.
//
// Example:
//
//	m := make(map[string]any)
//	SetNestedValue(m, "database.mysql.host", "localhost")
//	// Result: m["database"]["mysql"]["host"] = "localhost"
func SetNestedValue(m map[string]any, keyPath string, value any) {
	parts := strings.Split(keyPath, ".")
	last := len(parts) - 1
	current := m

	for i, part := range parts {
		if i == last {
			current[part] = value
		} else {
			if next, exists := current[part]; exists {
				if nextMap, ok := next.(map[string]any); ok {
					current = nextMap
				} else {
					nextMap := make(map[string]any)
					current[part] = nextMap
					current = nextMap
				}
			} else {
				nextMap := make(map[string]any)
				current[part] = nextMap
				current = nextMap
			}
		}
	}
}

// GetNestedValue retrieves a value from a nested map using a dot-separated key path.
// Returns an error if the key path doesn't exist or if an intermediate value is not a map.
//
// Example:
//
//	m := map[string]any{"database": map[string]any{"host": "localhost"}}
//	val, err := GetNestedValue(m, "database.host")
//	// Result: val = "localhost"
func GetNestedValue(m map[string]any, keyPath string) (any, error) {
	keys := strings.Split(keyPath, ".")
	current := m

	for i, key := range keys {
		val, ok := current[key]
		if !ok {
			return nil, fmt.Errorf("key %q not found in path %q", key, keyPath)
		}

		// If this is the last key, return the value
		if i == len(keys)-1 {
			return val, nil
		}

		// Otherwise, it must be a map to continue traversing
		nextMap, ok := val.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("key %q is not a map, got %T", key, val)
		}
		current = nextMap
	}

	return current, nil
}

// GetNestedMap is a convenience wrapper around GetNestedValue that returns the value as a map.
// Returns an error if the value is not a map[string]any.
func GetNestedMap(m map[string]any, keyPath string) (map[string]any, error) {
	val, err := GetNestedValue(m, keyPath)
	if err != nil {
		return nil, err
	}

	result, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("value at %q is not a map, got %T", keyPath, val)
	}

	return result, nil
}