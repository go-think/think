package thinkgo

import (
	"fmt"
	"net/http/httputil"
	"runtime"
	"strings"

	"github.com/go-think/think/context"
)

type RecoverHandler struct {
	app *Application
}

// NewRecoverHandler The default NewRecoverHandler
func NewRecoverHandler(app *Application) Handler {
	return &RecoverHandler{
		app: app,
	}
}

// Process Process the request to a router and return the response.
func (h *RecoverHandler) Process(req *context.Request, next Closure) (result interface{}) {
	defer func() {
		if err := recover(); err != nil {
			var stacktrace string
			for i := 1; ; i++ {
				_, f, l, got := runtime.Caller(i)
				if !got {
					break
				}

				stacktrace += fmt.Sprintf("%s:%d\n", f, l)
			}

			httpRequest, _ := httputil.DumpRequest(req.Request, false)

			headers := strings.Split(string(httpRequest), "\r\n")
			for idx, header := range headers {
				current := strings.Split(header, ":")
				if current[0] == "Authorization" {
					headers[idx] = current[0] + ": *"
				}
			}

			logMessage := fmt.Sprintf("Recovered at Request: %s\n", strings.Join(headers, "\r\n"))
			logMessage += fmt.Sprintf("Trace: %s\n", err)
			logMessage += fmt.Sprintf("\n%s", stacktrace)

			response := context.ErrorResponse()
			if h.app.Debug {
				response.SetContent(logMessage)
			}
			result = response
		}
	}()

	return next(req)
}
