package middleware

import (
	"strings"

	"github.com/go-think/think/flow"
)

type TrimStringsHandler struct {
	except []string
}

// NewTrimStringsHandler creates a new parameter trimming middleware handler.
func NewTrimStringsHandler(except ...string) Handler {
	return &TrimStringsHandler{except: except}
}

func (h *TrimStringsHandler) Process(req *flow.Request, next Closure) interface{} {
	allInput := req.All()
	if len(allInput) > 0 {
		cleaned := make(map[string]string, len(allInput))
		for k, v := range allInput {
			if h.isExcepted(k) {
				continue
			}
			cleaned[k] = strings.TrimSpace(v)
		}
		req.Merge(cleaned)
	}

	return next(req)
}

// CleanValue recursively trims strings in nested maps and slices.
func CleanValue(val interface{}) interface{} {
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]interface{}:
		res := make(map[string]interface{}, len(v))
		for k, item := range v {
			res[k] = CleanValue(item)
		}
		return res
	case []interface{}:
		res := make([]interface{}, len(v))
		for i, item := range v {
			res[i] = CleanValue(item)
		}
		return res
	default:
		return v
	}
}

func (h *TrimStringsHandler) isExcepted(key string) bool {
	for _, ex := range h.except {
		if ex == key {
			return true
		}
	}
	return false
}
