package apperr

import (
	"fmt"
	"net/http"
)

// Error 表示可直接映射为 HTTP 响应的业务错误。
type Error struct {
	Status  int
	Code    int
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

func BadRequest(message string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: 40001, Message: message}
}

func Unauthorized(message string) *Error {
	return &Error{Status: http.StatusUnauthorized, Code: 40101, Message: message}
}

func Forbidden(message string) *Error {
	return &Error{Status: http.StatusForbidden, Code: 40301, Message: message}
}

func NotFound(message string) *Error {
	return &Error{Status: http.StatusNotFound, Code: 40401, Message: message}
}

func Conflict(message string) *Error {
	return &Error{Status: http.StatusConflict, Code: 40901, Message: message}
}

func Internal(message string, err error) *Error {
	return &Error{Status: http.StatusInternalServerError, Code: 50001, Message: message, Err: err}
}
