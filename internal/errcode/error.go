// Package errcode defines errors safe to return to API clients.
package errcode

type Error struct {
	Code    int
	Message string
	Details []string
}

func (e *Error) Error() string            { return e.Message }
func New(code int, message string) *Error { return &Error{Code: code, Message: message} }
