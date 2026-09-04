package middleware

import (
	"net/http"

	"github.com/go-think/think/facades"
	"github.com/go-think/think/flow"
	"github.com/go-think/think/router"
)

type ValidateSignatureHandler struct {
	router *router.Route
}

// NewValidateSignatureHandler creates a new URL signature validation middleware.
func NewValidateSignatureHandler(r ...*router.Route) Handler {
	var targetRouter *router.Route
	if len(r) > 0 && r[0] != nil {
		targetRouter = r[0]
	}
	return &ValidateSignatureHandler{router: targetRouter}
}

func (h *ValidateSignatureHandler) Process(req *flow.Request, next Closure) interface{} {
	r := h.router
	if r == nil {
		if facades.App != nil {
			r = facades.Container().Make[*router.Route]()
		}
	}

	if r != nil {
		if !r.HasValidSignature(req) {
			return flow.NewResponse().
				SetCode(http.StatusForbidden).
				SetContent("Invalid signature.")
		}
	}

	return next(req)
}
