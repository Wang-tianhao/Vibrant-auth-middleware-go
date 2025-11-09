package jwtauth

import "fmt"

// ErrorCode represents a validation error code
type ErrorCode string

const (
	ErrExpired                  ErrorCode = "EXPIRED"
	ErrInvalidSignature         ErrorCode = "INVALID_SIGNATURE"
	ErrMissingToken             ErrorCode = "MISSING_TOKEN"
	ErrMalformed                ErrorCode = "MALFORMED"
	ErrAlgorithmMismatch        ErrorCode = "ALGORITHM_MISMATCH" // Deprecated: use ErrUnsupportedAlgorithm
	ErrNoneAlgorithm            ErrorCode = "NONE_ALGORITHM"
	ErrConfigError              ErrorCode = "CONFIG_ERROR"
	ErrUnsupportedAlgorithm     ErrorCode = "UNSUPPORTED_ALGORITHM"
	ErrMalformedAlgorithmHeader ErrorCode = "MALFORMED_ALGORITHM_HEADER"
)

// ValidationError represents a JWT validation error with a code and message
type ValidationError struct {
	Code     ErrorCode
	Message  string
	Internal error
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap implements the error unwrapping interface
func (e *ValidationError) Unwrap() error {
	return e.Internal
}

// NewValidationError creates a new validation error
func NewValidationError(code ErrorCode, message string, internal error) *ValidationError {
	return &ValidationError{
		Code:     code,
		Message:  message,
		Internal: internal,
	}
}
