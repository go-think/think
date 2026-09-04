package container

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type binding struct {
	factory any
	shared  bool
	scoped  bool
}

// Container is the IoC Service Container with full Go 1.27 method generics.
type Container struct {
	bindings        map[string]binding
	instances       map[string]any
	scopedInstances map[string]any
	aliases         map[string]string
	tags            map[string][]string
	mu              sync.RWMutex
}

// New creates a new IoC container.
func New() *Container {
	return &Container{
		bindings:        make(map[string]binding),
		instances:       make(map[string]any),
		scopedInstances: make(map[string]any),
		aliases:         make(map[string]string),
		tags:            make(map[string][]string),
	}
}

// Alias registers an alias for an abstract type or key in the container.
func (c *Container) Alias(alias, abstract string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliases[alias] = abstract
}

// resolveKey derives the binding key from type T, or uses the custom name if provided.
func resolveKey[T any](name ...string) string {
	if len(name) > 0 && name[0] != "" {
		return name[0]
	}
	var zero T
	t := reflect.TypeOf(&zero).Elem()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.PkgPath() != "" {
		return t.PkgPath() + "." + t.Name()
	}
	return t.String()
}

// Make resolves a strongly-typed service of type T from the container.
func (c *Container) Make[T any](name ...string) T {
	key := resolveKey[T](name...)
	raw := c.MakeByName(key)
	if raw == nil && len(name) > 0 {
		// Fallback 1: try default full type key
		defaultKey := resolveKey[T]()
		raw = c.MakeByName(defaultKey)
		// Fallback 2: try short type name
		if raw == nil {
			raw = c.MakeByName(shortName[T]())
		}
	}
	if raw == nil {
		var zero T
		return zero
	}
	if val, ok := raw.(T); ok {
		return val
	}
	var zero T
	panic(fmt.Sprintf("container: service [%s] expected %T, got %T", key, zero, raw))
}

// MakeByName resolves a service dynamically by name or type string.
func (c *Container) MakeByName(key string) any {
	c.mu.RLock()
	if target, ok := c.aliases[key]; ok {
		key = target
	}
	if inst, ok := c.instances[key]; ok {
		c.mu.RUnlock()
		return inst
	}
	if scopedInst, ok := c.scopedInstances[key]; ok {
		c.mu.RUnlock()
		return scopedInst
	}
	b, ok := c.bindings[key]
	c.mu.RUnlock()

	if !ok {
		// Fallback: match by short name if key contains dots (e.g. "contract.Config" -> "Config")
		if idx := strings.LastIndex(key, "."); idx != -1 {
			short := key[idx+1:]
			c.mu.RLock()
			if inst, ok := c.instances[short]; ok {
				c.mu.RUnlock()
				return inst
			}
			if scopedInst, ok := c.scopedInstances[short]; ok {
				c.mu.RUnlock()
				return scopedInst
			}
			b, ok = c.bindings[short]
			c.mu.RUnlock()
			if !ok {
				return nil
			}
		} else {
			return nil
		}
	}

	val := c.buildAny(b.factory)

	if b.shared {
		c.mu.Lock()
		c.instances[key] = val
		c.mu.Unlock()
	} else if b.scoped {
		c.mu.Lock()
		c.scopedInstances[key] = val
		c.mu.Unlock()
	}

	return val
}

func (c *Container) buildAny(factory any) any {
	val := reflect.ValueOf(factory)
	if val.Kind() == reflect.Func {
		out := c.Call(factory)
		if len(out) > 0 {
			return out[0]
		}
		return nil
	}
	return factory
}

func shortName[T any]() string {
	var zero T
	t := reflect.TypeOf(&zero).Elem()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// Singleton registers a typed shared singleton binding in the container.
func (c *Container) Singleton[T any](factory any, name ...string) {
	key := resolveKey[T](name...)
	c.mu.Lock()
	defer c.mu.Unlock()
	b := binding{factory: factory, shared: true, scoped: false}
	c.bindings[key] = b
	if len(name) == 0 {
		if sn := shortName[T](); sn != "" && sn != key {
			c.bindings[sn] = b
		}
	}
}

// Bind registers a typed transient binding in the container.
func (c *Container) Bind[T any](factory any, name ...string) {
	key := resolveKey[T](name...)
	c.mu.Lock()
	defer c.mu.Unlock()
	b := binding{factory: factory, shared: false, scoped: false}
	c.bindings[key] = b
	if len(name) == 0 {
		if sn := shortName[T](); sn != "" && sn != key {
			c.bindings[sn] = b
		}
	}
}

// Scoped registers a typed scoped binding in the container.
func (c *Container) Scoped[T any](factory any, name ...string) {
	key := resolveKey[T](name...)
	c.mu.Lock()
	defer c.mu.Unlock()
	b := binding{factory: factory, shared: false, scoped: true}
	c.bindings[key] = b
	if len(name) == 0 {
		if sn := shortName[T](); sn != "" && sn != key {
			c.bindings[sn] = b
		}
	}
}

// Instance registers an existing typed instance in the container.
func (c *Container) Instance[T any](instance T, name ...string) {
	key := resolveKey[T](name...)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.instances[key] = instance
	if len(name) == 0 {
		if sn := shortName[T](); sn != "" && sn != key {
			c.instances[sn] = instance
		}
	}
}

// FlushScoped flushes all current scoped instances.
func (c *Container) FlushScoped() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scopedInstances = make(map[string]any)
}

// Tag assigns a tag to a given service key.
func (c *Container) Tag(target string, tag string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tags[tag] = append(c.tags[tag], target)
}

// Tagged resolves all services tagged with the given tag.
func (c *Container) Tagged[T any](tag string) []T {
	c.mu.RLock()
	targets, ok := c.tags[tag]
	c.mu.RUnlock()

	if !ok {
		return nil
	}

	var results []T
	for _, target := range targets {
		results = append(results, c.Make[T](target))
	}
	return results
}

// Call calls the given function and injects its dependencies using reflection.
func (c *Container) Call(function any) []any {
	val := reflect.ValueOf(function)
	if val.Kind() != reflect.Func {
		panic(fmt.Sprintf("container: unable to Call non-function %T", function))
	}

	typ := val.Type()
	in := make([]reflect.Value, typ.NumIn())

	for i := 0; i < typ.NumIn(); i++ {
		paramType := typ.In(i)
		abstract := getAbstractName(paramType)

		inst := c.MakeByName(abstract)
		if inst == nil {
			inst = c.MakeByName(paramType.Name())
		}
		if inst == nil && paramType.Kind() == reflect.Ptr {
			inst = c.MakeByName(paramType.Elem().Name())
		}

		if inst != nil {
			in[i] = reflect.ValueOf(inst)
		} else {
			in[i] = reflect.Zero(paramType)
		}
	}

	out := val.Call(in)
	result := make([]any, len(out))
	for i, v := range out {
		result[i] = v.Interface()
	}
	return result
}

func getAbstractName(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.PkgPath() != "" {
		return t.PkgPath() + "." + t.Name()
	}
	return t.Name()
}
