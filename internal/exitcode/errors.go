package exitcode

import "fmt"

// Error is an error that carries an exit code.
type Error struct {
	Code    int
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// ExitCode extracts the exit code from an error.
// Returns GeneralError if the error is not an *Error.
func ExitCode(err error) int {
	if err == nil {
		return Success
	}
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	return GeneralError
}

// General returns a general error (exit code 1).
func General(msg string, err error) *Error {
	return &Error{Code: GeneralError, Message: msg, Err: err}
}

// Generalf returns a general error with a formatted message.
func Generalf(format string, args ...any) *Error {
	return &Error{Code: GeneralError, Message: fmt.Sprintf(format, args...)}
}

// Usage returns a usage error (exit code 2).
func Usage(msg string) *Error {
	return &Error{Code: UsageError, Message: msg}
}

// Auth returns an authentication failure error (exit code 3).
func Auth(msg string, err error) *Error {
	return &Error{Code: AuthFailure, Message: msg, Err: err}
}

// NotFoundError returns a not-found error (exit code 4).
func NotFoundError(msg string) *Error {
	return &Error{Code: NotFound, Message: msg}
}
