package provider

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/go-think/flow"
	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
	"github.com/go-think/think/exception"
	"github.com/go-think/think/validator"
)

// RoutingServiceProvider registers the core router services into the container.
type RoutingServiceProvider struct {
}

// RouteServiceProvider is an alias to RoutingServiceProvider for backward compatibility.
type RouteServiceProvider = RoutingServiceProvider

// flowContainerAdapter adapts think's container.Container to flow.Container.
type flowContainerAdapter struct {
	c *container.Container
}

type flowExceptionHandlerAdapter struct {
	handler contract.ExceptionHandler
}

func (a *flowExceptionHandlerAdapter) Report(err any) {
	a.handler.Report(err)
}

func (a *flowExceptionHandlerAdapter) Render(req *flow.Request, err any) any {
	if ve, ok := err.(*contract.ValidationException); ok {
		if req != nil && !req.ExpectsJson() && req.Session() != nil {
			targetUrl := req.Header("Referer")
			if targetUrl == "" {
				targetUrl = req.Path()
			}
			return flow.Redirect(targetUrl).
				WithInput().
				WithErrors(ve.Errors)
		}
	}
	return a.handler.Render(err)
}

func (a *flowContainerAdapter) Make(key string) any {
	if key == "ExceptionHandler" {
		if h := a.c.MakeByName("ExceptionHandler"); h != nil {
			if eh, ok := h.(flow.ExceptionHandler); ok {
				return eh
			}
			if eh, ok := h.(contract.ExceptionHandler); ok {
				return &flowExceptionHandlerAdapter{handler: eh}
			}
		}
	}
	return a.c.MakeByName(key)
}

func (a *flowContainerAdapter) Bound(key string) bool {
	if key == "ExceptionHandler" {
		return a.c.Bound("ExceptionHandler")
	}
	return a.c.Bound(key)
}

func (a *flowContainerAdapter) Instance(key string, instance any) {
	a.c.InstanceNamed(key, instance)
}

// Register registers the router into the container.
func (p *RoutingServiceProvider) Register(app *container.Container) {
	r := flow.New(nil, &flowContainerAdapter{c: app})
	if setter, ok := r.(interface{ SetParameterResolver(flow.ParameterResolver) }); ok {
		setter.SetParameterResolver(p.parameterResolver(app))
	}

	generator := flow.NewUrlGenerator(r, "").SetKeyResolver(func() []string {
		cfg := app.Make[contract.Config]()
		if cfg == nil {
			return nil
		}
		return append(
			[]string{cfg.GetString("app.key")},
			cfg.GetStringSlice("app.previous_keys")...,
		)
	})

	generator.SetRequestProvider(func() *flow.Request { return r.CurrentRequest() })

	// Route lifecycle events flow through the application event dispatcher.
	r.OnRouting(func(req *flow.Request) {
		if dispatcher := app.Make[contract.EventDispatcher](); dispatcher != nil {
			dispatcher.Dispatch("flow.routing", req)
		}
	})
	r.OnRouteMatched(func(route *flow.Route, req *flow.Request) {
		if dispatcher := app.Make[contract.EventDispatcher](); dispatcher != nil {
			dispatcher.Dispatch("flow.route.matched", route)
		}
	})

	app.Instance[flow.Router](r)
	app.Instance[*flow.UrlGenerator](generator)
	app.Instance[*flow.Redirector](flow.NewRedirector(generator).SetRequestProvider(func() *flow.Request { return r.CurrentRequest() }))
	app.Alias[flow.Router]("router")
}

// parameterResolver creates a parameter resolver backed by the application container.
func (p *RoutingServiceProvider) parameterResolver(app *container.Container) flow.ParameterResolver {
	return flow.ParameterResolverFunc(func(t reflect.Type, req *flow.Request) (reflect.Value, bool) {
		// 1. If bound in container, resolve it directly
		if dep := app.Get(t); dep != nil {
			return reflect.ValueOf(dep), true
		}

		// 2. Struct pointer resolution (FormRequest / DTO auto-binding and validation)
		if t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct {
			val := reflect.New(t.Elem())
			instance := val.Interface()

			if req != nil {
				// If JSON request, bind JSON payload
				if req.IsJson() {
					_ = req.BindJson(instance)
				}
				// Also bind form/query parameters into struct fields
				bindRequestDataToStruct(req, val.Elem())

				// Check AuthorizesRequests
				if authReq, ok := instance.(contract.AuthorizesRequests); ok {
					if !authReq.Authorize() {
						panic(&exception.HttpException{
							Code:    403,
							Message: "This action is unauthorized.",
						})
					}
				}

				// Check ValidatesWhenResolved
				if vwr, ok := instance.(contract.ValidatesWhenResolved); ok {
					if err := vwr.ValidateResolved(); err != nil {
						panic(err)
					}
					return val, true
				}

				// Check FormRequest
				if formReq, ok := instance.(contract.FormRequest); ok {
					rules := formReq.Rules()
					var customMsgs map[string]string
					if cm, ok := instance.(contract.CustomMessages); ok {
						customMsgs = cm.Messages()
					}
					if _, err := validator.Validate(instance, rules, customMsgs); err != nil {
						panic(err)
					}
					return val, true
				}
			}

			return val, true
		}

		return reflect.Value{}, false
	})
}

func bindRequestDataToStruct(req *flow.Request, structVal reflect.Value) {
	allData := make(map[string]string)
	for k, v := range req.All() {
		allData[k] = v
	}
	for k, v := range req.RouteParams() {
		allData[k] = v
	}
	if len(allData) == 0 {
		return
	}
	bindDataMapToStruct(allData, structVal)
}

func bindDataMapToStruct(allData map[string]string, structVal reflect.Value) {
	t := structVal.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := structVal.Field(i)

		// Support embedded anonymous structs
		if field.Anonymous && fv.Kind() == reflect.Struct {
			bindDataMapToStruct(allData, fv)
			continue
		}

		if !field.IsExported() {
			continue
		}

		key := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			key = strings.Split(jsonTag, ",")[0]
		} else if formTag := field.Tag.Get("form"); formTag != "" && formTag != "-" {
			key = strings.Split(formTag, ",")[0]
		} else if routeTag := field.Tag.Get("route"); routeTag != "" && routeTag != "-" {
			key = strings.Split(routeTag, ",")[0]
		}

		rawVal, exists := allData[key]
		if !exists {
			rawVal, exists = allData[strings.ToLower(key)]
		}
		if !exists {
			rawVal, exists = allData[toSnake(key)]
		}
		if !exists {
			continue
		}

		// Only set if field is zero-value (so JSON binding takes precedence if already set)
		if !fv.IsZero() {
			continue
		}

		switch fv.Kind() {
		case reflect.String:
			fv.SetString(rawVal)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if iv, err := strconv.ParseInt(rawVal, 10, 64); err == nil {
				fv.SetInt(iv)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if uv, err := strconv.ParseUint(rawVal, 10, 64); err == nil {
				fv.SetUint(uv)
			}
		case reflect.Float32, reflect.Float64:
			if flv, err := strconv.ParseFloat(rawVal, 64); err == nil {
				fv.SetFloat(flv)
			}
		case reflect.Bool:
			lower := strings.ToLower(rawVal)
			if lower == "true" || lower == "1" || lower == "on" {
				fv.SetBool(true)
			} else if lower == "false" || lower == "0" || lower == "off" {
				fv.SetBool(false)
			}
		}
	}
}

func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Boot compiles route rules and validates signing configuration.
// The key itself is resolved lazily by the UrlGenerator's key resolver
// (registered in Register); an unconfigured key leaves signing disabled —
// the generator fails closed.
func (p *RoutingServiceProvider) Boot(app *container.Container) {
	r := app.Make[flow.Router]()
	if r == nil {
		return
	}

	if cfg := app.Make[contract.Config](); cfg != nil && cfg.GetString("app.key") == "" {
		if logger := app.Make[contract.Logger](); logger != nil {
			logger.Error("app.key is not configured: signed URLs are disabled and signature validation always fails. Set APP_KEY / app.key before using signed URLs.")
		}
	}

	if eh := app.Make[contract.ExceptionHandler](); eh != nil {
		r.SetExceptionHandler(&flowExceptionHandlerAdapter{handler: eh})
	}

	r.Register()
}
