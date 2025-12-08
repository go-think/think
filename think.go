package thinkgo

import (
	"fmt"
	"net/http"

	"github.com/go-think/log"
	"github.com/go-think/log/record"

	"time"

	"github.com/go-think/think/config"
	"github.com/go-think/think/helper"
	"github.com/go-think/think/router"
)

type registerRouteFunc func(route *router.Route)

type registerConfigFunc func()

type Think struct {
	App      *Application
	handlers []HandlerFunc
}

// New Create The Application
func New() *Think {
	application := NewApplication()
	application.Logger = log.NewLogger("develop", record.DEBUG)
	t := &Think{
		App: application,
	}
	//t.bootView()
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

// RegisterConfig Register Config
func (th *Think) RegisterHandler(handler HandlerFunc) {
	th.handlers = append(th.handlers, handler)
}

// Run thinkgo application.
// Run() default run on HttpPort
// Run("localhost")
// Run(":9011")
// Run("127.0.0.1:9011")
func (th *Think) Run(params ...string) {
	var err error
	var endRunning = make(chan bool, 1)

	var addrs = helper.ParseAddr(params...)

	// register route handler
	th.RegisterHandler(NewRouteHandler)

	pipeline := NewPipeline()
	for _, h := range th.handlers {
		pipeline.Pipe(h(th.App))
	}

	th.App.Logger.Debug("\r\nLoaded routes:\r\n%s", string(th.App.GetRoute().Dump()))

	go func() {
		th.App.Logger.Debug("ThinkGo server running on http://%s", addrs)

		err = http.ListenAndServe(addrs, pipeline)

		if err != nil {
			fmt.Println(err.Error())
			time.Sleep(100 * time.Microsecond)
			endRunning <- true
		}
	}()

	<-endRunning
}

//func (th *Think) bootView() {
//	v := view.NewView()
//	v.SetPath(config.View.Path)
//	th.App.RegisterView(v)
//}

func (th *Think) bootRoute() {
	r := router.New()
	r.Statics(config.Route.Static)
	th.App.RegisterRoute(r)
}
