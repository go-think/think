package think

import (
	"github.com/go-think/think/context"
	"github.com/go-think/think/middleware"
	"github.com/go-think/think/pipeline"
	"github.com/go-think/think/router"
)

type (
	Req = context.Request
	Res = context.Response

	Route = router.Route

	Handler  = middleware.Handler
	Closure  = middleware.Closure
	Pipeline = pipeline.Pipeline
)

