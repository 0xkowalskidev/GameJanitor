package service

import "fmt"

// ServiceError is a typed error that maps service-layer errors to HTTP status codes.
type ServiceError struct {
	Code    int
	Message string
}

func (e *ServiceError) Error() string { return e.Message }

func ErrNotFound(msg string) error    { return &ServiceError{Code: 404, Message: msg} }
func ErrConflict(msg string) error    { return &ServiceError{Code: 409, Message: msg} }
func ErrBadRequest(msg string) error  { return &ServiceError{Code: 400, Message: msg} }

func ErrNotFoundf(format string, args ...any) error   { return ErrNotFound(fmt.Sprintf(format, args...)) }
func ErrConflictf(format string, args ...any) error    { return ErrConflict(fmt.Sprintf(format, args...)) }
func ErrBadRequestf(format string, args ...any) error  { return ErrBadRequest(fmt.Sprintf(format, args...)) }
