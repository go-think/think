package contract

// FormRequest represents an incoming request that defines its own validation rules.
type FormRequest interface {
	Rules() map[string]string
}

// AuthorizesRequests represents an incoming request that verifies authorization.
type AuthorizesRequests interface {
	Authorize() bool
}

// CustomMessages represents an incoming request that provides custom validation error messages.
type CustomMessages interface {
	Messages() map[string]string
}

// ValidatesWhenResolved represents an object that automatically runs its validation
// when resolved by the application service container.
type ValidatesWhenResolved interface {
	ValidateResolved() error
}

// ValidationException is thrown when request data fails validation rules.
type ValidationException struct {
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors"`
	Status  int                 `json:"status"`
}

func (e *ValidationException) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "The given data was invalid."
}

// NewValidationException creates a new validation exception with default 422 status code.
func NewValidationException(errors map[string][]string, message ...string) *ValidationException {
	msg := "The given data was invalid."
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	return &ValidationException{
		Message: msg,
		Errors:  errors,
		Status:  422,
	}
}

// Validator defines the interface for data validation.
type Validator interface {
	Passes() bool
	Fails() bool
	Errors() map[string][]string
	Validated() map[string]any
}
