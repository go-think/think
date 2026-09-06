package exception

import (
	"fmt"
	"reflect"

	"github.com/go-think/flow"
	"github.com/go-think/think/contract"
)

type Handler struct {
	logger       contract.Logger
	customReport func(err interface{}) bool
	customRender func(err interface{}) interface{}
	dontReport   []reflect.Type
	debug        func() bool
}

func NewHandler(logger contract.Logger) contract.ExceptionHandler {
	return &Handler{
		logger:     logger,
		dontReport: make([]reflect.Type, 0),
	}
}

// SetDebugResolver injects the debug flag lookup (typically app.IsDebug).
// When unset, rendering falls back to the production-safe behavior.
func (h *Handler) SetDebugResolver(fn func() bool) {
	h.debug = fn
}

// DontReport registers exception types that should not be reported.
func (h *Handler) DontReport(errTypes ...interface{}) {
	for _, t := range errTypes {
		if t != nil {
			h.dontReport = append(h.dontReport, reflect.TypeOf(t))
		}
	}
}

// ShouldReport determines if the exception should be reported.
func (h *Handler) ShouldReport(err interface{}) bool {
	if err == nil {
		return false
	}
	errType := reflect.TypeOf(err)
	for _, dont := range h.dontReport {
		if errType == dont {
			return false
		}
	}
	return true
}

// ReportUsing registers a custom reporting callback. If returns true, default report is skipped.
func (h *Handler) ReportUsing(callback func(err interface{}) bool) {
	h.customReport = callback
}

// RenderUsing registers a custom rendering callback. If returns non-nil, default render is skipped.
func (h *Handler) RenderUsing(callback func(err interface{}) interface{}) {
	h.customRender = callback
}

func (h *Handler) Report(err interface{}) {
	if !h.ShouldReport(err) {
		return
	}

	if h.customReport != nil {
		if handled := h.customReport(err); handled {
			return
		}
	}

	if h.logger != nil {
		h.logger.Error("Panic Recovered: %v", err)
	} else {
		fmt.Printf("Panic Recovered: %v\n", err)
	}
}

func (h *Handler) Render(err interface{}) interface{} {
	if h.customRender != nil {
		if res := h.customRender(err); res != nil {
			return res
		}
	}

	resp := flow.NewResponse()
	if e, ok := err.(*HttpException); ok {
		// HTTP exceptions carry user-facing messages by design.
		resp.SetCode(e.Code)
		resp.SetContent(e.Message)
		return resp
	}

	resp.SetCode(500)
	if h.debug != nil && h.debug() {
		// Detailed content is for debug environments only; the full error is
		// already reported to the log by Report().
		resp.SetContent(fmt.Sprintf("Internal Server Error: %v", err))
		return resp
	}
	resp.SetContent("Internal Server Error")
	return resp
}
