package think

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-think/think/context"
)

// Json Create a new HTTP Response with JSON data
func Json(v interface{}) *context.Response {
	c, err := json.Marshal(v)
	if err != nil {
		return context.NewResponse().SetContent("").SetContentType("application/json")
	}
	return context.NewResponse().SetContent(string(c)).SetContentType("application/json")
}

// Text Create a new HTTP Response with TEXT data
func Text(s string) *context.Response {
	return context.NewResponse().SetContent(s).SetContentType("text/plain")
}

// Html Create a new HTTP Response with HTML data
func Html(s string) *context.Response {
	return context.NewResponse().SetContent(s)
}

func FormatContent(v interface{}) string {
	if v == nil {
		return ""
	}
	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Bool:
		return fmt.Sprintf("%t", v)
	case reflect.String:
		return fmt.Sprintf("%s", v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v)
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

func Response(v interface{}) *context.Response {
	r := context.NewResponse()
	if v == nil {
		return r.SetContent("")
	}

	content := FormatContent(v)
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Map || t.Kind() == reflect.Slice || t.Kind() == reflect.Struct {
		r.SetContentType("application/json")
	} else {
		r.SetContentType("text/plain")
	}
	r.SetContent(content)

	return r
}

