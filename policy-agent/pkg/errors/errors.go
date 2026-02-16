package errors

import "fmt"

// ErrorCode represents categorized error types
type ErrorCode string

const (
	ErrCodePolicyViolation    ErrorCode = "POLICY_VIOLATION"
	ErrCodeParseFailed        ErrorCode = "PARSE_FAILED"
	ErrCodePolicyLoadFailed   ErrorCode = "POLICY_LOAD_FAILED"
	ErrCodeAIRequestFailed    ErrorCode = "AI_REQUEST_FAILED"
	ErrCodeConfigInvalid      ErrorCode = "CONFIG_INVALID"
	ErrCodeUnknownDomain      ErrorCode = "UNKNOWN_DOMAIN"
	ErrCodeValidationFailed   ErrorCode = "VALIDATION_FAILED"
	ErrCodeFileNotFound       ErrorCode = "FILE_NOT_FOUND"
	ErrCodeUnsupportedFormat  ErrorCode = "UNSUPPORTED_FORMAT"
)

// Error represents a policy agent error
type Error struct {
	Code       ErrorCode
	Message    string
	Domain     string
	Resource   string
	Details    map[string]interface{}
	Underlying error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Domain != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Domain, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Underlying
}

// New creates a new Error
func New(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with context
func Wrap(err error, code ErrorCode, message string) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		Underlying: err,
		Details:    make(map[string]interface{}),
	}
}

// WithDomain adds domain context to the error
func (e *Error) WithDomain(domain string) *Error {
	e.Domain = domain
	return e
}

// WithResource adds resource context to the error
func (e *Error) WithResource(resource string) *Error {
	e.Resource = resource
	return e
}

// WithDetails adds additional details to the error
func (e *Error) WithDetails(key string, value interface{}) *Error {
	e.Details[key] = value
	return e
}
