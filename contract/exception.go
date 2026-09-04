package contract

// ExceptionHandler handles exceptions (panics) and converts them into responses.
type ExceptionHandler interface {
	// Report logs or reports the exception.
	Report(err interface{})
	// Render renders the exception into an HTTP response.
	Render(err interface{}) interface{}
}
