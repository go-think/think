package router

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/go-think/think/context"
)

// Rule Route rule
type Rule struct {
	middlewares    []Middleware
	method         []string
	pattern        string
	handler        interface{}
	parameterNames []string
	parameters     []*parameter
	Compiled       *Compiled
}

type Compiled struct {
	Regex string
}

// Matches Determine if the rule matches given request.
func (r *Rule) Matches(method, path string) bool {
	r.compile()

	if false == matchMethods(method, r.method) {
		return false
	}

	if false == matchPath(path, r.Compiled.Regex) {
		return false
	}

	return true
}

// Bind Bind the router to a given request for execution.
func (r *Rule) Bind(path string) {
	r.compile()

	path = "/" + strings.TrimLeft(path, "/")
	reg := regexp.MustCompile(r.Compiled.Regex)
	matches := reg.FindStringSubmatch(path)[1:]

	parameterNames := r.getParameterNames()

	if len(parameterNames) == 0 {
		return
	}

	parameters := make([]*parameter, 0)

	for k, v := range parameterNames {
		parameters = append(parameters, &parameter{
			name:  v,
			value: matches[k],
		})
	}

	r.parameters = parameters
}

// Middleware Set the middleware attached to the rule.
func (r *Rule) Middleware(middlewares ...Middleware) *Rule {
	for _, m := range middlewares {
		r.middlewares = append(r.middlewares, m)
	}
	return r
}

// GatherRouteMiddleware Get all middleware, including the ones from the controller.
func (r *Rule) GatherRouteMiddleware() []Middleware {
	return r.middlewares
}

// Run Run the route action and return the response.
func (r *Rule) Run(request *context.Request) (result interface{}) {
	if r.handler == nil {
		return
	}

	v := reflect.ValueOf(r.handler)
	switch v.Type().Kind() {
	case reflect.Func:
		in := parseParams(v, request, r.parameters)
		out := v.Call(in)

		if len(out) > 0 {
			result = out[0].Interface()
		}
	default:
		result = r.handler
	}

	return
}

// getParameterNames Get all of the parameter names for the rule.
func (r *Rule) getParameterNames() []string {
	if r.parameterNames != nil {
		return r.parameterNames
	}
	r.parameterNames = r.compileParameterNames()

	return r.parameterNames
}

func (r *Rule) compile() {
	if r.Compiled != nil {
		return
	}

	r.pattern = strings.Replace(r.pattern, "/*", "/.*", -1)

	reg, _ := regexp.Compile(`\{\w+\}`)
	regex := reg.ReplaceAllString(r.pattern, "([^/]+)")

	r.Compiled = &Compiled{
		Regex: "^" + regex + "$",
	}
}

func (r *Rule) compileParameterNames() []string {
	reg := regexp.MustCompile(`\{(.*?)\}`)
	matches := reg.FindAllStringSubmatch(r.pattern, -1)

	var result []string
	for _, v := range matches {
		result = append(result, v[1])
	}

	return result
}

func parseParams(value reflect.Value, request *context.Request, parameters []*parameter) []reflect.Value {
	valueType := value.Type()
	needNum := valueType.NumIn()
	if needNum < 1 {
		return nil
	}

	in := make([]reflect.Value, 0, needNum)
	t := valueType.In(0)
	k := t.Kind()
	ptr := reflect.Ptr == k
	if ptr {
		k = t.Elem().Kind()
	}
	if k == reflect.ValueOf(request).Elem().Kind() {
		var v reflect.Value
		if ptr {
			v = reflect.ValueOf(request)
		} else {
			v = reflect.ValueOf(request).Elem()
		}
		in = append(in, v)
		needNum--
	}

	for _, p := range parameters {
		in = append(in, reflect.ValueOf(p.value))
	}

	return in
}
