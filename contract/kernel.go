package contract

// HttpKernel handles HTTP requests and manages middleware.
type HttpKernel interface {
	// Bootstrap bootstraps the application for HTTP requests.
	Bootstrap()
	// Handle handles the request through the middleware stack and returns the response.
	Handle(request interface{}) interface{}
	// AddGlobalMiddleware adds middleware to the global stack.
	AddGlobalMiddleware(middleware ...interface{})
	// AddRouteMiddleware adds middleware to a specific group or aliases it.
	AddRouteMiddleware(name string, middleware interface{})
	// AddMiddlewareGroup adds a group of middlewares under a single key.
	AddMiddlewareGroup(name string, middlewares []interface{})
	// GetMiddlewareGroup retrieves a group of middlewares by its key.
	GetMiddlewareGroup(name string) []interface{}
	// GetRouteMiddleware retrieves a middleware by its alias.
	GetRouteMiddleware(name string) interface{}
	// ServeHTTP implements the standard http.Handler interface.
	ServeHTTP(w interface{}, r interface{})
}

// ConsoleKernel handles CLI console commands.
type ConsoleKernel interface {
	// Bootstrap bootstraps the application for console commands.
	Bootstrap()
	// Handle handles an incoming console command and returns the exit status code.
	Handle(args ...string) int
	// Terminate terminates the application and command.
	Terminate(status int)
}
