package apierr

import "fmt"

type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func BadRequest(code, message string) *Error { return New(400, code, message) }
func Unauthorized(message string) *Error     { return New(401, "unauthorized", message) }
func Forbidden(message string) *Error        { return New(403, "forbidden", message) }
func NotFound(resource string) *Error        { return New(404, "not_found", resource+" not found") }
func Conflict(code, message string) *Error   { return New(409, code, message) }
func Entitlement(message string) *Error      { return New(402, "entitlement_denied", message) }
func RateLimited(message string) *Error      { return New(429, "rate_limited", message) }
func Unavailable(code, message string) *Error {
	return New(503, code, message)
}

func Wrap(err error, status int, code, message string) *Error {
	if err == nil {
		return New(status, code, message)
	}
	return New(status, code, fmt.Sprintf("%s: %v", message, err))
}
