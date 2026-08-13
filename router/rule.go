package router

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/go-think/think/context"
)

// Rule Route rule
type Rule struct {
	middlewares    []Middleware
	method         []string
	pattern        string
	handler        interface{}
	parameterNames []string
	Compiled       *Compiled
	compileOnce    sync.Once
}

type Compiled struct {
	Regex  string
	Regexp *regexp.Regexp
}

// Matches Determine if the rule matches given request.
func (r *Rule) Matches(method, path string) bool {
	r.compile()

	if false == matchMethods(method, r.method) {
		return false
	}

	if r.Compiled != nil && r.Compiled.Regexp != nil {
		return r.Compiled.Regexp.MatchString(path)
	}

	return matchPath(path, r.Compiled.Regex)
}

// Bind Bind the router parameters to a given request and return parsed parameters.
func (r *Rule) Bind(req Request, path string, treeParams ...[]*parameter) []*parameter {
	r.compile()

	path = "/" + strings.TrimLeft(path, "/")
	parameters := make([]*parameter, 0)

	// 如果来自 Radix Tree 已经提前提取好的参数
	if len(treeParams) > 0 && len(treeParams[0]) > 0 {
		for _, p := range treeParams[0] {
			parameters = append(parameters, p)
			if reqSetter, ok := req.(interface{ SetRouteParam(k, v string) }); ok {
				reqSetter.SetRouteParam(p.name, p.value)
			}
		}
		return parameters
	}

	// 使用预编译正则提取参数
	if r.Compiled == nil || r.Compiled.Regexp == nil {
		return parameters
	}

	rawMatches := r.Compiled.Regexp.FindStringSubmatch(path)
	if len(rawMatches) <= 1 {
		return parameters
	}
	matches := rawMatches[1:]

	parameterNames := r.getParameterNames()
	for k, v := range parameterNames {
		val := ""
		if k < len(matches) {
			val = matches[k]
		}
		p := &parameter{
			name:  v,
			value: val,
		}
		parameters = append(parameters, p)
		if reqSetter, ok := req.(interface{ SetRouteParam(k, v string) }); ok {
			reqSetter.SetRouteParam(v, val)
		}
	}

	return parameters
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
func (r *Rule) Run(request *context.Request, params ...[]*parameter) (result interface{}) {
	if r == nil || r.handler == nil {
		return nil
	}

	var parsedParams []*parameter
	if len(params) > 0 {
		parsedParams = params[0]
	}

	v := reflect.ValueOf(r.handler)
	switch v.Type().Kind() {
	case reflect.Func:
		in := parseParams(v, request, parsedParams)
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
	r.compileOnce.Do(func() {
		pat := strings.Replace(r.pattern, "/*", "/.*", -1)

		reg, _ := regexp.Compile(`\{\w+\}`)
		regex := reg.ReplaceAllString(pat, "([^/]+)")
		fullRegex := "^" + regex + "$"

		compiledReg, _ := regexp.Compile(fullRegex)

		r.Compiled = &Compiled{
			Regex:  fullRegex,
			Regexp: compiledReg,
		}
	})
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
	paramIdx := 0

	for i := 0; i < needNum; i++ {
		t := valueType.In(i)
		k := t.Kind()

		// 检查是否为 *context.Request 或 context.Request
		if (k == reflect.Ptr && t.Elem().Kind() == reflect.ValueOf(request).Elem().Kind()) ||
			(k == reflect.ValueOf(request).Elem().Kind()) {
			if k == reflect.Ptr {
				in = append(in, reflect.ValueOf(request))
			} else {
				in = append(in, reflect.ValueOf(request).Elem())
			}
			continue
		}

		// 路由正则提取的形参转换
		if paramIdx < len(parameters) {
			strVal := parameters[paramIdx].value
			paramIdx++
			in = append(in, convertParamValue(strVal, t))
		} else {
			in = append(in, reflect.Zero(t))
		}
	}

	return in
}

func convertParamValue(str string, targetType reflect.Type) reflect.Value {
	kind := targetType.Kind()
	if kind == reflect.Ptr {
		elemVal := convertParamValue(str, targetType.Elem())
		ptr := reflect.New(targetType.Elem())
		ptr.Elem().Set(elemVal)
		return ptr
	}

	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, _ := strconv.ParseInt(str, 10, 64)
		return reflect.ValueOf(intVal).Convert(targetType)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, _ := strconv.ParseUint(str, 10, 64)
		return reflect.ValueOf(uintVal).Convert(targetType)
	case reflect.Bool:
		boolVal, _ := strconv.ParseBool(str)
		return reflect.ValueOf(boolVal)
	case reflect.Float32, reflect.Float64:
		floatVal, _ := strconv.ParseFloat(str, 64)
		return reflect.ValueOf(floatVal).Convert(targetType)
	default:
		return reflect.ValueOf(str).Convert(targetType)
	}
}

