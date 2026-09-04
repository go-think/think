package provider

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-think/cache"
	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
	"github.com/gomodule/redigo/redis"
)

// CacheServiceProvider registers the core cache services into the container.
type CacheServiceProvider struct{}

// Register registers cache services into the container using lazy resolution.
func (p *CacheServiceProvider) Register(app *container.Container) {
	app.Singleton[*cache.Repository](func() *cache.Repository {
		prefix := ""
		driver := "memory"

		var cfg contract.Config
		if app != nil {
			cfg = app.Make[contract.Config]()
		}

		if cfg != nil {
			if d := cfg.GetString("cache.default"); d != "" {
				driver = d
			}
			if p := cfg.GetString("cache.prefix"); p != "" {
				prefix = strings.TrimSuffix(p, ":")
			}
		}

		var store cache.Store
		switch driver {
		case "redis":
			store = p.createRedisStore(app, cfg, prefix)
		case "memory", "array":
			store = cache.NewMemoryStore(prefix)
		default:
			store = cache.NewMemoryStore(prefix)
		}

		return cache.NewRepository(store)
	})

	app.Alias[*cache.Repository]("cache")
}

// createRedisStore creates a cache.Store backed by Redis.
func (p *CacheServiceProvider) createRedisStore(app *container.Container, cfg contract.Config, prefix string) cache.Store {
	// Reuse existing *redis.Pool instance if already registered in the container
	if app != nil {
		if pool := app.Make[*redis.Pool](); pool != nil {
			return cache.NewRedisStore(pool, prefix)
		}
	}

	host := "127.0.0.1"
	port := 6379
	password := ""
	db := 0
	connectTimeout := 5 * time.Second
	readTimeout := 3 * time.Second
	writeTimeout := 3 * time.Second

	if cfg != nil {
		if h := cfg.GetString("cache.stores.redis.host"); h != "" {
			host = h
		} else if h := cfg.GetString("cache.redis.host"); h != "" {
			host = h
		} else if h := cfg.GetString("database.redis.default.host"); h != "" {
			host = h
		}

		if pt := cfg.GetInt("cache.stores.redis.port"); pt > 0 {
			port = pt
		} else if pt := cfg.GetInt("cache.redis.port"); pt > 0 {
			port = pt
		} else if pt := cfg.GetInt("database.redis.default.port"); pt > 0 {
			port = pt
		}

		if pwd := cfg.GetString("cache.stores.redis.password"); pwd != "" {
			password = pwd
		} else if pwd := cfg.GetString("cache.redis.password"); pwd != "" {
			password = pwd
		} else if pwd := cfg.GetString("database.redis.default.password"); pwd != "" {
			password = pwd
		}

		if database := cfg.GetInt("cache.stores.redis.database"); database > 0 {
			db = database
		} else if database := cfg.GetInt("cache.redis.database"); database > 0 {
			db = database
		}
	}

	pool := &redis.Pool{
		MaxIdle:     10,
		MaxActive:   100,
		IdleTimeout: 240 * time.Second,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			opts := []redis.DialOption{
				redis.DialConnectTimeout(connectTimeout),
				redis.DialReadTimeout(readTimeout),
				redis.DialWriteTimeout(writeTimeout),
			}
			if password != "" {
				opts = append(opts, redis.DialPassword(password))
			}
			if db > 0 {
				opts = append(opts, redis.DialDatabase(db))
			}
			return redis.Dial("tcp", fmt.Sprintf("%s:%d", host, port), opts...)
		},
	}

	return cache.NewRedisStore(pool, prefix)
}

// Boot boots the cache service provider.
func (p *CacheServiceProvider) Boot(app *container.Container) {}
