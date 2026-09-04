package contract

// Config defines the interface for the configuration repository.
type Config interface {
	// Get retrieves a configuration value.
	Get(key string, defaultValue ...interface{}) interface{}
	// GetString retrieves a configuration value as a string.
	GetString(key string, defaultValue ...string) string
	// GetInt retrieves a configuration value as an int.
	GetInt(key string, defaultValue ...int) int
	// GetBool retrieves a configuration value as a boolean.
	GetBool(key string, defaultValue ...bool) bool
	// Set sets a configuration value.
	Set(key string, value interface{})
	// SetStruct registers a struct or pointer to struct under a prefix and recursively maps its fields.
	SetStruct(prefix string, value interface{})
	// Load registers multiple struct or map configurations.
	Load(configs map[string]interface{})
	// Reload clears and reloads configurations.
	Reload(configs map[string]interface{})
	// Has checks if a configuration key exists.
	Has(key string) bool
	// All returns all configuration items.
	All() map[string]interface{}
	// Push pushes a value onto an array configuration value.
	Push(key string, value interface{})
	// Prepend prepends a value onto an array configuration value.
	Prepend(key string, value interface{})
	// GetMany retrieves multiple configuration values at once.
	GetMany(keys []string) map[string]interface{}
}
