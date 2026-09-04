package flow

type Session interface {
	Get(name string, value ...interface{}) interface{}

	Set(name string, value interface{})

	Has(name string) bool

	Pull(name string, value ...interface{}) interface{}

	Flash(name string, value interface{})

	Regenerate()

	All() map[string]interface{}

	Remove(name string) interface{}

	Forget(names ...string)

	Clear()

	Save()

	Reflash()

	Keep(names ...string)

	Now(name string, value interface{})

	Token() string

	RegenerateToken() string

	Only(names ...string) map[string]interface{}

	Except(names ...string) map[string]interface{}

	PreviousUrl() string

	SetPreviousUrl(url string)

	Invalidate()

	Increment(key string, amount ...int) int

	Decrement(key string, amount ...int) int
}
