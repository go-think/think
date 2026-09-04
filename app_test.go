package think

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	contextPkg "github.com/go-think/think/flow"
	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
	"github.com/go-think/think/facades"
	"github.com/go-think/think/middleware"
	"github.com/go-think/think/router"
	"github.com/go-think/think/support/env"
	"github.com/stretchr/testify/assert"
)

func testRequest(t *testing.T, method, reqUrl string, data url.Values, res *contextPkg.Response) {
	tr := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}

	var body io.Reader
	method = strings.ToUpper(method)
	switch method {
	case "GET":
		if data != nil {
			reqUrl = strings.TrimRight(reqUrl, "?") + "?" + data.Encode()
		}
	case "POST", "PUT", "DELETE":
		if data != nil {
			body = strings.NewReader(data.Encode())
		}
	}

	req, err := http.NewRequest(method, reqUrl, body)
	assert.NoError(t, err)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	content, ioerr := io.ReadAll(resp.Body)
	assert.NoError(t, ioerr)

	assert.Equal(t, res.GetCode(), resp.StatusCode)
	assert.Equal(t, res.GetContent(), string(content))
}

func TestRunWithPort(t *testing.T) {
	app := New()

	done := make(chan struct{})
	go func() {
		defer close(done)
		r := app.Make[*router.Route]()
		r.Get("/", func() string {
			return "it worked"
		})
		r.Register()
		app.Run(":9012")
	}()

	time.Sleep(300 * time.Millisecond)

	testRequest(t, "get", "http://localhost:9012/", nil, contextPkg.NewResponse().SetContent("it worked"))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = app.Shutdown(ctx)
	<-done
}

func TestApplication_Run(t *testing.T) {
	app := New()

	done := make(chan struct{})
	go func() {
		defer close(done)
		r := app.Make[*router.Route]()
		r.Get("/", func(req *contextPkg.Request) interface{} {
			return "it worked"
		})
		r.Get("/user/{name}", func(req *contextPkg.Request, name string) interface{} {
			return fmt.Sprintf("Hello %s !", name)
		})
		r.Post("/user", func(req *contextPkg.Request) interface{} {
			name, err := req.Post("name")
			assert.Nil(t, err)
			return name
		})
		r.Delete("/user/{name}", func(name string) interface{} {
			return name
		})
		r.Register()
		app.Run(":9011")
	}()

	time.Sleep(300 * time.Millisecond)

	testRequest(t, "get", "http://localhost:9011/", nil, contextPkg.NewResponse().SetContent("it worked"))
	testRequest(t, "get", "http://localhost:9011/user/think", nil, contextPkg.NewResponse().SetContent("Hello think !"))
	testRequest(t, "post", "http://localhost:9011/user", url.Values{"name": {"think"}}, contextPkg.NewResponse().SetContent("think"))
	testRequest(t, "delete", "http://localhost:9011/user/think", nil, contextPkg.NewResponse().SetContent("think"))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = app.Shutdown(ctx)
	<-done
}

// TestApplication_DeferredMiddleware verifies both Response standardization in the pipeline.Pipeline and Deferred middleware resolution.
func TestApplication_DeferredMiddleware(t *testing.T) {
	app := New()

	done := make(chan struct{})
	go func() {
		defer close(done)
		r := app.Make[*router.Route]()
		
		// 1. Register a route with a middleware alias that hasn't been defined in the kernel yet!
		// In the old design, this would crash or fail.
		r.Get("/deferred", func(req *contextPkg.Request) interface{} {
			return "controller_result"
		}).Middleware("mock_alias")
		
		r.Register()
		
		// 2. Define the mock middleware now (after route registration)
		mockMiddleware := func(req *contextPkg.Request, next router.Closure) interface{} {
			// Call next. Because of our pipeline.Pipeline rewrite, result is guaranteed to be a *flow.Response
			result := next(req)
			if res, ok := result.(*contextPkg.Response); ok {
				res.Header.Set("X-Mock-Framework", "go-think") // modify the header on the way back out
				return res
			}
			return result
		}
		
		// 3. Bind the alias to the Kernel now
		app.HttpKernel().AddRouteMiddleware("mock_alias", router.Middleware(mockMiddleware))

		app.Run(":9013")
	}()

	time.Sleep(300 * time.Millisecond)

	// Make request
	req, _ := http.NewRequest("GET", "http://localhost:9013/deferred", nil)
	req.Close = true
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	content, ioerr := io.ReadAll(resp.Body)
	assert.NoError(t, ioerr)
	
	// Verify response body
	assert.Equal(t, "controller_result", string(content))
	
	// Verify that the middleware was able to set the header on the standardized Response object!
	assert.Equal(t, "go-think", resp.Header.Get("X-Mock-Framework"))
	
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = app.Shutdown(ctx)
	<-done
	time.Sleep(50 * time.Millisecond)
}

func TestApplication_Environment(t *testing.T) {
	app := New()
	assert.NotNil(t, app)

	// Default or env loaded
	app.SetEnvironment("production")
	assert.True(t, app.IsProduction())
	assert.False(t, app.IsLocal())
	assert.True(t, app.Environment("production"))
	assert.True(t, app.Environment("local", "production"))
	assert.False(t, app.Environment("staging", "testing"))
	assert.Equal(t, "production", app.GetEnvironment())

	// Test Facade access
	assert.True(t, facades.App.IsProduction())
	assert.False(t, facades.App.IsLocal())

	// Change to local
	app.SetEnvironment("local")
	assert.True(t, app.IsLocal())
	assert.False(t, app.IsProduction())
	assert.True(t, facades.App.IsLocal())
	assert.True(t, facades.App.Environment("local", "dev"))

	// Change to testing
	app.SetEnvironment("testing")
	assert.True(t, app.IsTesting())
	assert.True(t, facades.App.IsTesting())
}

func TestApplication_WithConfig_SetStruct(t *testing.T) {
	type CustomAppConfig struct {
		Name     string `config:"name"`
		Env      string `config:"env"`
		Debug    bool   `config:"debug"`
		Port     int    `config:"port"`
		Timezone string `config:"timezone"`
	}

	type CustomDBConfig struct {
		Host string `config:"host"`
		Port int    `config:"port"`
	}

	type CustomDatabaseConfig struct {
		Default string         `config:"default"`
		MySQL   CustomDBConfig `config:"mysql"`
	}

	builder := Configure()
	if cfg := builder.app.Make[contract.Config](); cfg != nil {
		cfg.Load(map[string]interface{}{
			"app": CustomAppConfig{
				Name:     "TestThinkApp",
				Env:      "testing",
				Debug:    true,
				Port:     9999,
				Timezone: "Asia/Shanghai",
			},
			"database": &CustomDatabaseConfig{
				Default: "mysql",
				MySQL: CustomDBConfig{
					Host: "192.168.1.100",
					Port: 3307,
				},
			},
		})
	}
	app := builder.Create()
	app.Boot()

	// 1. Verify environment, debug and timezone were automatically updated via Boot
	assert.True(t, app.IsTesting())
	assert.False(t, app.IsProduction())
	assert.True(t, facades.App.IsTesting())
	assert.True(t, app.IsDebug())
	assert.Equal(t, "Asia/Shanghai", app.GetTimezone())

	// 2. Verify dot-notation access on nested struct fields
	cfg := facades.Config()
	assert.Equal(t, "TestThinkApp", cfg.GetString("app.name"))
	assert.Equal(t, "testing", cfg.GetString("app.env"))
	assert.Equal(t, 9999, cfg.GetInt("app.port"))
	assert.Equal(t, true, cfg.GetBool("app.debug"))
	assert.Equal(t, "Asia/Shanghai", cfg.GetString("app.timezone"))
	assert.Equal(t, "mysql", cfg.GetString("database.default"))
	assert.Equal(t, "192.168.1.100", cfg.GetString("database.mysql.host"))
	assert.Equal(t, 3307, cfg.GetInt("database.mysql.port"))
}

func TestApplication_EnvironmentAutoDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .env and .env.testing
	_ = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("APP_NAME=DefaultApp\nAPP_ENV=local\n"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, ".env.testing"), []byte("APP_NAME=TestingApp\nAPP_ENV=testing\n"), 0644)

	// Set APP_ENV=testing to test automatic inference
	t.Setenv("APP_ENV", "testing")

	app := New(tmpDir)
	app.Bootstrap()

	assert.Equal(t, ".env.testing", app.EnvironmentFile())
	assert.Equal(t, filepath.Join(tmpDir, ".env.testing"), app.EnvironmentFilePath())
	assert.Equal(t, "TestingApp", env.Get("APP_NAME"))
	assert.True(t, app.IsTesting())
}

type testConfigProvider struct{}

func (p *testConfigProvider) Register(app *container.Container) {
	if cfg := app.Make[contract.Config](); cfg != nil {
		cfg.Load(map[string]interface{}{
			"app": map[string]interface{}{
				"name": "BuilderApp",
				"env":  "testing",
			},
		})
	}
}

func (p *testConfigProvider) Boot(app *container.Container) {}

func TestApplicationBuilder_Configure_Create(t *testing.T) {
	app := Configure().
		WithProviders(&testConfigProvider{}).
		Create()
	app.Boot()

	assert.NotNil(t, app)
	assert.True(t, app.IsTesting())
	assert.Equal(t, "BuilderApp", facades.Config().GetString("app.name"))
	assert.Equal(t, "testing", facades.Config().GetString("app.env"))
}

func TestApplication_EnvLifecycle_Priority(t *testing.T) {
	// Create temporary environment files in a fake project base directory
	tmpDir, err := os.MkdirTemp("", "think_base_test_*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	envLocalPath := filepath.Join(tmpDir, ".env")
	err = os.WriteFile(envLocalPath, []byte("APP_NAME=ThinkLocal\nAPP_PORT=8888\n"), 0644)
	assert.NoError(t, err)

	// Test New with basePath
	app := New(tmpDir)
	assert.Equal(t, tmpDir, app.BasePath())
	assert.Equal(t, filepath.Join(tmpDir, "config"), app.ConfigPath())
	assert.Equal(t, filepath.Join(tmpDir, "storage"), app.StoragePath())
	assert.Equal(t, filepath.Join(tmpDir, "public"), app.PublicPath())
	assert.Equal(t, filepath.Join(tmpDir, "config", "app.go"), app.ConfigPath("app.go"))

	// Explicitly bootstrap environment
	app.Bootstrap()
	assert.Equal(t, "ThinkLocal", env.Get("APP_NAME"))
	assert.Equal(t, "8888", env.Get("APP_PORT"))

	// Test Configure with custom env file chaining
	envProdPath := filepath.Join(tmpDir, ".env.prod")
	err = os.WriteFile(envProdPath, []byte("APP_NAME=ThinkProduction\nAPP_PORT=80\n"), 0644)
	assert.NoError(t, err)

	Configure().WithEnvFile(envProdPath).Create()
	assert.Equal(t, "ThinkProduction", env.Get("APP_NAME"))
	assert.Equal(t, "80", env.Get("APP_PORT"))
}


func TestApplicationBuilder_FullStyle(t *testing.T) {
	var reportedError interface{}
	var middlewareCalled bool

	app := Configure().
		WithRouting(Routing{
			Web: func(r *router.Route) {
				r.Get("/hello", func() string {
					return "world"
				})
			},
			Api: func(r *router.Route) {
				r.Get("/api/users", func() string {
					return "users_list"
				})
			},
			Health: "/up",
		}).
		WithMiddleware(func(m *MiddlewareConfig) {
			m.Use(middleware.HandlerFunc(func(req *contextPkg.Request, next middleware.Closure) interface{} {
				middlewareCalled = true
				return next(req)
			}))
			m.Alias("mock_auth", middleware.HandlerFunc(func(req *contextPkg.Request, next middleware.Closure) interface{} {
				return next(req)
			}))
		}).
		WithExceptions(func(e *ExceptionsConfig) {
			e.Report(func(err interface{}) bool {
				reportedError = err
				return true
			})
			e.Render(func(err interface{}) interface{} {
				return contextPkg.NewResponse().SetCode(503).SetContent("Custom Service Unavailable")
			})
		}).
		Create()

	assert.NotNil(t, app)

	// 1. Verify /up health check route
	reqUp := contextPkg.NewRequest(&http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/up"},
	})
	resUp := app.HttpKernel().Handle(reqUp)
	if resp, ok := resUp.(*contextPkg.Response); ok {
		assert.Equal(t, 200, resp.GetCode())
		assert.Equal(t, "UP", resp.GetContent())
	} else {
		t.Fatalf("expected *flow.Response from /up, got: %T", resUp)
	}

	// 2. Verify global middleware execution
	assert.True(t, middlewareCalled)

	// 3. Verify custom exception rendering
	handler := app.Make[contract.ExceptionHandler]()
	handler.Report("test_error")
	assert.Equal(t, "test_error", reportedError)

	renderResult := handler.Render("test_error")
	if resp, ok := renderResult.(*contextPkg.Response); ok {
		assert.Equal(t, 503, resp.GetCode())
		assert.Equal(t, "Custom Service Unavailable", resp.GetContent())
	} else {
		t.Fatalf("expected *flow.Response from handler.Render, got: %T", renderResult)
	}
}

func TestApplicationBuilder_WithEvents(t *testing.T) {
	var userRegisteredReceived string
	var orderPaidReceived string

	app := Configure().
		WithEvents(func(events contract.EventDispatcher) {
			events.Listen("user.registered", func(payload string) {
				userRegisteredReceived = payload
			})
		}, map[string]interface{}{
			"order.paid": func(payload string) {
				orderPaidReceived = payload
			},
		}).
		Create()

	assert.NotNil(t, app)

	// Dispatch events via facades.Event()
	facades.Event().Dispatch("user.registered", "alice")
	facades.Event().Dispatch("order.paid", "order_123")

	assert.Equal(t, "alice", userRegisteredReceived)
	assert.Equal(t, "order_123", orderPaidReceived)
}
