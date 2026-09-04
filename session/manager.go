package session

import (
	"sync"
	"time"
)

var (
	customHandlers map[string]Handler
	customHandlersMu sync.RWMutex
)

type Config struct {
	// Default Session Driver
	Driver string

	CookieName string

	// Session Lifetime
	Lifetime time.Duration

	// Session Encryption
	Encrypt bool

	// Session File Location
	Files string
}

type Manager struct {
	Config *Config
}

func NewManager(config *Config) *Manager {
	m := &Manager{
		Config: config,
	}

	return m
}

func (m *Manager) SessionStart(req Request) *Store {
	storeHandler := m.parseStoreHandler()
	store := NewStore(m.Config.CookieName, storeHandler)

	if handler, ok := storeHandler.(*CookieHandler); ok {
		handler.SetRequest(req)
	}

	cookieId, _ := req.Cookie(store.GetName())
	store.SetId(cookieId)
	store.Start()
	return store
}

func (m *Manager) SessionSave(res Response, store *Store) {
	if store == nil {
		return
	}
	if handler, ok := store.GetHandler().(*CookieHandler); ok {
		handler.SetResponse(res)
	}
	_ = res.Cookie(store.GetName(), store.GetId())
	store.Save()
}

func Extend(driver string, handler Handler) {
	customHandlersMu.Lock()
	defer customHandlersMu.Unlock()
	if customHandlers == nil {
		customHandlers = make(map[string]Handler)
	}
	customHandlers[driver] = handler
}

func (m *Manager) parseStoreHandler() Handler {
	var storeHandler Handler
	switch m.Config.Driver {
	case "cookie":
		storeHandler = &CookieHandler{}
	case "file":
		storeHandler = &FileHandler{
			Path:     m.Config.Files,
			Lifetime: m.Config.Lifetime,
		}
	default:
		customHandlersMu.RLock()
		handler, ok := customHandlers[m.Config.Driver]
		customHandlersMu.RUnlock()
		if !ok {
			// Fallback to default file driver to prevent direct panic
			storeHandler = &FileHandler{
				Path:     m.Config.Files,
				Lifetime: m.Config.Lifetime,
			}
		} else {
			storeHandler = handler
		}
	}

	return storeHandler
}

