package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-think/flow"
	"github.com/go-think/think/container"
	"github.com/stretchr/testify/assert"
)

type dummyTerminableMiddleware struct {
	terminated  bool
	shouldPanic bool
}

func (d *dummyTerminableMiddleware) Process(req *flow.Request, next flow.Closure) any {
	return next(req)
}

func (d *dummyTerminableMiddleware) Terminate(req *flow.Request, res interface{}) {
	if d.shouldPanic {
		panic("dummy panic in terminate")
	}
	d.terminated = true
}

func TestKernel_MiddlewareAndGroups(t *testing.T) {
	c := container.New()
	r := flow.New()
	c.Instance[flow.Router](r)

	kernel := NewKernel(c)
	k, ok := kernel.(*Kernel)
	assert.True(t, ok)

	// Route Middleware
	k.AddRouteMiddleware("auth", "dummyAuthMiddleware")
	assert.Equal(t, "dummyAuthMiddleware", k.GetRouteMiddleware("auth"))
	assert.Nil(t, k.GetRouteMiddleware("nonexistent"))

	// Middleware Groups
	k.AddMiddlewareGroup("web", []interface{}{"auth"})
	assert.Equal(t, []interface{}{"auth"}, k.GetMiddlewareGroup("web"))
	assert.Nil(t, k.GetMiddlewareGroup("api"))

	// Global Middleware
	k.AddGlobalMiddleware("global1", "global2")
	globals := k.GetGlobalMiddleware()
	assert.Equal(t, []interface{}{"global1", "global2"}, globals)
}

func TestKernel_BootstrapIdempotent(t *testing.T) {
	c := container.New()
	r := flow.New()
	c.Instance[flow.Router](r)

	k := NewKernel(c)
	k.AddRouteMiddleware("test", "testMiddleware")

	// Call Bootstrap multiple times
	k.Bootstrap()
	k.Bootstrap()
}

func TestKernel_ServeHTTP_And_Concurrency(t *testing.T) {
	c := container.New()
	r := flow.New()
	r.Get("/hello", func(req *flow.Request) *flow.Response {
		return flow.NewResponse().SetContent("hello world")
	})
	r.Register()
	c.Instance[flow.Router](r)

	k := NewKernel(c)

	// Concurrent ServeHTTP requests
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/hello", nil)
			w := httptest.NewRecorder()
			k.ServeHTTP(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "hello world", string(body))
		}()
	}
	wg.Wait()
}

func TestKernel_Terminate_PanicRecovery(t *testing.T) {
	c := container.New()
	k := NewKernel(c)

	mid := &dummyTerminableMiddleware{shouldPanic: true}
	k.AddGlobalMiddleware(mid)

	req := flow.NewRequest(httptest.NewRequest("GET", "/", nil))
	// Calling Terminate should not crash even if middleware panics
	if kernelImpl, ok := k.(*Kernel); ok {
		kernelImpl.Terminate(req, "response")
	}

	time.Sleep(50 * time.Millisecond)
}
