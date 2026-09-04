package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-think/think/flow"
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

	// Simulate request
	r := flow.NewRequest(nil)
	store := mgr.SessionStart(r)
	assert.NotNil(t, store)

	sessID := store.GetId()
	assert.NotEmpty(t, sessID)

	store.Set("user_id", 1001)
	assert.Equal(t, 1001, store.Get("user_id"))

	res := flow.NewResponse()
	mgr.SessionSave(res, store)

	// Simulate second request with Cookie
	r2 := flow.NewRequest(nil)
	r2.CookieHandler = flow.ParseCookieHandler()
	r2.Set("cookie_sess", sessID)

	store2 := NewStore("think_sess", &FileHandler{
		Path:     filepath.Join(tmpDir, "sessions"),
		Lifetime: time.Hour,
	})
	store2.SetId(sessID)
	store2.Start()
	assert.Equal(t, float64(1001), store2.Get("user_id"))
}

func TestSessionFlashExtensionAndToken(t *testing.T) {
	store := NewStore("sess", &CookieHandler{})
	store.Start()

	// Test Token
	tok1 := store.Token()
	assert.NotEmpty(t, tok1)
	assert.Equal(t, tok1, store.Token())

	tok2 := store.RegenerateToken()
	assert.NotEqual(t, tok1, tok2)
	assert.Equal(t, tok2, store.Token())

	// Test Flash & Now & Reflash
	store.Flash("status", "success")
	store.Now("temp", "only_now")
	assert.Equal(t, "success", store.Get("status"))
	assert.Equal(t, "only_now", store.Get("temp"))

	// Reflash preserves flash data
	store.Reflash()
	assert.Equal(t, "success", store.Get("status"))

	// Keep
	store.Keep("status")
	assert.Equal(t, "success", store.Get("status"))
}

func TestSessionOnlyExceptPreviousAndIncrement(t *testing.T) {
	store := NewStore("sess", &CookieHandler{})
	store.Start()

	store.Set("a", 1)
	store.Set("b", 2)
	store.Set("c", 3)

	// 1. Test Only
	only := store.Only("a", "c")
	assert.Len(t, only, 2)
	assert.Equal(t, 1, only["a"])
	assert.Equal(t, 3, only["c"])

	// 2. Test Except
	except := store.Except("b")
	assert.Equal(t, 1, except["a"])
	assert.Equal(t, 3, except["c"])
	assert.Nil(t, except["b"])

	// 3. Test PreviousUrl
	store.SetPreviousUrl("https://example.com/prev")
	assert.Equal(t, "https://example.com/prev", store.PreviousUrl())

	// 4. Test Increment & Decrement
	assert.Equal(t, 1, store.Increment("counter"))
	assert.Equal(t, 4, store.Increment("counter", 3))
	assert.Equal(t, 3, store.Decrement("counter"))
	assert.Equal(t, 1, store.Decrement("counter", 2))

	// 5. Test Invalidate
	oldId := store.GetId()
	store.Invalidate()
	assert.NotEqual(t, oldId, store.GetId())
	assert.Nil(t, store.Get("a"))
}
