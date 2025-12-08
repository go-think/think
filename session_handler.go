package thinkgo

import (
	"github.com/go-think/think/config"
	"github.com/go-think/think/context"
	"github.com/go-think/think/session"
)

type SessionHandler struct {
	Manager *session.Manager
	app     *Application
}

// SessionHandler The default SessionHandler
func NewSessionHandler(app *Application) Handler {
	handler := &SessionHandler{}
	handler.Manager = session.NewManager(&session.Config{
		Driver:     config.Session.Driver,
		CookieName: config.Session.CookieName,
		Lifetime:   config.Session.Lifetime,
		Encrypt:    config.Session.Encrypt,
		Files:      config.Session.Files,
	})

	handler.app = app

	return handler
}

func (h *SessionHandler) Process(req *context.Request, next Closure) interface{} {
	store := h.startSession(req)

	req.SetSession(store)

	result := next(req)

	if res, ok := result.(session.Response); ok {
		h.saveSession(res)
	}

	return result
}

func (h *SessionHandler) startSession(req *context.Request) *session.Store {
	return h.Manager.SessionStart(req)
}

func (h *SessionHandler) saveSession(res session.Response) {
	h.Manager.SessionSave(res)
}
