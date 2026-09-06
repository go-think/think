package exception

import (
	"testing"

	"github.com/go-think/flow"
	"github.com/stretchr/testify/assert"
)

func TestRenderHttpErrorKeepsUserFacingMessage(t *testing.T) {
	h := NewHandler(nil).(*Handler)

	resp := h.Render(&HttpException{Code: 404, Message: "Not Found"}).(*flow.Response)
	assert.Equal(t, 404, resp.GetCode())
	assert.Equal(t, "Not Found", resp.GetContent())
}

func TestRenderHidesInternalErrorOutsideDebug(t *testing.T) {
	h := NewHandler(nil).(*Handler)
	h.SetDebugResolver(func() bool { return false })

	resp := h.Render("sql failure at /db/secret.go:42").(*flow.Response)
	assert.Equal(t, 500, resp.GetCode())
	assert.Equal(t, "Internal Server Error", resp.GetContent())
}

func TestRenderShowsInternalErrorInDebug(t *testing.T) {
	h := NewHandler(nil).(*Handler)
	h.SetDebugResolver(func() bool { return true })

	resp := h.Render("sql failure at /db/secret.go:42").(*flow.Response)
	assert.Equal(t, 500, resp.GetCode())
	assert.Contains(t, resp.GetContent(), "sql failure")
}

func TestRenderDefaultsToProductionWithoutResolver(t *testing.T) {
	h := NewHandler(nil).(*Handler)

	resp := h.Render("secret detail").(*flow.Response)
	assert.Equal(t, "Internal Server Error", resp.GetContent())
}

func TestCustomRenderTakesPrecedence(t *testing.T) {
	h := NewHandler(nil).(*Handler)
	h.SetDebugResolver(func() bool { return false })
	h.RenderUsing(func(err interface{}) interface{} {
		return flow.NewResponse().SetCode(503).SetContent("custom")
	})

	resp := h.Render("anything").(*flow.Response)
	assert.Equal(t, 503, resp.GetCode())
	assert.Equal(t, "custom", resp.GetContent())
}
