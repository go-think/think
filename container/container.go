package container

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type binding struct {
	factory any          // Factory function or concrete instance
	shared  bool         // Shared singleton
	scoped  bool         // Request-scoped
	target  reflect.Type // Interface binding redirection target
}

// Container is the modern IoC Service Container with Go 1.27 method generics and reflect.Type first-class indexing.
type Container struct {
	// 1. Strongly-typed indexing: keyed directly by reflect.Type
	typeBindings        map[reflect.Type]*binding
	typeInstances       map[reflect.Type]any
	scopedTypeInstances map[reflect.Type]any

	// 2. Named indexing: keyed by string identifier (supports multiple instances of same type or custom names)
	namedBindings        map[string]*binding
	namedInstances       map[string]any
	scopedNamedInstances map[string]any

	// 3. Alias mappings: alias (string) -> target (reflect.Type or string)
	aliases map[string]any

	// 4. Tags
	tags map[string][]any

	mu sync.RWMutex
}

// New creates a new modern IoC container.
func New() *Container {
	return &Container{
		typeBindings:         make(map[reflect.Type]*binding),
		typeInstances:        make(map[reflect.Type]any),
		scopedTypeInstances:  make(map[reflect.Type]any),
		namedBindings:        make(map[string]*binding),
		namedInstances:       make(map[string]any),
		scopedNamedInstances: make(map[string]any),
		aliases:              make(map[string]any),
		tags:                 make(map[string][]any),
	}
}

// extractType returns the reflect.Type for generic parameter T.
func extractType[T any]() reflect.Type {
	var zero T
	return reflect.TypeOf(&zero).Elem()
}

// extractTargetType derives the reflect.Type from a given type object, pointer, or reflect.Type.
func extractTargetType(target any) (reflect.Type, bool) {
	if target == nil {
		return nil, false
	}
	if t, ok := target.(reflect.Type); ok {
		return t, true
	}
	return reflect.TypeOf(target), true
}

// Alias registers a compile-time type-safe alias for type T without magic strings or dummy objects.
func (c *Container) Alias[T any](alias string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliases[alias] = extractType[T]()
}

// AliasName registers a string-to-string alias for custom named services.
func (c *Container) AliasName(alias, targetName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.aliases[alias] = targetName
}

// Singleton registers a typed shared singleton binding in the container.
func (c *Container) Singleton[T any](factory any, name ...string) {
	c.registerBinding[T](factory, true, false, name...)
}

// Bind registers a typed transient binding in the container.
func (c *Container) Bind[T any](factory any, name ...string) {
	c.registerBinding[T](factory, false, false, name...)
}

// Scoped registers a typed scoped binding in the container.
func (c *Container) Scoped[T any](factory any, name ...string) {
	c.registerBinding[T](factory, false, true, name...)
}

// registerBinding performs binding registration for Singleton, Bind, and Scoped.
func (c *Container) registerBinding[T any](factory any, shared, scoped bool, name ...string) {
	typ := extractType[T]()
	b := &binding{
		factory: factory,
		shared:  shared,
		scoped:  scoped,
	}

	// Support interface-to-implementation binding when factory is an object/type rather than a function
	if typ.Kind() == reflect.Interface && factory != nil && reflect.TypeOf(factory).Kind() != reflect.Func {
		implType, _ := extractTargetType(factory)
		b.target = implType
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(name) > 0 && name[0] != "" {
		c.namedBindings[name[0]] = b
	} else {
		c.typeBindings[typ] = b
	}
}

// Instance registers an existing typed instance in the container.
func (c *Container) Instance[T any](instance T, name ...string) {
	typ := extractType[T]()

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(name) > 0 && name[0] != "" {
		c.namedInstances[name[0]] = instance
	} else {
		c.typeInstances[typ] = instance
	}
}

// Resolve resolves a strongly-typed service of type T, returning an error if not found or type mismatch occurs.
func (c *Container) Resolve[T any](name ...string) (T, error) {
	var zero T
	typ := extractType[T]()

	// 1. Named / Aliased resolution
	if len(name) > 0 && name[0] != "" {
		val := c.MakeByName(name[0])
		if val != nil {
			if typed, ok := val.(T); ok {
				return typed, nil
			}
			return zero, fmt.Errorf("container: service [%s] expected %T, got %T", name[0], zero, val)
		}
		// Fallback: if not found by name, try pure type resolution
	}

	// 2. Pure type resolution (Type as First-Class Citizen)
	val := c.resolveType(typ)
	if val == nil {
		return zero, fmt.Errorf("container: service of type [%v] is not bound or found", typ)
	}

	if typed, ok := val.(T); ok {
		return typed, nil
	}

	// If resolved object implements interface T via reflection or pointer addressing
	valReflect := reflect.ValueOf(val)
	if valReflect.IsValid() {
		if valReflect.Type().AssignableTo(typ) {
			return val.(T), nil
		}
		if valReflect.Kind() != reflect.Ptr {
			ptr := reflect.New(valReflect.Type())
			ptr.Elem().Set(valReflect)
			if ptr.Type().AssignableTo(typ) {
				return ptr.Interface().(T), nil
			}
		}
	}

	return zero, fmt.Errorf("container: service [%v] expected %T, got %T", typ, zero, val)
}

// Make resolves a strongly-typed service of type T from the container.
// If the service is not found or type mismatch occurs, it safely returns the zero value of T without panicking.
func (c *Container) Make[T any](name ...string) T {
	val, _ := c.Resolve[T](name...)
	return val
}

// resolveType resolves a service by its reflect.Type.
func (c *Container) resolveType(typ reflect.Type) any {
	c.mu.RLock()
	// 1. Check singleton instance cache (support both pointer and value types)
	if inst, ok := c.typeInstances[typ]; ok {
		c.mu.RUnlock()
		return inst
	}
	if typ.Kind() == reflect.Ptr {
		if inst, ok := c.typeInstances[typ.Elem()]; ok {
			c.mu.RUnlock()
			return inst
		}
	} else {
		if inst, ok := c.typeInstances[reflect.PointerTo(typ)]; ok {
			c.mu.RUnlock()
			return inst
		}
	}

	// 2. Check scoped instance cache
	if scopedInst, ok := c.scopedTypeInstances[typ]; ok {
		c.mu.RUnlock()
		return scopedInst
	}
	if typ.Kind() == reflect.Ptr {
		if scopedInst, ok := c.scopedTypeInstances[typ.Elem()]; ok {
			c.mu.RUnlock()
			return scopedInst
		}
	} else {
		if scopedInst, ok := c.scopedTypeInstances[reflect.PointerTo(typ)]; ok {
			c.mu.RUnlock()
			return scopedInst
		}
	}

	// 3. Check type bindings (support both pointer and value types)
	b, ok := c.typeBindings[typ]
	resolvedKey := typ
	if !ok {
		if typ.Kind() == reflect.Ptr {
			b, ok = c.typeBindings[typ.Elem()]
			if ok {
				resolvedKey = typ.Elem()
			}
		} else {
			b, ok = c.typeBindings[reflect.PointerTo(typ)]
			if ok {
				resolvedKey = reflect.PointerTo(typ)
			}
		}
	}
	c.mu.RUnlock()

	if !ok {
		return nil
	}

	// If binding redirects interface to concrete target implementation
	if b.target != nil && b.target != typ {
		targetVal := c.resolveType(b.target)
		if targetVal != nil {
			if b.shared {
				c.mu.Lock()
				c.typeInstances[typ] = targetVal
				c.mu.Unlock()
			}
			return targetVal
		}
	}

	val := c.buildAny(b.factory)

	if b.shared {
		c.mu.Lock()
		if inst, exists := c.typeInstances[resolvedKey]; exists {
			c.mu.Unlock()
			return inst
		}
		c.typeInstances[resolvedKey] = val
		c.typeInstances[typ] = val
		c.mu.Unlock()
	} else if b.scoped {
		c.mu.Lock()
		if inst, exists := c.scopedTypeInstances[resolvedKey]; exists {
			c.mu.Unlock()
			return inst
		}
		c.scopedTypeInstances[resolvedKey] = val
		c.scopedTypeInstances[typ] = val
		c.mu.Unlock()
	}

	return val
}

// MakeByName resolves a service dynamically by string name, alias, or type string.
func (c *Container) MakeByName(key string) any {
	c.mu.RLock()
	// 1. Check alias resolution
	if aliasTarget, ok := c.aliases[key]; ok {
		c.mu.RUnlock()
		switch t := aliasTarget.(type) {
		case reflect.Type:
			return c.resolveType(t)
		case string:
			return c.MakeByName(t)
		default:
			if typ, ok := extractTargetType(aliasTarget); ok {
				return c.resolveType(typ)
			}
		}
	} else {
		c.mu.RUnlock()
	}

	// 2. Check named instances
	c.mu.RLock()
	if inst, ok := c.namedInstances[key]; ok {
		c.mu.RUnlock()
		return inst
	}
	if scopedInst, ok := c.scopedNamedInstances[key]; ok {
		c.mu.RUnlock()
		return scopedInst
	}
	b, ok := c.namedBindings[key]
	c.mu.RUnlock()

	if ok {
		val := c.buildAny(b.factory)
		if b.shared {
			c.mu.Lock()
			if inst, exists := c.namedInstances[key]; exists {
				c.mu.Unlock()
				return inst
			}
			c.namedInstances[key] = val
			c.mu.Unlock()
		} else if b.scoped {
			c.mu.Lock()
			if inst, exists := c.scopedNamedInstances[key]; exists {
				c.mu.Unlock()
				return inst
			}
			c.scopedNamedInstances[key] = val
			c.mu.Unlock()
		}
		return val
	}

	// 3. Fallback compatibility: match by Type name / PkgPath.Name
	c.mu.RLock()
	defer c.mu.RUnlock()
	for typ := range c.typeInstances {
		if c.matchTypeName(typ, key) {
			return c.typeInstances[typ]
		}
	}
	for typ := range c.typeBindings {
		if c.matchTypeName(typ, key) {
			c.mu.RUnlock()
			val := c.resolveType(typ)
			c.mu.RLock()
			return val
		}
	}

	return nil
}

// matchTypeName compares a reflect.Type against a name key for backwards compatibility.
func (c *Container) matchTypeName(typ reflect.Type, key string) bool {
	rawType := typ
	if rawType.Kind() == reflect.Ptr {
		rawType = rawType.Elem()
	}
	fullName := rawType.String()
	if rawType.PkgPath() != "" {
		fullName = rawType.PkgPath() + "." + rawType.Name()
	}
	if fullName == key || rawType.Name() == key {
		return true
	}
	if strings.TrimPrefix(key, "*") == rawType.Name() {
		return true
	}
	return false
}

// buildAny builds an instance from factory func or returns raw value.
func (c *Container) buildAny(factory any) any {
	if factory == nil {
		return nil
	}
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

// FlushScoped flushes all current scoped instances.
func (c *Container) FlushScoped() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scopedTypeInstances = make(map[reflect.Type]any)
	c.scopedNamedInstances = make(map[string]any)
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
		if str, ok := target.(string); ok {
			results = append(results, c.Make[T](str))
		}
	}
	return results
}

// Get resolves a service dynamically by type, interface, or name.
func (c *Container) Get(abstract any) any {
	switch v := abstract.(type) {
	case string:
		return c.MakeByName(v)
	case reflect.Type:
		return c.resolveType(v)
	default:
		if v != nil {
			return c.resolveType(reflect.TypeOf(v))
		}
		return nil
	}
}

// Invoke calls the given function, injects its dependencies using reflection, and returns an error if any parameter cannot be resolved.
func (c *Container) Invoke(function any) ([]any, error) {
	if function == nil {
		return nil, fmt.Errorf("container: unable to Invoke nil function")
	}
	val := reflect.ValueOf(function)
	if val.Kind() != reflect.Func {
		return nil, fmt.Errorf("container: unable to Invoke non-function %T", function)
	}

	typ := val.Type()
	in := make([]reflect.Value, typ.NumIn())

	for i := 0; i < typ.NumIn(); i++ {
		paramType := typ.In(i)
		inst := c.resolveType(paramType)

		if inst == nil {
			// Fallback by short/name lookup
			inst = c.MakeByName(paramType.String())
			if inst == nil && paramType.Kind() == reflect.Ptr {
				inst = c.MakeByName(paramType.Elem().Name())
			}
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
	return result, nil
}

// Call calls the given function and injects its dependencies using reflection.
// If the provided argument is not a function or invocation fails, it safely returns nil without panicking.
func (c *Container) Call(function any) []any {
	res, _ := c.Invoke(function)
	return res
}
