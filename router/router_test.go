package router

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/go-think/think/context"
	"github.com/stretchr/testify/assert"
)

func TestConcurrentRouteMatchingParamIsolation(t *testing.T) {
	r := New()
	r.Get("/user/{id}", func(req *context.Request, id string) string {
		return "user:" + id
	})
	r.Register()

	var wg sync.WaitGroup
	concurrentCount := 100

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		userID := fmt.Sprintf("id_%d", i)
		go func(idStr string) {
			defer wg.Done()
			httpReq, err := http.NewRequest("GET", "/user/"+idStr, nil)
			assert.NoError(t, err)

			req := context.NewRequest(httpReq)
			rule, params, err := r.Dispatch(req)
			assert.NoError(t, err)
			assert.NotNil(t, rule)

			res := rule.Run(req, params)
			assert.Equal(t, "user:"+idStr, res)
		}(userID)
	}

	wg.Wait()
}

func TestRoutePrefixAndGroup(t *testing.T) {
	r := New()
	r.Get("/ping", func() string {
		return "pong"
	})
	r.Register()

	httpReq, _ := http.NewRequest("GET", "/ping", nil)
	req := context.NewRequest(httpReq)
	rule, params, err := r.Dispatch(req)
	assert.NoError(t, err)
	assert.Equal(t, "pong", rule.Run(req, params))
}
