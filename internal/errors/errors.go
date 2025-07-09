package errors

import "fmt"

type ErrorType string

const (
	InvalidRequest   ErrorType = "INVALID_REQUEST"
	NotFound         ErrorType = "NOT_FOUND"
	InsufficientFund ErrorType = "INSUFFICIENT_FUND"
	Conflict         ErrorType = "CONFLICT"
	Internal         ErrorType = "INTERNAL_ERROR"
)

type Error struct {
	Type    ErrorType
	Message string
	Op      string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s [%s]: %s -> %v", e.Type, e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("%s [%s]: %s", e.Type, e.Op, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// 快捷构造函数
func NewInvalidInput(op, field string, value interface{}) *Error {
	return &Error{
		Type:    InvalidRequest,
		Op:      op,
		Message: fmt.Sprintf("invalid %s: %v", field, value),
	}
}

func NewNotFound(op, resource string) *Error {
	return &Error{
		Type:    NotFound,
		Op:      op,
		Message: fmt.Sprintf("%s not found", resource),
	}
}

func NewInsufficientBalance(op string) *Error {
	return &Error{
		Type:    InsufficientFund,
		Op:      op,
		Message: "insufficient balance",
	}
}

func NewCurrencyMismatch(op string) *Error {
	return &Error{
		Type:    InvalidRequest,
		Op:      op,
		Message: "currency mismatch",
	}
}

func NewInternal(op string, err error) *Error {
	return &Error{
		Type:    Internal,
		Op:      op,
		Message: "internal server error",
		Err:     err,
	}
}

func NewConflict(op, msg string) *Error {
	return &Error{
		Type:    Conflict,
		Op:      op,
		Message: msg,
	}
}

// 辅助函数
func IsNotFound(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Type == NotFound
	}
	return false
}

func WrapInternal(op string, err error) error {
	if err == nil {
		return nil
	}
	return NewInternal(op, err)
}

func IfInternalError(op string, err error) error {
	if err == nil {
		return nil
	}
	return NewInternal(op, err)
}
