package pipeline

import (
	"html/template"
	"net/http"

	"github.com/go-think/think/flow"
	"github.com/go-think/think/middleware"
)

type Pipeline struct {
	handlers []middleware.Handler
}

// NewPipeline returns a new Pipeline
func NewPipeline() *Pipeline {
	return &Pipeline{
		handlers: make([]middleware.Handler, 0),
	}
}

// Pipe Push a Middleware Handler to the pipeline
func (p *Pipeline) Pipe(m middleware.Handler) *Pipeline {
	p.handlers = append(p.handlers, m)
	return p
}

// Through Batch push Middleware Handlers to the pipeline
func (p *Pipeline) Through(hls []middleware.Handler) *Pipeline {
	p.handlers = append(p.handlers, hls...)
	return p
}

// Run run the pipeline with a given request in a thread-safe way
func (p *Pipeline) Run(req *flow.Request) interface{} {
	if len(p.handlers) == 0 {
		return nil
	}
	return p.dispatch(0, req)
}

func (p *Pipeline) dispatch(index int, req *flow.Request) interface{} {
	if index >= len(p.handlers) {
		return nil
	}
	handler := p.handlers[index]
	return handler.Process(req, func(nextReq *flow.Request) interface{} {
		return p.dispatch(index+1, nextReq)
	})
}

// ServeHTTP Implement http.Handler safely
func (p *Pipeline) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	request := flow.NewRequest(r)
	request.CookieHandler = flow.ParseCookieHandler()

	result := p.Run(request)

	switch res := result.(type) {
	case *flow.Response:
		res.Send(w)
	case flow.Response:
		res.Send(w)
	case template.HTML:
		flow.NewResponse().SetContent(string(res)).Send(w)
	case http.Handler:
		res.ServeHTTP(w, r)
	default:
		flow.NewResponse().SetContent(flow.FormatContent(result)).Send(w)
	}
}
