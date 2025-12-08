package thinkgo

import (
	"container/list"
	"html/template"
	"net/http"

	"github.com/go-think/think/context"
	"github.com/go-think/think/router"
)

type Pipeline struct {
	handlers []Handler
	pipeline *list.List
	passable *context.Request
}

// Pipeline returns a new Pipeline
func NewPipeline() *Pipeline {
	p := &Pipeline{
		pipeline: list.New(),
	}
	return p
}

// Pipe Push a Middleware Handler to the pipeline
func (p *Pipeline) Pipe(m Handler) *Pipeline {
	p.pipeline.PushBack(m)
	return p
}

// Pipe Batch push Middleware Handlers to the pipeline
func (p *Pipeline) Through(hls []Handler) *Pipeline {
	for _, hl := range hls {
		p.Pipe(hl)
	}
	return p
}

// Passable set the request being sent through the pipeline.
func (p *Pipeline) Passable(passable *context.Request) *Pipeline {
	p.passable = passable
	return p
}

// Run run the pipeline
func (p *Pipeline) Run() interface{} {
	var result interface{}
	e := p.pipeline.Front()
	if e != nil {
		result = p.handler(p.passable, e)
	}
	return result
}

// ServeHTTP
func (p *Pipeline) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	request := context.NewRequest(r)
	request.CookieHandler = context.ParseCookieHandler()
	p.Passable(request)

	result := p.Run()

	switch result.(type) {
	case router.Response:
		result.(router.Response).Send(w)
		break
	case template.HTML:
		Html(string(result.(template.HTML))).Send(w)
		break
	case http.Handler:
		result.(http.Handler).ServeHTTP(w, r)
		break
	default:
		Response(result).Send(w)
		break
	}
}

func (p *Pipeline) handler(passable *context.Request, e *list.Element) interface{} {
	if e == nil {
		return nil
	}
	hl := e.Value.(Handler)
	result := hl.Process(passable, p.closure(e))
	return result
}

func (p *Pipeline) closure(e *list.Element) Closure {
	return func(req *context.Request) interface{} {
		e = e.Next()
		return p.handler(req, e)
	}
}
