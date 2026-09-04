package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepository_GetSetHas(t *testing.T) {
	repo := NewRepository(map[string]interface{}{
		"app": map[string]interface{}{
			"name":  "Think",
			"env":   "production",
			"debug": true,
			"port":  8080,
		},
		"database": map[string]interface{}{
			"default": "mysql",
			"connections": map[string]interface{}{
				"mysql": map[string]interface{}{
					"host": "127.0.0.1",
					"port": 3306,
				},
			},
		},
	})

	// Test GetString
	assert.Equal(t, "Think", repo.GetString("app.name"))
	assert.Equal(t, "production", repo.GetString("app.env"))
	assert.Equal(t, "127.0.0.1", repo.GetString("database.connections.mysql.host"))
	assert.Equal(t, "fallback", repo.GetString("app.not_found", "fallback"))

	// Test GetBool
	assert.True(t, repo.GetBool("app.debug"))
	assert.False(t, repo.GetBool("app.non_existing", false))

	// Test GetInt
	assert.Equal(t, 8080, repo.GetInt("app.port"))
	assert.Equal(t, 3306, repo.GetInt("database.connections.mysql.port"))
	assert.Equal(t, 9000, repo.GetInt("app.missing_port", 9000))

	// Test Has
	assert.True(t, repo.Has("app.name"))
	assert.True(t, repo.Has("database.connections.mysql.host"))
	assert.False(t, repo.Has("app.invalid_key"))

	// Test Set
	repo.Set("app.name", "NewThinkApp")
	assert.Equal(t, "NewThinkApp", repo.GetString("app.name"))

	repo.Set("cache.default", "redis")
	assert.Equal(t, "redis", repo.GetString("cache.default"))
	assert.True(t, repo.Has("cache.default"))

	// Test All
	all := repo.All()
	assert.NotNil(t, all["app"])
	assert.NotNil(t, all["cache"])
}

func TestRepository_PushPrependGetMany(t *testing.T) {
	repo := NewRepository(map[string]interface{}{
		"app": map[string]interface{}{
			"providers": []interface{}{"AppProvider"},
		},
	})

	// 1. Test Push
	repo.Push("app.providers", "RouteProvider")
	val := repo.Get("app.providers").([]interface{})
	assert.Equal(t, []interface{}{"AppProvider", "RouteProvider"}, val)

	// 2. Test Prepend
	repo.Prepend("app.providers", "EventProvider")
	val = repo.Get("app.providers").([]interface{})
	assert.Equal(t, []interface{}{"EventProvider", "AppProvider", "RouteProvider"}, val)

	// 3. Test GetMany
	many := repo.GetMany([]string{"app.providers", "app.non_existing"})
	assert.Len(t, many, 2)
	assert.NotNil(t, many["app.providers"])
	assert.Nil(t, many["app.non_existing"])
}
