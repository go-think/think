package router

import (
	"container/list"

	"github.com/go-think/think/context"
)

type Pipeline struct {
	handlers []Middleware
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
func (p *Pipeline) Pipe(m Middleware) *Pipeline {
	p.pipeline.PushBack(m)
	return p
}

// Pipe Batch push Middleware Handlers to the pipeline
func (p *Pipeline) Through(hls []Middleware) *Pipeline {
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
func (p *Pipeline) Run(destination Middleware) interface{} {
	var result interface{}

	p.Pipe(destination)

	e := p.pipeline.Front()
	if e != nil {
		result = p.handler(p.passable, e)
	}
	return result
}

func (p *Pipeline) handler(passable *context.Request, e *list.Element) interface{} {
	if e == nil {
		return nil
	}
	middleware := e.Value.(Middleware)
	result := middleware(passable, p.closure(e))
	return result
}

func (p *Pipeline) closure(e *list.Element) Closure {
	return func(req *context.Request) interface{} {
		e = e.Next()
		return p.handler(req, e)
	}
}
