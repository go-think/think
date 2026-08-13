package pipeline

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-think/think/context"
	"github.com/go-think/think/middleware"
	"github.com/stretchr/testify/assert"
)

type dummyHandler struct {
	fn func(req *context.Request, next middleware.Closure) interface{}
}

func (d *dummyHandler) Process(req *context.Request, next middleware.Closure) interface{} {
	return d.fn(req, next)
}

func TestPipelineServeHTTPPointerResponse(t *testing.T) {
	p := NewPipeline()
	p.Pipe(&dummyHandler{
		fn: func(req *context.Request, next middleware.Closure) interface{} {
			resp := context.NewResponse()
			resp.SetCode(http.StatusCreated)
			resp.SetContentType("application/json")
			resp.SetContent(`{"status":"created"}`)
			return resp
		},
	})

	rec := httptest.NewRecorder()
	httpReq, err := http.NewRequest("POST", "/api/item", nil)
	assert.NoError(t, err)

	p.ServeHTTP(rec, httpReq)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, `{"status":"created"}`, rec.Body.String())
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}
