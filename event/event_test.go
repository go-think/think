package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventDispatcher_Until(t *testing.T) {
	d := NewDispatcher()

	calls := 0
	d.Listen("order.creating", func(data string) interface{} {
		calls++
		if data == "block" {
			return "blocked_by_first"
		}
		return nil
	})

	d.Listen("order.creating", func(data string) interface{} {
		calls++
		return "reached_second"
	})

	// 1. Should be blocked by first listener
	res1 := d.Until("order.creating", "block")
	assert.Equal(t, "blocked_by_first", res1)
	assert.Equal(t, 1, calls)

	// 2. First listener returns nil, reaches second listener
	calls = 0
	res2 := d.Until("order.creating", "pass")
	assert.Equal(t, "reached_second", res2)
	assert.Equal(t, 2, calls)
}
