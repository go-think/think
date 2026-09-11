package think

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-think/flow"
	"github.com/go-think/think/console"
	"github.com/go-think/think/contract"
	"github.com/go-think/think/exception"
	"github.com/go-think/think/helper"
	thinkHttp "github.com/go-think/think/http"
)

// HttpKernel gets the HttpKernel instance from the container (auto-binds default if not registered).
func (a *Application) HttpKernel() contract.HttpKernel {
	if kernel := a.Make[contract.HttpKernel](); kernel != nil {
		return kernel
	}

	// Auto-bind default HTTP Kernel if not yet registered
	kernel := thinkHttp.NewKernel(a.Container)

	flow.HandleException = func(err interface{}) *flow.Response {
		if handler := a.Make[contract.ExceptionHandler](); handler != nil {
			handler.Report(err)
			if res, ok := handler.Render(err).(*flow.Response); ok {
				return res
			}
		}
		if he, ok := err.(*exception.HttpException); ok {
			response := flow.NewResponse()
			response.SetCode(he.Code)
			response.SetContent(he.Message)
			return response
		}
		return nil
	}
	kernel.AddGlobalMiddleware(flow.NewRecoverMiddleware(true))
	a.Instance[contract.HttpKernel](kernel)
	return kernel
}

// ConsoleKernel gets the ConsoleKernel instance from the container (auto-binds default if not registered).
func (a *Application) ConsoleKernel() contract.ConsoleKernel {
	if kernel := a.Make[contract.ConsoleKernel](); kernel != nil {
		return kernel
	}

	// Auto-bind default Console Kernel if not yet registered
	kernel := console.NewKernel(a.Container, a.routeDump())
	a.Instance[contract.ConsoleKernel](kernel)
	return kernel
}

// HandleCommand handles an incoming CLI console command.
func (a *Application) HandleCommand(args ...string) int {
	kernel := a.ConsoleKernel()
	status := kernel.Handle(args...)
	kernel.Terminate(status)
	return status
}

// BuildServer builds the standard HTTP Server instance for this application.
func (a *Application) BuildServer(params ...string) *http.Server {
	a.Boot()
	addrs := helper.ParseAddr(params...)

	kernel := a.HttpKernel()
	// Warm up application & kernel in persistent memory
	kernel.Bootstrap()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		kernel.ServeHTTP(w, r)
	})

	a.server = &http.Server{
		Addr:    addrs,
		Handler: handler,
	}
	return a.server
}

// Shutdown gracefully shuts down the application server.
func (a *Application) Shutdown(ctx context.Context) error {
	if a.server != nil {
		return a.server.Shutdown(ctx)
	}
	return nil
}

// Run boots the application and runs the HTTP server with graceful shutdown support.
func (a *Application) Run(params ...string) {
	// 1. Boot all service providers
	a.Boot()

	srv := a.BuildServer(params...)

	logger := a.Make[contract.Logger]()
	r := a.Make[flow.Router]()

	if logger != nil && r != nil {
		logger.Debug("\r\nLoaded routes:\r\n%s", string(r.Dump()))
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		select {
		case <-sigint:
			if logger != nil {
				logger.Debug("Shutting down Think application server gracefully...")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := a.Shutdown(ctx); err != nil {
				if logger != nil {
					logger.Debug("Application Server Shutdown Error: %v", err)
				}
			}
		case <-idleConnsClosed:
			signal.Stop(sigint)
			return
		}
	}()

	if logger != nil {
		logger.Debug("Think application server running on http://%s", srv.Addr)
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		if logger != nil {
			logger.Error("HTTP server ListenAndServe error: %v", err)
		} else {
			fmt.Printf("HTTP server ListenAndServe error: %v\n", err)
		}
	}

	// Close channel to notify listener and complete shutdown
	close(idleConnsClosed)
	if logger != nil {
		logger.Debug("Think application server stopped gracefully.")
	}
}
