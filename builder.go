package think

import (
	"github.com/go-think/flow"
	"github.com/go-think/think/console"
	"github.com/go-think/think/contract"
	"github.com/go-think/think/exception"
	thinkHttp "github.com/go-think/think/http"

	"github.com/go-think/think/support"
	"github.com/go-think/think/support/env"
)

// Routing defines the routing callbacks and health check endpoints for ApplicationBuilder.
type Routing struct {
	Web       func(r flow.Router)
	Api       func(r flow.Router)
	ApiPrefix string // Prefix for API routes (default: "api", use "/" or "none" for root)
	Health    string // Path to health check endpoint (e.g. "/up")
}

// MiddlewareConfig provides fluent middleware configuration.
type MiddlewareConfig struct {
	kernel contract.HttpKernel
}

// Use adds one or more global middlewares to the application pipeline.
func (m *MiddlewareConfig) Use(middleware ...interface{}) *MiddlewareConfig {
	if m.kernel != nil {
		m.kernel.AddGlobalMiddleware(middleware...)
	}
	return m
}

// Group defines a named middleware group (e.g. "web", "api").
func (m *MiddlewareConfig) Group(name string, middlewares ...interface{}) *MiddlewareConfig {
	if m.kernel != nil {
		m.kernel.AddMiddlewareGroup(name, middlewares)
	}
	return m
}

// Alias registers a route-specific middleware alias (e.g. "auth").
func (m *MiddlewareConfig) Alias(name string, middleware interface{}) *MiddlewareConfig {
	if m.kernel != nil {
		m.kernel.AddRouteMiddleware(name, middleware)
	}
	return m
}

// Cors registers the standard CORS middleware globally.
func (m *MiddlewareConfig) Cors(config ...flow.CorsConfig) *MiddlewareConfig {
	return m.Use(flow.NewCorsMiddleware(config...))
}

// TrimStrings registers the parameter trimming middleware globally.
func (m *MiddlewareConfig) TrimStrings(except ...string) *MiddlewareConfig {
	return m.Use(flow.NewTrimStringsMiddleware(except...))
}

// ValidateSignatures registers the URL signature validation middleware.
func (m *MiddlewareConfig) ValidateSignatures() *MiddlewareConfig {
	return m.Use(flow.NewValidateSignatureMiddleware())
}

// ExceptionsConfig provides custom exception reporting and rendering callbacks.
type ExceptionsConfig struct {
	handler contract.ExceptionHandler
}

// Report registers a custom exception reporting callback.
func (e *ExceptionsConfig) Report(callback func(err interface{}) bool) *ExceptionsConfig {
	if h, ok := e.handler.(*exception.Handler); ok {
		h.ReportUsing(callback)
	}
	return e
}

// Render registers a custom exception rendering callback.
func (e *ExceptionsConfig) Render(callback func(err interface{}) interface{}) *ExceptionsConfig {
	if h, ok := e.handler.(*exception.Handler); ok {
		h.RenderUsing(callback)
	}
	return e
}

// DontReport registers exception types that should not be reported.
func (e *ExceptionsConfig) DontReport(errTypes ...interface{}) *ExceptionsConfig {
	if h, ok := e.handler.(*exception.Handler); ok {
		h.DontReport(errTypes...)
	}
	return e
}

// ApplicationBuilder configures and builds a new Think Application instance.
type ApplicationBuilder struct {
	app *Application
}

// Configure begins configuring a new Think Application instance.
func Configure(basePath ...string) *ApplicationBuilder {
	app := New(basePath...)

	// Bootstrap environment variables before configuring components
	app.bootstrapEnvironmentVariables()

	builder := &ApplicationBuilder{
		app: app,
	}

	return builder.
		WithKernels().
		WithEvents().
		WithCommands().
		WithProviders()
}

// WithKernels registers the standard HTTP and Console kernels into the application.
func (b *ApplicationBuilder) WithKernels() *ApplicationBuilder {
	// 1. Register HTTP Kernel
	if b.app.Make[contract.HttpKernel]() == nil {
		kernel := thinkHttp.NewKernel(b.app.Container)
		b.app.Instance[contract.HttpKernel](kernel)

		flow.HandleException = func(err interface{}) *flow.Response {
			if handler := b.app.Make[contract.ExceptionHandler](); handler != nil {
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

		// Add global recover middleware to Kernel automatically
		kernel.AddGlobalMiddleware(flow.NewRecoverMiddleware(true))
	}

	// 2. Register Console Kernel
	if b.app.Make[contract.ConsoleKernel]() == nil {
		consoleKernel := console.NewKernel(b.app.Container)
		b.app.Instance[contract.ConsoleKernel](consoleKernel)
	}

	return b
}

// WithEvents registers application event listeners or discovery configuration.
func (b *ApplicationBuilder) WithEvents(events ...interface{}) *ApplicationBuilder {
	dispatcher := b.app.Make[contract.EventDispatcher]()
	if dispatcher == nil {
		return b
	}

	for _, item := range events {
		if item == nil {
			continue
		}
		switch v := item.(type) {
		case func(contract.EventDispatcher):
			v(dispatcher)
		case map[string]interface{}:
			for evt, listener := range v {
				dispatcher.Listen(evt, listener)
			}
		case map[string][]interface{}:
			for evt, listeners := range v {
				for _, listener := range listeners {
					dispatcher.Listen(evt, listener)
				}
			}
		}
	}

	return b
}

// WithCommands registers additional console commands or command discovery.
func (b *ApplicationBuilder) WithCommands(commands ...interface{}) *ApplicationBuilder {
	return b
}

// WithRouting registers application route definitions and optional health check.
func (b *ApplicationBuilder) WithRouting(routing Routing) *ApplicationBuilder {
	r := b.app.Make[flow.Router]()
	if r == nil {
		return b
	}

	// 1. Register Web routes (can apply web middleware group)
	if routing.Web != nil {
		routing.Web(r)
	}

	// 2. Register API routes (defaults to "api" prefix if not specified)
	if routing.Api != nil {
		prefix := routing.ApiPrefix
		if prefix == "" {
			prefix = "api"
		}
		if prefix == "/" || prefix == "none" {
			routing.Api(r)
		} else {
			r.Prefix(prefix).Group(func(group flow.Router) {
				routing.Api(group)
			})
		}
	}

	// 3. Register Health probe endpoint (e.g. /up)
	if routing.Health != "" {
		healthPath := routing.Health
		if healthPath[0] != '/' {
			healthPath = "/" + healthPath
		}
		r.Get(healthPath, func(req *flow.Request) interface{} {
			return flow.NewResponse().SetContent("UP").SetCode(200)
		})
	}

	return b
}

// WithMiddleware configures application middleware stack (global, groups, aliases).
func (b *ApplicationBuilder) WithMiddleware(callback func(m *MiddlewareConfig)) *ApplicationBuilder {
	if callback == nil {
		return b
	}

	kernel := b.app.Make[contract.HttpKernel]()
	if kernel == nil {
		return b
	}

	config := &MiddlewareConfig{
		kernel: kernel,
	}

	callback(config)

	return b
}

// WithExceptions configures application exception reporting and rendering.
func (b *ApplicationBuilder) WithExceptions(callbacks ...func(e *ExceptionsConfig)) *ApplicationBuilder {
	// Register ExceptionHandler singleton
	if b.app.Make[contract.ExceptionHandler]() == nil {
		b.app.Singleton[contract.ExceptionHandler](func() contract.ExceptionHandler {
			logger := b.app.Make[contract.Logger]()
			return exception.NewHandler(logger)
		})
	}

	if len(callbacks) > 0 && callbacks[0] != nil {
		handler := b.app.Make[contract.ExceptionHandler]()
		if handler != nil {
			config := &ExceptionsConfig{
				handler: handler,
			}
			callbacks[0](config)
		}
	}

	return b
}

// WithEnvFile specifies the environment file to load during bootstrapping.
func (b *ApplicationBuilder) WithEnvFile(file string) *ApplicationBuilder {
	b.app.LoadEnvironmentFrom(file)
	_ = env.Overload(b.app.EnvironmentFilePath())
	if envVal := env.Get("APP_ENV"); envVal != "" {
		b.app.SetEnvironment(envVal)
	}
	return b
}

// WithProviders registers additional application service providers.
func (b *ApplicationBuilder) WithProviders(providers ...support.ServiceProvider) *ApplicationBuilder {
	for _, p := range providers {
		b.app.Register(p)
	}
	return b
}

// WithBindings registers an array of container bindings.
func (b *ApplicationBuilder) WithBindings(bindings map[string]interface{}) *ApplicationBuilder {
	for k, v := range bindings {
		b.app.Bind[any](v, k)
	}
	return b
}

// WithSingletons registers an array of container singletons.
func (b *ApplicationBuilder) WithSingletons(singletons map[string]interface{}) *ApplicationBuilder {
	for k, v := range singletons {
		b.app.Singleton[any](v, k)
	}
	return b
}

// Create builds and returns the final configured Think Application instance.
func (b *ApplicationBuilder) Create() *Application {
	return b.app
}
