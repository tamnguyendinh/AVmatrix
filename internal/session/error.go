package session

import "fmt"

type RuntimeError struct {
	Code    ErrorCode
	Message string
	Status  int
	Details map[string]any
}

func (e *RuntimeError) Error() string {
	return e.Message
}

func NewRuntimeError(code ErrorCode, message string, status int, details map[string]any) *RuntimeError {
	if status == 0 {
		status = 400
	}
	return &RuntimeError{Code: code, Message: message, Status: status, Details: details}
}

func wrapStartError(err error) *RuntimeError {
	if runtimeErr, ok := err.(*RuntimeError); ok {
		return runtimeErr
	}
	return NewRuntimeError(ErrorSessionStartFailed, fmt.Sprint(err), 500, nil)
}
