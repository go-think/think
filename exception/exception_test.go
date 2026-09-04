package exception

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type CustomIgnoredError struct {
	Msg string
}

func (e *CustomIgnoredError) Error() string {
	return e.Msg
}

func TestExceptionHandler_DontReport(t *testing.T) {
	h := NewHandler(nil).(*Handler)
	h.DontReport(&CustomIgnoredError{})

	// CustomIgnoredError should be skipped from report
	assert.False(t, h.ShouldReport(&CustomIgnoredError{Msg: "skip me"}))

	// Normal errors should be reported
	assert.True(t, h.ShouldReport(errors.New("report me")))
}
