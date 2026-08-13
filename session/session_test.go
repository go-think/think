package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-think/think/context"
	"github.com/stretchr/testify/assert"
)

func TestSessionManagerFileDriver(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "session_test_*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cfg := &Config{
		Driver:     "file",
		CookieName: "think_sess",
		Lifetime:   time.Hour,
		Files:      filepath.Join(tmpDir, "sessions"),
	}

	mgr := NewManager(cfg)

	// 模拟请求
	r := context.NewRequest(nil)
	store := mgr.SessionStart(r)
	assert.NotNil(t, store)

	sessID := store.GetId()
	assert.NotEmpty(t, sessID)

	store.Set("user_id", 1001)
	assert.Equal(t, 1001, store.Get("user_id"))

	res := context.NewResponse()
	mgr.SessionSave(res, store)

	// 模拟第二次带着 Cookie 请求
	r2 := context.NewRequest(nil)
	r2.CookieHandler = context.ParseCookieHandler()
	r2.Set("cookie_sess", sessID)

	store2 := NewStore("think_sess", &FileHandler{
		Path:     filepath.Join(tmpDir, "sessions"),
		Lifetime: time.Hour,
	})
	store2.SetId(sessID)
	store2.Start()

	assert.Equal(t, float64(1001), store2.Get("user_id"))
}
