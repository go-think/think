package router

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/go-think/think/flow"
	"github.com/stretchr/testify/assert"
)

func TestConcurrentRouteMatchingParamIsolation(t *testing.T) {
	r := New()
	r.Get("/user/{id}", func(req *flow.Request, id string) string {
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

			req := flow.NewRequest(httpReq)
			rule, params, err := r.MatchRequest(req)
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
	req := flow.NewRequest(httpReq)
	rule, params, err := r.MatchRequest(req)
	assert.NoError(t, err)
	assert.Equal(t, "pong", rule.Run(req, params))
}

func TestRouteWhereConstraints(t *testing.T) {
	r := New()
	// Only allow numeric id
	r.Get("/items/{id}", func(id int) string {
		return fmt.Sprintf("item:%d", id)
	}).WhereNumber("id")

	// Only allow enum role
	r.Get("/roles/{role}", func(role string) string {
		return "role:" + role
	}).WhereIn("role", []string{"admin", "editor"})

	r.Register()

	// 1. Valid numeric id -> should match
	httpReq1, _ := http.NewRequest("GET", "/items/123", nil)
	req1 := flow.NewRequest(httpReq1)
	rule1, params1, err1 := r.MatchRequest(req1)
	assert.NoError(t, err1)
	assert.NotNil(t, rule1)
	assert.Equal(t, "item:123", rule1.Run(req1, params1))

	// 2. Invalid alpha id -> should NOT match
	httpReq2, _ := http.NewRequest("GET", "/items/abc", nil)
	req2 := flow.NewRequest(httpReq2)
	_, _, err2 := r.MatchRequest(req2)
	assert.Error(t, err2)

	// 3. Valid role enum -> should match
	httpReq3, _ := http.NewRequest("GET", "/roles/admin", nil)
	req3 := flow.NewRequest(httpReq3)
	rule3, params3, err3 := r.MatchRequest(req3)
	assert.NoError(t, err3)
	assert.Equal(t, "role:admin", rule3.Run(req3, params3))

	// 4. Invalid role enum -> should NOT match
	httpReq4, _ := http.NewRequest("GET", "/roles/guest", nil)
	req4 := flow.NewRequest(httpReq4)
	_, _, err4 := r.MatchRequest(req4)
	assert.Error(t, err4)
}

func TestSignedUrlAndSignatureValidation(t *testing.T) {
	r := New()
	r.Get("/download/{file}", func(file string) string {
		return "download:" + file
	}).Name("file.download")
	r.Register()

	// Generate signed url valid for 1 hour
	signedUrl := r.SignedUrl("file.download", 1*time.Hour, map[string]string{"file": "report.pdf"})
	assert.Contains(t, signedUrl, "/download/report.pdf?")
	assert.Contains(t, signedUrl, "expires=")
	assert.Contains(t, signedUrl, "signature=")

	// Validate valid request
	httpReq, _ := http.NewRequest("GET", signedUrl, nil)
	req := flow.NewRequest(httpReq)
	assert.True(t, r.HasValidSignature(req))

	// Tampered request -> should be invalid
	tamperedUrl := signedUrl + "x"
	httpReqTampered, _ := http.NewRequest("GET", tamperedUrl, nil)
	reqTampered := flow.NewRequest(httpReqTampered)
	assert.False(t, r.HasValidSignature(reqTampered))
}

func TestRouteHasAndCurrentNameAndIs(t *testing.T) {
	r := New()
	r.Get("/admin/dashboard", func() string {
		return "admin-ok"
	}).Name("admin.dashboard")
	r.Register()

	// 1. Has
	assert.True(t, r.Has("admin.dashboard"))
	assert.False(t, r.Has("admin.users"))

	// 2. CurrentRouteName & Is
	httpReq, _ := http.NewRequest("GET", "/admin/dashboard", nil)
	req := flow.NewRequest(httpReq)
	res := r.Dispatch(req)
	resp, ok := res.(*flow.Response)
	assert.True(t, ok)
	assert.Equal(t, "admin-ok", resp.GetContent())

	assert.Equal(t, "admin.dashboard", r.CurrentRouteName(req))
	assert.True(t, r.Is(req, "admin.dashboard"))
	assert.True(t, r.Is(req, "admin.*"))
	assert.False(t, r.Is(req, "user.*"))
}
