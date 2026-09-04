package middleware

import (
	"fmt"
	"runtime"

	"github.com/go-think/think/flow"
	"github.com/go-think/think/contract"
	"github.com/go-think/think/exception"
	"github.com/go-think/think/facades"
)

type RecoverHandler struct {
	debug bool
}

// NewRecoverHandler The default NewRecoverHandler
func NewRecoverHandler(debug bool) Handler {
	return &RecoverHandler{
		debug: debug,
	}
}

// Process Process the request to a router and return the response.
func (h *RecoverHandler) Process(req *flow.Request, next Closure) (result interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if facades.App != nil {
				handler := facades.Container().Make[contract.ExceptionHandler]()
				if handler != nil {
					handler.Report(err)
					result = handler.Render(err)
					return
				}
			}

			// Fallback if ExceptionHandler is not configured
			if he, ok := err.(*exception.HttpException); ok {
				response := flow.NewResponse()
				response.SetCode(he.Code)
				response.SetContent(he.Message)
				result = response
				return
			}
			
			var stacktrace string
			for i := 1; ; i++ {
				_, f, l, got := runtime.Caller(i)
				if !got {
					break
				}
				stacktrace += fmt.Sprintf("%s:%d\n", f, l)
			}

			logMessage := fmt.Sprintf("Trace: %s\n\n%s", err, stacktrace)
			response := flow.ErrorResponse()
			if h.debug {
				response.SetContent(logMessage)
			} else {
				response.SetContent("Internal Server Error")
			}
			result = response
		}
	}()

	return next(req)
}
