package apperror

import "net/http"

type Code string

const (
	BadRequest Code = "BAD_REQUEST"
	NotFound   Code = "NOT_FOUND"
	Internal   Code = "INTERNAL"
	Conflict   Code = "CONFLICT"
)

type AppError struct {
	code    Code
	message string
}

func New(code Code, message string) *AppError {
	return &AppError{code: code, message: message}
}

func (e *AppError) Error() string   { return e.message }
func (e *AppError) Code() Code      { return e.code }
func (e *AppError) Message() string { return e.message }

func (e *AppError) HTTPStatus() int {
	switch e.code {
	case BadRequest:
		return http.StatusBadRequest
	case NotFound:
		return http.StatusNotFound
	case Conflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
