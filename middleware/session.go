package middleware

import (
	"time"

	"github.com/go-think/think/facades"
	"github.com/go-think/think/flow"
	"github.com/go-think/think/session"
)

type SessionHandler struct {
	Manager *session.Manager
}

// NewSessionHandler creates the default SessionHandler by reading configuration from repository.
func NewSessionHandler() Handler {
	handler := &SessionHandler{}
	
	driver := "file"
	cookieName := "think_session"
	lifetime := 120 * time.Minute
	encrypt := false
	files := "storage/framework/sessions"

	if facades.App != nil {
		if cfg := facades.Config(); cfg != nil {
			if d := cfg.GetString("session.driver"); d != "" {
				driver = d
			}
			if cn := cfg.GetString("session.cookie"); cn != "" {
				cookieName = cn
			} else if cn := cfg.GetString("session.cookie_name"); cn != "" {
				cookieName = cn
			}
			if lt := cfg.GetInt("session.lifetime"); lt > 0 {
				lifetime = time.Duration(lt) * time.Minute
			}
			if cfg.Has("session.encrypt") {
				encrypt = cfg.GetBool("session.encrypt")
			}
			if f := cfg.GetString("session.files"); f != "" {
				files = f
			}
		}
	}

	handler.Manager = session.NewManager(&session.Config{
		Driver:     driver,
		CookieName: cookieName,
		Lifetime:   lifetime,
		Encrypt:    encrypt,
		Files:      files,
	})

	return handler
}

func (h *SessionHandler) Process(req *flow.Request, next Closure) interface{} {
	store := h.startSession(req)

	req.SetSession(store)

	result := next(req)

	if res, ok := result.(session.Response); ok {
		h.saveSession(res, store)
	}

	return result
}

func (h *SessionHandler) startSession(req *flow.Request) *session.Store {
	return h.Manager.SessionStart(req)
}

func (h *SessionHandler) saveSession(res session.Response, store *session.Store) {
	h.Manager.SessionSave(res, store)
}
