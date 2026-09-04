package router

import (
	"reflect"
	"strings"

	"github.com/go-think/think/flow"
	"github.com/go-think/think/contract"
)

// RunRoute Return the response for the given rule.
func RunRoute(request *flow.Request, rule *Rule, params ...[]*parameter) interface{} {
	return PrepareResponse(
		request,
		rule,
		runMiddlewares(request, rule, params...),
	)
}

// PrepareResponse Create a response instance from the given value.
func PrepareResponse(request *flow.Request, rule *Rule, result interface{}) interface{} {
	if res, ok := result.(*flow.Response); ok {
		return res
	}
	return flow.NewResponse().SetContent(flow.FormatContent(result))
}

// resolveMiddleware resolves a single middleware interface{} into Middleware(s)
func resolveMiddleware(m interface{}, kernel contract.HttpKernel) []Middleware {
	var resolved []Middleware
	if md, ok := m.(Middleware); ok {
		resolved = append(resolved, md)
	} else if handler, ok := m.(interface{ Process(req *flow.Request, next Closure) interface{} }); ok {
		resolved = append(resolved, handler.Process)
	} else if name, ok := m.(string); ok && kernel != nil {
		var params []string
		if idx := strings.Index(name, ":"); idx != -1 {
			paramsStr := name[idx+1:]
			name = name[:idx]
			params = strings.Split(paramsStr, ",")
		}

		if group := kernel.GetMiddlewareGroup(name); group != nil {
			for _, gm := range group {
				resolved = append(resolved, resolveMiddleware(gm, kernel)...)
			}
		} else if alias := kernel.GetRouteMiddleware(name); alias != nil {
			if len(params) > 0 {
				if factory, ok := alias.(ParameterizedMiddleware); ok {
					resolved = append(resolved, factory(params...))
				} else if factory, ok := alias.(func(...string) Middleware); ok {
					resolved = append(resolved, factory(params...))
				} else {
					val := reflect.ValueOf(alias)
					if val.Kind() == reflect.Func {
						var inArgs []reflect.Value
						if val.Type().IsVariadic() {
							for _, p := range params {
								inArgs = append(inArgs, reflect.ValueOf(p))
							}
						} else {
							numIn := val.Type().NumIn()
							for i := 0; i < numIn && i < len(params); i++ {
								inArgs = append(inArgs, reflect.ValueOf(params[i]))
							}
						}
						out := val.Call(inArgs)
						if len(out) > 0 {
							resolved = append(resolved, resolveMiddleware(out[0].Interface(), kernel)...)
						}
					} else {
						resolved = append(resolved, resolveMiddleware(alias, kernel)...)
					}
				}
			} else {
				resolved = append(resolved, resolveMiddleware(alias, kernel)...)
			}
		}
	} else if m != nil {
		val := reflect.ValueOf(m)
		method := val.MethodByName("Process")
		if method.IsValid() && method.Type().NumIn() == 2 {
			resolved = append(resolved, func(req *flow.Request, next Closure) interface{} {
				nextVal := reflect.ValueOf(next)
				targetType := method.Type().In(1)
				if nextVal.Type().ConvertibleTo(targetType) {
					nextVal = nextVal.Convert(targetType)
				}
				res := method.Call([]reflect.Value{reflect.ValueOf(req), nextVal})
				if len(res) > 0 {
					return res[0].Interface()
				}
				return nil
			})
		}
	}
	return resolved
}

// runMiddlewares Run the given route within Middlewares instance.
func runMiddlewares(request *flow.Request, rule *Rule, params ...[]*parameter) interface{} {
	pipeline := NewPipeline()

	var kernel contract.HttpKernel
	if app := getAppContainer(); app != nil {
		if k := app.Make[contract.HttpKernel](); k != nil {
			kernel = k
		}
	}

	var routeMiddlewares []interface{}
	for _, m := range rule.GatherRouteMiddleware() {
		for _, md := range resolveMiddleware(m, kernel) {
			pipeline.Pipe(md)
			routeMiddlewares = append(routeMiddlewares, md)
		}
	}

	request.SetRouteMiddlewares(routeMiddlewares)

	return pipeline.Passable(request).Run(func(request *flow.Request, next Closure) interface{} {
		return PrepareResponse(
			request,
			rule,
			rule.Run(request, params...),
		)
	})
}
