package think

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-think/log"
	"github.com/go-think/log/record"
	"github.com/go-think/think/config"
	"github.com/go-think/think/helper"
	"github.com/go-think/think/middleware"
	"github.com/go-think/think/pipeline"
	"github.com/go-think/think/router"
	"github.com/go-think/think/view"
)

type registerRouteFunc func(route *router.Route)
type registerConfigFunc func()
type HandlerFunc func(app *Application) middleware.Handler

type Think struct {
	App      *Application
	handlers []middleware.Handler
	server   *http.Server
}

// New Create The Application
func New() *Think {
	application := NewApplication()
	application.Logger = log.NewLogger("develop", record.DEBUG)
	t := &Think{
		App: application,
	}
	t.bootView()
	t.bootRoute()
	return t
}

// RegisterRoute Register Route
func (th *Think) RegisterRoute(register registerRouteFunc) {
	route := th.App.GetRoute()
	defer route.Register()
	register(route)
}

// RegisterConfig Register Config
func (th *Think) RegisterConfig(register registerConfigFunc) {
	register()
}

// RegisterHandler Register Middleware Handler
func (th *Think) RegisterHandler(handler middleware.Handler) {
	th.handlers = append(th.handlers, handler)
}

// BuildServer 构建 Pipeline 与 HTTP Server 实例
func (th *Think) BuildServer(params ...string) *http.Server {
	addrs := helper.ParseAddr(params...)

	pipe := pipeline.NewPipeline()
	// 挂载已注册的中间件
	pipe.Through(th.handlers)
	// 挂载核心路由分发 Handler
	pipe.Pipe(middleware.NewRouteHandler(th.App.GetRoute()))

	th.server = &http.Server{
		Addr:    addrs,
		Handler: pipe,
	}
	return th.server
}

// Shutdown 优雅关闭 Server
func (th *Think) Shutdown(ctx context.Context) error {
	if th.server != nil {
		return th.server.Shutdown(ctx)
	}
	return nil
}

// Run think application with graceful shutdown support
func (th *Think) Run(params ...string) {
	srv := th.BuildServer(params...)

	th.App.Logger.Debug("\r\nLoaded routes:\r\n%s", string(th.App.GetRoute().Dump()))

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		th.App.Logger.Debug("Shutting down Think server gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := th.Shutdown(ctx); err != nil {
			th.App.Logger.Debug("Think Server Shutdown Error: %v", err)
		}
		close(idleConnsClosed)
	}()

	th.App.Logger.Debug("Think server running on http://%s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("HTTP server ListenAndServe error: %v\n", err)
	}

	// 等待 Graceful Shutdown 完整执行
	<-idleConnsClosed
	th.App.Logger.Debug("Think server stopped gracefully.")
}

func (th *Think) bootView() {
	v := view.New()
	v.ParseGlob(config.View.Path)
	th.App.RegisterView(v)
}

func (th *Think) bootRoute() {
	r := router.New()
	r.Statics(config.Route.Static)
	th.App.RegisterRoute(r)
}

