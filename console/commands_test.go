package console

import (
	"os"
	"strings"
	"testing"

	"github.com/go-think/flow"
	"github.com/go-think/think/container"
)

func TestConsoleCommands_RouteList(t *testing.T) {
	app := container.New()
	r := flow.New(nil, nil)
	r.Get("/users", func(req *flow.Request) *flow.Response {
		return flow.Text("users")
	}).Name("users.index")
	r.Post("/users", func(req *flow.Request) *flow.Response {
		return flow.Text("create user")
	}).Name("users.store")
	r.Get("/api/posts", func(req *flow.Request) *flow.Response {
		return flow.Text("posts")
	}).Name("posts.index")

	app.Instance[flow.Router](r)
	kernel := NewKernel(app, nil)

	// Test default route:list
	status := kernel.Handle("route:list")
	if status != 0 {
		t.Fatalf("expected status 0, got %d", status)
	}

	// Test route:list --json
	status = kernel.Handle("route:list", "--json")
	if status != 0 {
		t.Fatalf("expected status 0 for --json, got %d", status)
	}

	// Test route:list with filters
	status = kernel.Handle("route:list", "--method=POST", "--path=/users")
	if status != 0 {
		t.Fatalf("expected status 0 for filtered list, got %d", status)
	}
}

func TestRenderRouteList_Filtering(t *testing.T) {
	summaries := []flow.RouteSummary{
		{
			Methods: []string{"GET", "HEAD"},
			URI:     "/users",
			Name:    "users.index",
			Action:  "UserController@Index",
		},
		{
			Methods: []string{"POST"},
			URI:     "/users",
			Name:    "users.store",
			Action:  "UserController@Store",
		},
		{
			Methods: []string{"GET"},
			URI:     "/api/posts",
			Name:    "posts.index",
			Action:  "PostController@Index",
		},
	}

	// Filter by method
	status := renderRouteList(summaries, []string{"--method=POST"})
	if status != 0 {
		t.Errorf("expected status 0, got %d", status)
	}

	// Filter by path and json
	status = renderRouteList(summaries, []string{"--path=/api", "--json"})
	if status != 0 {
		t.Errorf("expected status 0, got %d", status)
	}
}

func TestRunMakeControllerAndMiddleware(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "think_console_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	// Test make:controller
	status := runMakeController([]string{"OrderController", "--api"})
	if status != 0 {
		t.Fatalf("expected make:controller status 0, got %d", status)
	}

	controllerContent, err := os.ReadFile("app/http/controllers/OrderController.go")
	if err != nil {
		t.Fatal(err)
	}
	controllerStr := string(controllerContent)
	if !strings.Contains(controllerStr, `import "github.com/go-think/flow"`) {
		t.Errorf("expected controller to import flow, got:\n%s", controllerStr)
	}
	if !strings.Contains(controllerStr, "type OrderController struct{}") {
		t.Errorf("expected OrderController struct definition")
	}

	// Test make:middleware
	status = runMakeMiddleware([]string{"VerifyToken"})
	if status != 0 {
		t.Fatalf("expected make:middleware status 0, got %d", status)
	}

	middlewareContent, err := os.ReadFile("app/http/middleware/VerifyToken.go")
	if err != nil {
		t.Fatal(err)
	}
	middlewareStr := string(middlewareContent)
	if !strings.Contains(middlewareStr, `import "github.com/go-think/flow"`) {
		t.Errorf("expected middleware to import flow, got:\n%s", middlewareStr)
	}
	if !strings.Contains(middlewareStr, "return &VerifyToken{}") {
		t.Errorf("expected middleware to return &VerifyToken{}, got:\n%s", middlewareStr)
	}
}
