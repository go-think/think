package event

import (
	"reflect"
	"sync"

	"github.com/go-think/think/contract"
)

type Dispatcher struct {
	listeners map[string][]interface{}
	mu        sync.RWMutex
}

func NewDispatcher() contract.EventDispatcher {
	return &Dispatcher{
		listeners: make(map[string][]interface{}),
	}
}

func (d *Dispatcher) Listen(event string, listener interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners[event] = append(d.listeners[event], listener)
}

func (d *Dispatcher) Dispatch(event string, payload interface{}) {
	d.mu.RLock()
	listeners, ok := d.listeners[event]
	d.mu.RUnlock()

	if !ok {
		return
	}

	for _, listener := range listeners {
		d.executeListener(listener, payload)
	}
}

// Until dispatches an event until the first listener returns a non-nil result.
func (d *Dispatcher) Until(event string, payload interface{}) interface{} {
	d.mu.RLock()
	listeners, ok := d.listeners[event]
	d.mu.RUnlock()

	if !ok {
		return nil
	}

	for _, listener := range listeners {
		res := d.executeListener(listener, payload)
		if res != nil {
			return res
		}
	}
	return nil
}

func (d *Dispatcher) executeListener(listener interface{}, payload interface{}) interface{} {
	v := reflect.ValueOf(listener)
	if v.Kind() != reflect.Func {
		return nil
	}

	var out []reflect.Value
	if v.Type().NumIn() == 1 {
		var inVal reflect.Value
		if payload != nil {
			inVal = reflect.ValueOf(payload)
		} else {
			inVal = reflect.Zero(v.Type().In(0))
		}
		out = v.Call([]reflect.Value{inVal})
	} else if v.Type().NumIn() == 0 {
		out = v.Call(nil)
	}

	if len(out) > 0 {
		return out[0].Interface()
	}
	return nil
}
