package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-think/think/flow"
)

// CorsConfig defines the configuration options for CORS middleware.
type CorsConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCorsConfig returns standard default CORS configuration.
func DefaultCorsConfig() CorsConfig {
	return CorsConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-CSRF-Token"},
		ExposeHeaders:    []string{},
		AllowCredentials: true,
		MaxAge:           86400,
	}
}

type CorsHandler struct {
	config CorsConfig
}

// NewCorsHandler creates a new CORS middleware handler.
func NewCorsHandler(config ...CorsConfig) Handler {
	cfg := DefaultCorsConfig()
	if len(config) > 0 {
		cfg = config[0]
	}
	return &CorsHandler{config: cfg}
}

func (h *CorsHandler) Process(req *flow.Request, next Closure) interface{} {
	origin := req.Header("Origin")

	// If not CORS request, proceed normally
	if origin == "" {
		return next(req)
	}

	allowOrigin := h.determineAllowedOrigin(origin)

	// Preflight request handling
	if req.IsMethod("OPTIONS") {
		res := flow.NewResponse().SetCode(http.StatusNoContent)
		h.applyHeaders(res, allowOrigin)
		return res
	}

	// Actual request handling
	result := next(req)
	if res, ok := result.(*flow.Response); ok {
		h.applyHeaders(res, allowOrigin)
	}

	return result
}

func (h *CorsHandler) determineAllowedOrigin(origin string) string {
	for _, allowed := range h.config.AllowOrigins {
		if allowed == "*" {
			return "*"
		}
		if allowed == origin {
			return origin
		}
	}
	return ""
}

func (h *CorsHandler) applyHeaders(res *flow.Response, allowOrigin string) {
	if allowOrigin != "" {
		res.Header.Set("Access-Control-Allow-Origin", allowOrigin)
	}
	if len(h.config.AllowMethods) > 0 {
		res.Header.Set("Access-Control-Allow-Methods", strings.Join(h.config.AllowMethods, ", "))
	}
	if len(h.config.AllowHeaders) > 0 {
		res.Header.Set("Access-Control-Allow-Headers", strings.Join(h.config.AllowHeaders, ", "))
	}
	if len(h.config.ExposeHeaders) > 0 {
		res.Header.Set("Access-Control-Expose-Headers", strings.Join(h.config.ExposeHeaders, ", "))
	}
	if h.config.AllowCredentials && allowOrigin != "*" {
		res.Header.Set("Access-Control-Allow-Credentials", "true")
	}
	if h.config.MaxAge > 0 {
		res.Header.Set("Access-Control-Max-Age", strconv.Itoa(h.config.MaxAge))
	}
}
