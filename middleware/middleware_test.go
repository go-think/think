package middleware

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-think/think/flow"
	"github.com/go-think/think/router"
	"github.com/stretchr/testify/assert"
)

func TestCorsHandler_PreflightAndNormal(t *testing.T) {
	cors := NewCorsHandler()

	// 1. Preflight Request
	httpReq, _ := http.NewRequest("OPTIONS", "/api/data", nil)
	httpReq.Header.Set("Origin", "http://example.com")
	req := flow.NewRequest(httpReq)

	res := cors.Process(req, func(r *flow.Request) interface{} {
		return flow.NewResponse().SetContent("ok")
	})

	resp, ok := res.(*flow.Response)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNoContent, resp.GetCode())
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))

	// 2. Normal Request
	httpReq2, _ := http.NewRequest("GET", "/api/data", nil)
	httpReq2.Header.Set("Origin", "http://example.com")
	req2 := flow.NewRequest(httpReq2)

	res2 := cors.Process(req2, func(r *flow.Request) interface{} {
		return flow.NewResponse().SetContent("data")
	})

	resp2, ok := res2.(*flow.Response)
	assert.True(t, ok)
	assert.Equal(t, "data", resp2.GetContent())
	assert.Equal(t, "*", resp2.Header.Get("Access-Control-Allow-Origin"))
}

func TestTrimStringsHandler(t *testing.T) {
	trim := NewTrimStringsHandler("password")

	httpReq, _ := http.NewRequest("POST", "/login?username=%20alice%20&password=%20secret%20", nil)
	req := flow.NewRequest(httpReq)

	_ = trim.Process(req, func(r *flow.Request) interface{} {
		return nil
	})

	val, _ := req.Input("username")
	assert.Equal(t, "alice", val)

	pwd, _ := req.Input("password")
	assert.Equal(t, " secret ", pwd)
}

func TestValidateSignatureHandler(t *testing.T) {
	r := router.New()
	r.Get("/secret", func() string {
		return "secret-data"
	}).Name("secret.route")
	r.Register()

	signedUrl := r.SignedUrl("secret.route", 10*time.Minute, nil)

	handler := NewValidateSignatureHandler(r)

	// Valid signature
	httpReq, _ := http.NewRequest("GET", signedUrl, nil)
	req := flow.NewRequest(httpReq)
	res := handler.Process(req, func(r *flow.Request) interface{} {
		return flow.NewResponse().SetContent("passed")
	})
	resp, ok := res.(*flow.Response)
	assert.True(t, ok)
	assert.Equal(t, "passed", resp.GetContent())

	// Invalid signature
	httpReqBad, _ := http.NewRequest("GET", "/secret?signature=fake&expires=9999999999", nil)
	reqBad := flow.NewRequest(httpReqBad)
	resBad := handler.Process(reqBad, func(r *flow.Request) interface{} {
		return flow.NewResponse().SetContent("passed")
	})
	respBad, ok := resBad.(*flow.Response)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, respBad.GetCode())
}
