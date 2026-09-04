<h1 align="center">
  Think
</h1>

<p align="center">
	<strong>Think is an expressive, elegant, and modular Web Framework for Go.</strong>
</p>
<p align="center">
	<a href="https://github.com/go-think/think/actions/workflows/build.yml">
		<img src="https://github.com/go-think/think/actions/workflows/build.yml/badge.svg" alt="Build Status">
  	</a>
  	<a href="https://coveralls.io/github/go-think/think">
        <img src="https://coveralls.io/repos/github/go-think/think/badge.svg" alt="Coverage Status">
    </a>
</p>
<p align="center">
	<a href="https://goreportcard.com/report/github.com/go-think/think">
		<img src="https://goreportcard.com/badge/github.com/go-think/think" alt="Go Report Card">
  	</a>
	<a href="https://codeclimate.com/github/go-think/think/maintainability">
		<img src="https://api.codeclimate.com/v1/badges/c315fda3b07b5aef3529/maintainability" />
	</a>
	<a href="https://godoc.org/github.com/go-think/think">
		<img src="https://godoc.org/github.com/go-think/think?status.svg" alt="GoDoc">
  	</a>
	<a href="https://github.com/go-think/think/releases">
		<img src="https://img.shields.io/github/release/go-think/think.svg" alt="Latest Stable Version">
	</a>
	<a href="LICENSE">
		<img src="https://img.shields.io/github/license/go-think/think.svg" alt="License">
	</a>
</p>

## Requirements

- Go 1.27 or higher

## Installation

```bash
go get -u github.com/go-think/think
```

## Quick Start

Think provides a fluent, modular builder approach to bootstrap and configure your application:

```go
package main

import (
	"fmt"

	"github.com/go-think/think"
	"github.com/go-think/think/flow"
	"github.com/go-think/think/router"
)

func main() {
	app := think.Configure().
		WithRouting(think.Routing{
			Web: func(r *router.Route) {
				r.Get("/", func() *flow.Response {
					return flow.Text("Hello Think!")
				})

				r.Get("/ping", func() *flow.Response {
					return flow.Json(map[string]string{
						"message": "pong",
					})
				})

				// Route parameters & dependency injection
				r.Get("/user/{name}", func(req *flow.Request, name string) *flow.Response {
					return flow.Text(fmt.Sprintf("Hello %s!", name))
				})
			},
			Health: "/up",
		}).
		Create()

	// Listen and serve on 0.0.0.0:9011
	app.Run(":9011")
}
```

Or you can use the classic instantiation style:

```go
package main

import (
	"github.com/go-think/think"
	"github.com/go-think/think/flow"
	"github.com/go-think/think/router"
)

func main() {
	app := think.New()
	app.RegisterRoute(func(r *router.Route) {
		r.Get("/", func() *flow.Response {
			return flow.Text("Hello Think!")
		})
	})
	app.Run(":9011")
}
```

---

## Features

- [Routing](#routing)
  - [Basic Routing](#basic-routing)
  - [Route Verbs](#route-verbs)
  - [Route Parameters](#route-parameters)
  - [Parameter Regex Constraints](#parameter-regex-constraints)
  - [Route Prefixes & Groups](#route-prefixes--groups)
  - [Named Routes & URL Generation](#named-routes--url-generation)
  - [Signed URLs](#signed-urls)
- [Middleware](#middleware)
  - [Writing Middleware](#writing-middleware)
  - [Built-in Middlewares](#built-in-middlewares)
- [HTTP Request](#http-request)
  - [Input & Values](#input--values)
  - [Type Safe Conversions](#type-safe-conversions)
  - [Path & Inspection](#path--inspection)
  - [Client Details & Fingerprint](#client-details--fingerprint)
- [HTTP Response](#http-response)
  - [Factory Methods](#factory-methods)
  - [Streaming & Downloads](#streaming--downloads)
  - [Custom Headers & Status Codes](#custom-headers--status-codes)
- [IoC Container & Facades](#ioc-container--facades)
- [Configuration & Environment](#configuration--environment)
- [Events](#events)
- [Exceptions](#exceptions)
- [HTTP Session](#http-session)
- [View](#view)
- [Logging](#logging)
- [License](#license)

---

## Routing

#### Basic Routing

The most basic routes accept a URI and a Closure callback:

```go
r.Get("/foo", func() *flow.Response {
	return flow.Text("Hello Think!")
})
```

Handlers can return `*flow.Response`, `string`, `map`, `struct`, or any serializable type. Think will automatically format the output and set the appropriate `Content-Type`.

#### Route Verbs

The router allows you to register routes for any HTTP verb:

```go
r.Get("/users", getUsers)
r.Post("/users", createUser)
r.Put("/users/{id}", updateUser)
r.Delete("/users/{id}", deleteUser)
r.Patch("/users/{id}", patchUser)
r.Options("/users", optionsHandler)

// Match any HTTP verb
r.Any("/any", anyHandler)
```

#### Route Parameters

Capture URI segments directly into handler parameters:

```go
r.Get("/user/{id}", func(req *flow.Request, id string) *flow.Response {
	return flow.Text(fmt.Sprintf("User ID: %s", id))
})

r.Get("/posts/{post}/comments/{comment}", func(req *flow.Request, post, comment string) *flow.Response {
	return flow.Json(map[string]string{
		"post_id":    post,
		"comment_id": comment,
	})
})
```

#### Parameter Regex Constraints

Constrain the format of your route parameters using `Where` methods:

```go
// Match only digits
r.Get("/user/{id}", getUser).WhereNumber("id")

// Match only alphabetic characters
r.Get("/user/{name}", getUser).WhereAlpha("name")

// Match against allowed enum values
r.Get("/order/{status}", getOrder).WhereIn("status", []string{"pending", "paid", "shipped"})

// Custom regex pattern
r.Get("/user/{code}", getUser).Where("code", "^[A-Z]{3}-[0-9]{4}$")
```

#### Route Prefixes & Groups

Share common middleware or path prefixes across a collection of routes:

```go
import "github.com/go-think/think/contract"

r.Prefix("/admin").Group(func(admin contract.Router) {
	admin.Get("/dashboard", adminDashboard)

	admin.Prefix("/users").Group(func(users contract.Router) {
		users.Get("", listAdminUsers)
		users.Get("/{id}", getAdminUser)
	})
})
```

#### Named Routes & URL Generation

Assign names to routes to conveniently generate URLs:

```go
r.Get("/user/{id}/profile", showProfile).Name("profile")

// Generate URL: "/user/42/profile"
url := r.Url("profile", map[string]string{"id": "42"})
```

#### Signed URLs

Generate tamper-proof URLs protected with a cryptographic HMAC signature:

```go
// Create a signed URL valid for 30 minutes
signedUrl := r.SignedUrl("unsubscribe", 30*time.Minute, map[string]string{"user": "123"})

// Validate signature inside handler or middleware
if !r.HasValidSignature(req) {
	return flow.NewResponse().SetCode(403).SetContent("Invalid or expired signature")
}
```

---

## Middleware

Middleware provide a convenient mechanism for inspecting and filtering HTTP requests entering your application.

#### Writing Middleware

Implement a standard middleware closure:

```go
func AuthMiddleware(req *flow.Request, next middleware.Closure) interface{} {
	token := req.Header("Authorization")
	if token == "" {
		return flow.NewResponse().SetCode(401).SetContent("Unauthorized")
	}

	// Proceed to next middleware or route handler
	return next(req)
}

// Attach to specific route
r.Get("/secret", secretHandler).Middleware(AuthMiddleware)
```

Or implement the `middleware.Handler` interface:

```go
type MyMiddleware struct{}

func (m *MyMiddleware) Process(req *flow.Request, next middleware.Closure) interface{} {
	// Perform action before handler
	res := next(req)
	// Perform action after handler
	return res
}
```

#### Built-in Middlewares

Think includes essential built-in middlewares configurable fluently via `ApplicationBuilder`:

```go
app := think.Configure().
	WithMiddleware(func(m *think.MiddlewareConfig) {
		// Enable standard CORS handling
		m.Cors()

		// Automatically trim incoming request strings
		m.TrimStrings("password", "password_confirmation")

		// Validate signed URL signatures
		m.ValidateSignatures()
	}).
	Create()
```

---

## HTTP Request

Access incoming request data via `*flow.Request`:

#### Input & Values

```go
func Handler(req *flow.Request) *flow.Response {
	// Retrieve from any source (query, json body, post form)
	name, _ := req.Input("name")

	// Query parameters
	page, _ := req.Query("page")

	// POST / JSON body parameters
	email, _ := req.Post("email")

	// All inputs as a map
	all := req.All()

	return flow.Json(all)
}
```

#### Type Safe Conversions

```go
// Returns boolean or default value
isAdmin := req.Boolean("is_admin", false)

// Returns integer or default value
page := req.Integer("page", 1)

// Returns float64 or default value
price := req.Float("price", 0.0)
```

#### Path & Inspection

```go
// Get request path (e.g. "/users/1")
path := req.Path()

// Match path patterns with wildcards
if req.Is("admin/*") {
	// Matches /admin/dashboard, /admin/users, etc.
}

// Match current route name
if req.RouteIs("api.*") {
	// ...
}
```

#### Client Details & Fingerprint

```go
// Client IP address (with X-Forwarded-For support)
ip := req.ClientIP()

// User-Agent header
ua := req.UserAgent()

// SHA-256 fingerprint of the request
fingerprint := req.Fingerprint()
```

---

## HTTP Response

All response helpers are centralized in the `flow` package.

#### Factory Methods

```go
// JSON response (application/json)
flow.Json(map[string]interface{}{"status": "ok", "code": 200})

// Plain text response (text/plain)
flow.Text("Hello World")

// HTML response (text/html)
flow.Html("<h1>Welcome</h1>")

// 204 No Content response
flow.NoContent()

// Dynamic auto-detecting response
flow.MakeResponse(data)
```

#### Streaming & Downloads

```go
// File download attachment
flow.Download("/path/to/report.pdf", "annual_report.pdf")

// Stream response / SSE (Server-Sent Events)
flow.StreamResponse(func(w io.Writer) bool {
	fmt.Fprintf(w, "data: %s\n\n", time.Now().Format(time.RFC3339))
	time.Sleep(1 * time.Second)
	return true // return false to stop streaming
})

// Stream download
flow.StreamDownload(func(w io.Writer) bool {
	w.Write([]byte("chunked-data..."))
	return false
}, "large_file.zip")
```

#### Custom Headers & Status Codes

```go
res := flow.NewResponse().
	SetCode(201).
	SetContentType("application/json").
	SetContent(`{"created": true}`)

res.Header.Set("X-Powered-By", "Think")
res.Cookie("session_id", "xyz123")
```

---

## IoC Container & Facades

Think features a modern IoC Container supporting Go generics:

```go
// Register singleton
app.Singleton[MyService](func() MyService {
	return NewMyService()
})

// Resolve instance with type safety
service := app.Make[MyService]()
```

#### Global Facades

Access core components conveniently from anywhere via `facades`:

```go
import "github.com/go-think/think/facades"

// Router facade
facades.Route().Get("/status", statusHandler)

// Config facade
appName := facades.Config().GetString("app.name")

// Logger facade
facades.Log().Info("Service started")

// Event dispatcher facade
facades.Event().Dispatch("order.created", order)

// Container facade
db := facades.Container().Make[Database]()
```

---

## Configuration & Environment

Think automatically loads environment files on startup with support for `.env`, `.env.local`, and environment-specific files (`.env.production`, `.env.testing`).

#### Environment Detection

Pass `--env=testing` via command line arguments or specify the `APP_ENV` environment variable:

```bash
APP_ENV=production go run main.go
```

#### Configuration Repository

```go
cfg := facades.Config()

// Read values
name := cfg.GetString("app.name")
port := cfg.GetInt("app.port", 9011)
debug := cfg.GetBool("app.debug", false)

// Mutate configuration
cfg.Set("app.timezone", "Asia/Shanghai")
cfg.Push("app.providers", "custom_provider")
```

---

## Events

Decouple your application logic using the event dispatcher:

```go
// Register listener
facades.Event().Listen("user.registered", func(payload interface{}) {
	user := payload.(*User)
	sendWelcomeEmail(user)
})

// Dispatch event
facades.Event().Dispatch("user.registered", user)

// Dispatch until first non-nil response
result := facades.Event().Until("order.validating", order)
```

---

## Exceptions

Configure centralized exception reporting and rendering:

```go
app := think.Configure().
	WithExceptions(func(e *think.ExceptionsConfig) {
		// Ignore specific exception types from error reporting
		e.DontReport(NotFoundError{})

		// Custom reporting callback
		e.Report(func(err interface{}) bool {
			facades.Log().Error(fmt.Sprintf("Caught error: %v", err))
			return true
		})
	}).
	Create()
```

---

## HTTP Session

Manage session data with multiple storage drivers:

```go
// Retrieve session data
user := req.Session().Get("user")

// Store session data
req.Session().Set("user", "alice")

// Flash data (only available in subsequent request)
req.Session().Flash("message", "Task created successfully")

// Reflash all flash data for another request
req.Session().Reflash()

// Invalidate & regenerate session
req.Session().Invalidate()
```

---

## View

Specify your templates directory and render HTML views:

```go
import "github.com/go-think/think/view"

// Parse views directory
view.ParseGlob("views/*")

// Render view inside handler (returns template.HTML, automatically handled by Think)
r.Get("/profile", func(req *flow.Request) interface{} {
	return view.Render("profile.html", map[string]interface{}{
		"Title": "Profile Page",
		"User":  "Alice",
	})
})
```

---

## Logging

Think provides standard RFC 5424 logging levels:

```go
facades.Log().Debug("Debug message")
facades.Log().Info("Information message")
facades.Log().Notice("Notice message")
facades.Log().Warn("Warning message")
facades.Log().Error("Error message")
```

---

## License

This project is open-sourced software licensed under the [Apache 2.0 license](LICENSE).
