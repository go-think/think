package config

import (
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Repository is the configuration manager.
type Repository struct {
	items map[string]interface{}
	mu    sync.RWMutex
}

var (
	defaultRepo     *Repository
	defaultRepoOnce sync.Once
)

// Default returns the default global repository instance.
func Default() *Repository {
	defaultRepoOnce.Do(func() {
		defaultRepo = NewRepository(nil)
	})
	return defaultRepo
}

// NewRepository creates a new configuration repository.
func NewRepository(items map[string]interface{}) *Repository {
	if items == nil {
		items = make(map[string]interface{})
	}
	return &Repository{
		items: items,
	}
}

// Get retrieves a configuration value by key using dot notation.
func (r *Repository) Get(key string, defaultValue ...interface{}) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := strings.Split(key, ".")
	var current interface{} = r.items

	for _, k := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[k]; exists {
				current = val
			} else {
				if len(defaultValue) > 0 {
					return defaultValue[0]
				}
				return nil
			}
		} else {
			if len(defaultValue) > 0 {
				return defaultValue[0]
			}
			return nil
		}
	}

	return current
}

// Set sets a configuration value by key using dot notation.
func (r *Repository) Set(key string, value interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	keys := strings.Split(key, ".")
	current := r.items

	for i := 0; i < len(keys)-1; i++ {
		k := keys[i]
		if _, exists := current[k]; !exists {
			current[k] = make(map[string]interface{})
		}
		
		if m, ok := current[k].(map[string]interface{}); ok {
			current = m
		} else {
			// If it's not a map, we overwrite it with a map to continue
			newMap := make(map[string]interface{})
			current[k] = newMap
			current = newMap
		}
	}

	current[keys[len(keys)-1]] = value
}

// GetString retrieves a configuration value as a string.
func (r *Repository) GetString(key string, defaultValue ...string) string {
	val := r.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return "" // or fmt.Sprint(val)
}

// GetInt retrieves a configuration value as an int.
func (r *Repository) GetInt(key string, defaultValue ...int) int {
	val := r.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		i, _ := strconv.Atoi(v)
		return i
	default:
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}
}

// GetBool retrieves a configuration value as a boolean.
func (r *Repository) GetBool(key string, defaultValue ...bool) bool {
	val := r.Get(key)
	if val == nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		b, _ := strconv.ParseBool(v)
		return b
	case int:
		return v != 0
	default:
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}
}

// Has checks if a configuration key exists.
func (r *Repository) Has(key string) bool {
	return r.Get(key) != nil
}

// All returns all configuration items.
func (r *Repository) All() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.items
}

// Load registers multiple struct or map configurations.
func (r *Repository) Load(configs map[string]interface{}) {
	for prefix, val := range configs {
		r.SetStruct(prefix, val)
	}
}

// Reload clears existing configuration items and loads new configurations.
func (r *Repository) Reload(configs map[string]interface{}) {
	r.mu.Lock()
	r.items = make(map[string]interface{})
	r.mu.Unlock()

	r.Load(configs)
}

// Reset resets the default global repository singleton (useful for testing and reload cycles).
func Reset() {
	defaultRepoOnce = sync.Once{}
	defaultRepo = nil
}

// SetStruct recursively maps a struct or pointer-to-struct into dot-notation keys.
func (r *Repository) SetStruct(prefix string, value interface{}) {
	if value == nil {
		return
	}

	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		// Not a struct (e.g. map or scalar), directly set
		r.Set(prefix, value)
		return
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}

		keyName := fieldType.Tag.Get("config")
		if keyName == "" {
			keyName = strings.ToLower(fieldType.Name)
		}

		fullKey := keyName
		if prefix != "" {
			fullKey = prefix + "." + keyName
		}

		fieldReal := fieldVal
		if fieldReal.Kind() == reflect.Ptr {
			if fieldReal.IsNil() {
				continue
			}
			fieldReal = fieldReal.Elem()
		}

		if fieldReal.Kind() == reflect.Struct {
			r.SetStruct(fullKey, fieldVal.Interface())
		} else {
			r.Set(fullKey, fieldVal.Interface())
		}
	}
}

// Push pushes a value onto an array configuration value.
func (r *Repository) Push(key string, value interface{}) {
	current := r.Get(key)
	var list []interface{}
	if current != nil {
		if arr, ok := current.([]interface{}); ok {
			list = arr
		} else {
			list = []interface{}{current}
		}
	} else {
		list = []interface{}{}
	}
	list = append(list, value)
	r.Set(key, list)
}

// Prepend prepends a value onto an array configuration value.
func (r *Repository) Prepend(key string, value interface{}) {
	current := r.Get(key)
	var list []interface{}
	if current != nil {
		if arr, ok := current.([]interface{}); ok {
			list = arr
		} else {
			list = []interface{}{current}
		}
	} else {
		list = []interface{}{}
	}
	list = append([]interface{}{value}, list...)
	r.Set(key, list)
}

// GetMany retrieves multiple configuration values at once.
func (r *Repository) GetMany(keys []string) map[string]interface{} {
	results := make(map[string]interface{}, len(keys))
	for _, key := range keys {
		results[key] = r.Get(key)
	}
	return results
}


