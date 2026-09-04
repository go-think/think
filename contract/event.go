package contract

// EventDispatcher defines the contract for dispatching events and listening to them.
type EventDispatcher interface {
	// Dispatch fires an event with the given payload.
	Dispatch(event string, payload interface{})
	
	// Listen registers a listener for a given event.
	Listen(event string, listener interface{})

	// Until dispatches an event until the first listener returns a non-nil result.
	Until(event string, payload interface{}) interface{}
}
