<h1 align="center">
  Think
</h1>

<p align="center">
	<strong>Think is a lightweight MVC framework written in Go (Golang).</strong>
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


## Installation

The only requirement is the [Go Programming Language](https://golang.org/dl/)

```
go get -u github.com/go-think/think
```

## Quick start

```go
package main

import (
    "fmt"

	"github.com/go-think/think"	
	"github.com/go-think/think/think"
)

func main() {
	th := thinkgo.New()
	th.RegisterRoute(func(route *think.Route) {

		route.Get("/", func(req *think.Req) *think.Res {
			return think.Text("Hello ThinkGo !")
		})

		route.Get("/ping", func(req *think.Req) *think.Res {
			return think.Json(map[string]string{
				"message": "pong",
		    })
		})

		// Dependency injection
		route.Get("/user/{name}", func(req *think.Req, name string) *think.Res {
			return think.Text(fmt.Sprintf("Hello %s !", name))
		})
	})
	// listen and serve on 0.0.0.0:9011
	th.Run()
}
```

## Features

- [Routing](#routing)
- [Middleware](#middleware)
- [Controller](#controller)
- [Request](#http-request)
- [Response](#http-response)
- [View](#view)
- [HTTP Session](#http-session)
- [Logging](#logging)
- [Cache](#cache)
- [ORM](#orm)

## Routing

#### Basic Routing

The most basic routes accept a URI and a Closure, providing a very simple and expressive method of defining routes:

```go
think.RegisterRoute(func(route *router.Route) {
	route.Get("/foo", func(req *context.Request) *context.Response {
		return thinkgo.Text("Hello ThinkGo !")
	})
})
```

#### Available Router Methods

The router allows you to register routes that respond to any HTTP verb:

```go
route.Get("/someGet", getting)
route.Post("/somePost", posting)
route.Put("/somePut", putting)
route.Delete("/someDelete", deleting)
route.Patch("/somePatch", patching)
route.Options("/someOptions", options)
```

Sometimes you may need to register a route that responds to multiple HTTP verbs. You may even register a route that responds to all HTTP verbs using the `Any` method:

```go
route.Any("/someAny", any)
```

#### Parameters in path

Of course, sometimes you will need to capture segments of the URI within your route. For example, you may need to capture a user's ID from the URL. You may do so by defining route parameters:

```go
route.Get("/user/{id}", func(req *context.Request, id string) *context.Response {
	return thinkgo.Text(fmt.Sprintf("User %s", id))
})
```

You may define as many route parameters as required by your route:

```go
route.Get("/posts/{post}/comments/{comment}", func(req *context.Request, postId, commentId string) *context.Response {
	//
})
```

#### Route Prefixes

The prefix method may be used to prefix each route in the group with a given URI. For example, you may want to prefix all route URIs within the group with `admin`:

```go
route.Prefix("/admin").Group(func(group *router.Route) {
	group.Prefix("user").Group(func(group *router.Route) {
	    // ... 	
	})
	group.Prefix("posts").Group(func(group *router.Route) {
		// ... 	
    })
})
```

#### Route Groups

Route groups allow you to share route attributes, such as middleware or prefix, across a large number of routes without needing to define those attributes on each individual route.

```go
route.Prefix("/admin").Group(func(group *router.Route) {
	group.Prefix("user").Group(func(group *router.Route) {
		group.Get("", func(request *context.Request) *context.Response {
			return thinkgo.Text("admin user !")
		}).Middleware(func(request *context.Request, next router.Closure) interface{} {
			if _, err := request.Input("id"); err != nil {
				return thinkgo.Text("Invalid parameters")
			}
			return next(request)
		})
		group.Get("edit", func(request *context.Request) *context.Response {
			return thinkgo.Text("admin user edit !")
		})
	}).Middleware(func(request *context.Request, next router.Closure) interface{} {
		if _, err := request.Input("user"); err != nil {
			return thinkgo.Text("Invalid parameters")
		}
		return next(request)
	})
}).Middleware(func(request *context.Request, next router.Closure) interface{} {
	if _, err := request.Input("token"); err != nil {
		return thinkgo.Text("Invalid parameters")
	}
	return next(request)
})
```

## Middleware

Middleware provide a convenient mechanism for filtering HTTP requests entering your application. You only need to implement the `Middleware` interface.

```go
route.Get("/foo", func(request *context.Request) *context.Response {
	return thinkgo.Text("Hello ThinkGo !")
}).Middleware(func(request *context.Request, next router.Closure) interface{} {
	if _, err := request.Input("name"); err != nil {
		return thinkgo.Text("Invalid parameters")
	}
	return next(request)
})
```

#### Before Middleware

Whether a middleware runs before or after a request depends on the middleware itself. For example, the following middleware would perform some task `before` the request is handled by the application:

```go
func(request *context.Request, next router.Closure) interface{} {
	
	// Perform action	
	// ...
	
	return next(request)
}
```

#### After  Middleware

However, this middleware would perform its task `after` the request is handled by the application:

```go
func(request *context.Request, next router.Closure) interface{} {
	
	response := next(request)
	
	// Perform action	
	// ...
	
	return response
}
```

## Controller

#### Basic Controller

Below is an example of a basic controller class.

```go
package controller

import (
	"github.com/go-think/think"
	"github.com/go-think/think/context"
)

func Index(req *context.Request) *context.Response {
	return thinkgo.Text("Hello ThinkGo !")
}

```

You can define a route to this controller like so:

```go
route.Get("/", controller.Index)
```

#### Resource Controller

This feature will be supported in a future release.

## HTTP Request

#### Accessing The Request

To obtain an instance of the current HTTP request via dependency injection

```go
func Handler(req *context.Request) *context.Response {
	name := req.Input("name")
}
```

#### Dependency Injection & Route Parameters

If your controller method is also expecting input from a route parameter you should list your route parameters after the request dependencies. For example, you can access your route parameter `name` like so:

```go
route.Put("/user/{name}", func(req *context.Request, name string) *context.Response {
	//
})
```

#### Request Path & Method

The `path` method returns the request's path information. So, if the incoming request is targeted at `http://domain.com/foo/bar`, the `path` method will return `foo/bar`:

```go
uri := req.GetPath()
```

The `method` method will return the HTTP verb for the request. 

```go
method := req.GetMethod();
```

#### Retrieving Cookies From Requests

```go
name, _ := request.Cookie("name")
```

## HTTP Response

an HTTP Response Must implement the `*context.Response` interface

#### Creating Responses

a simple strings or json Response:

```go
thinkgo.Text("Hello ThinkGo !")

thinkgo.Json(map[string]string{
				"message": "pong",
			})
```

#### Attaching Cookies To Responses

```go
response.Cookie("name", "alice")
```

#### Redirects

```go
route.Get("/redirect", func(request *context.Request) *context.Response {
	return context.Redirect("https://www.google.com")
})
```

## View

Specify the `views` directory before running the app:

```go
view.ParseGlob("/path/to/views/*")
```

views are stored in the `views` directory, A simple view might look something like this:

`views/layout.html` like this:

```html
{{ define "layout" }}
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>{{ .Title }}</title>
</head>
<body>
	{{ template "content" .}}
</body>
</html>
{{ end }}
```

`views/tpl.html` like this:

```html
{{ define "content" }}
<h2>{{ .Message }}</h2>
{{ end }}
{{ template "layout" . }}
```

we may return it using the `Render` function like so:

```go
route.Get("/tpl", func(request *context.Request) *context.Response {
	data := map[string]interface{}{"Title": "ThinkGo", "Message": "Hello ThinkGo !"}
	return view.Render("tpl.html", data)
})
```

## HTTP Session

When the app starts, you need to register the session handler.

```go
think.RegisterHandler(app.NewSessionHandler)
```

`ThinkGo` ships with several great drivers out of the box:

- cookie - sessions are stored in cookies
- file - sessions are stored in files.

#### Using The Session

retrieving Data like this:

```go
request.Session().Get("user")
```

storing Data like this:

```go
request.Session().Set("user", "alice")
```

#### Adding Custom Session Drivers

Your custom session driver should implement the `Handler`. 

```go
type Handler interface {
	Read(id string) string
	Write(id string, data string)
}
```

Once your driver has been implemented, you are ready to register it:

```go
import "github.com/go-think/think/session"

session.Extend("my_session", MySessionHandler)
```

## Logging

The logger provides the eight logging levels defined in [RFC 5424]( https://tools.ietf.org/html/rfc5424 ): **emergency**, **alert**, **critical**, **error**, **warning**, **notice**, **info** and **debug**.

#### Basic Usage

```go
import "github.com/go-think/log"

log.Debug("log with Debug")
log.Info("log with Info")
log.Notice("log with Notice")
log.Warn("log with Warn")
log.Error("log with Error")
log.Crit("log with Crit")
log.Alert("log with Alert")
log.Emerg("log with Emerg")
```

#### Log Storage

Out of the box, ThinkGo supports writing log information to `daily` files, the `console`.

For example, if you wish to use `daily` log files, you can do this: 

```go
import (
	"github.com/go-think/log"
	"github.com/go-think/log/handler"
	"github.com/go-think/log/record"
)

fh := handler.NewFileHandler("path/to/think.log", record.INFO)

log.GetLogger().PushHandler(fh)
```

## Cache

ThinkGo Cache Currently supports redis, memory, and can customize the store adapter.

#### Basic Usage

```go
import (
	"github.com/go-think/cache"
	"time"
)


var foo string 

// Create a cache with memory store
c, _ := cache.Cache(cache.NewMemoryStore("think go"))

// Set the value
c.Put("foo", "think go", 10 * time.Minute)

// Get the string associated with the key "foo" from the cache
c.Get("foo", &foo)

```

#### Retrieve & Store

Sometimes you may wish to retrieve an item from the cache, but also store a default value if the requested item doesn't exist. For example, you may wish to retrieve all users from the cache or, if they don't exist, retrieve them from the callback and add them to the cache. You may do this using the `Remember` method:

```go
var foo int

cache.Remember("foo", &a, 1*time.Minute, func() interface{} {
	return "thinkgo"
})
```

refer to [ThinkGo Cache]( https://github.com/go-think/think/tree/master/cache )

## ORM

refer to [ThinkORM]( https://github.com/go-think/think )

## License

This project is licensed under the [Apache 2.0 license](LICENSE).

## Contact

If you have any issues or feature requests, please contact us. PR is welcomed.
- https://github.com/go-think/think/issues
- leeqvip@gmail.com
