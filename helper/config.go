package helper

import (
	"github.com/go-think/think/facades"
	"github.com/go-think/think/support/env"
)

// Env gets an environment variable, returning a fallback if it isn't set.
func Env(key string, defaultVal ...string) string {
	return env.Get(key, defaultVal...)
}

// Config retrieves a configuration value from the global configuration repository.
func Config(key string, defaultVal ...interface{}) interface{} {
	if facades.App != nil {
		if cfg := facades.Config(); cfg != nil {
			return cfg.Get(key, defaultVal...)
		}
	}

	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return nil
}
