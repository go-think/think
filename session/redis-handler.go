package session

import (
	"time"

	"github.com/gomodule/redigo/redis"
)

type RedisHandler struct {
	pool     *redis.Pool // redis connection pool
	prefix   string
	lifetime time.Duration
}

// NewRedisHandler Create a redis session handler
func NewRedisHandler(pool *redis.Pool, prefix string, lifetime time.Duration) *RedisHandler {
	return &RedisHandler{pool, prefix, lifetime}
}

func (rh *RedisHandler) Read(id string) string {
	c := rh.pool.Get()
	defer c.Close()

	val, err := redis.String(c.Do("GET", rh.prefix+":"+id))
	if err != nil {
		return ""
	}

	return val
}

func (rh *RedisHandler) Write(id string, data string) {
	c := rh.pool.Get()
	defer c.Close()

	_, _ = c.Do("SETEX", rh.prefix+":"+id, int64(rh.lifetime.Seconds()), data)
}

