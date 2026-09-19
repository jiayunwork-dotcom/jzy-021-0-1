package main

import "fmt"

// ErrorType 错误类型，随响应体返回给调用方。
type ErrorType string

const (
	ErrTypeMissingField       ErrorType = "missing_field"
	ErrTypeDuplicateID        ErrorType = "duplicate_id"
	ErrTypeUnknownPredecessor ErrorType = "unknown_predecessor"
	ErrTypeSelfLoop           ErrorType = "self_loop"
	ErrTypeCycle              ErrorType = "cycle"
	ErrTypeInvalidDuration    ErrorType = "invalid_duration"
	ErrTypeInvalidThreePoint  ErrorType = "invalid_three_point"
	ErrTypeBadRequest         ErrorType = "bad_request"
	ErrTypeNotFound           ErrorType = "not_found"
	ErrTypeInternal           ErrorType = "internal"
)

// APIError 带类型的错误。
type APIError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func newError(t ErrorType, format string, args ...any) *APIError {
	return &APIError{Type: t, Message: fmt.Sprintf(format, args...)}
}

// statusOf 错误类型到 HTTP 状态码的映射。
func statusOf(t ErrorType) int {
	switch t {
	case ErrTypeNotFound:
		return 404
	case ErrTypeInternal:
		return 500
	default:
		return 400
	}
}
