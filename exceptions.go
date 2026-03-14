package x402

import "fmt"

// X402Error is the base x402 exception
type X402Error struct {
	Msg string
}

func (e *X402Error) Error() string {
	return e.Msg
}

// SignatureError is signature-related error
type SignatureError struct {
	*X402Error
}

// SignatureVerificationError is signature verification failed
type SignatureVerificationError struct {
	*SignatureError
}

// SignatureCreationError is signature creation failed
type SignatureCreationError struct {
	*SignatureError
}

// AllowanceError is allowance-related error
type AllowanceError struct {
	*X402Error
}

// InsufficientAllowanceError is insufficient allowance
type InsufficientAllowanceError struct {
	*AllowanceError
}

// AllowanceCheckError is failed to check allowance
type AllowanceCheckError struct {
	*AllowanceError
}

// SettlementError is settlement-related error
type SettlementError struct {
	*X402Error
}

// TransactionError is transaction-related error
type TransactionError struct {
	*X402Error
}

// TransactionTimeoutError is transaction timeout
type TransactionTimeoutError struct {
	*TransactionError
}

// TransactionFailedError is transaction execution failed
type TransactionFailedError struct {
	*TransactionError
}

// ValidationError is validation-related error
type ValidationError struct {
	*X402Error
}

// PermitValidationError is permit validation failed
type PermitValidationError struct {
	*ValidationError
	Reason string
}

func (e *PermitValidationError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return e.Reason
}

// ConfigurationError is configuration-related error
type ConfigurationError struct {
	*X402Error
}

// UnsupportedNetworkError is unsupported network
type UnsupportedNetworkError struct {
	*ConfigurationError
}

// UnknownTokenError is unknown token
type UnknownTokenError struct {
	*ConfigurationError
}

// Helper constructors
func newX402Error(msg string) *X402Error {
	return &X402Error{Msg: msg}
}

func NewSignatureVerificationError(msg string) error {
	return &SignatureVerificationError{
		&SignatureError{newX402Error(msg)},
	}
}

func NewSignatureCreationError(msg string) error {
	return &SignatureCreationError{
		&SignatureError{newX402Error(msg)},
	}
}

func NewInsufficientAllowanceError(msg string) error {
	return &InsufficientAllowanceError{
		&AllowanceError{newX402Error(msg)},
	}
}

func NewUnsupportedNetworkError(msg string) error {
	return &UnsupportedNetworkError{
		&ConfigurationError{newX402Error(msg)},
	}
}

func NewSettlementError(msg string) error {
	return &SettlementError{newX402Error(msg)}
}

func NewPermitValidationError(reason, message string) error {
	msg := reason
	if message != "" {
		msg = message
	}
	return &PermitValidationError{
		ValidationError: &ValidationError{newX402Error(msg)},
		Reason:          reason,
	}
}

func NewUnknownTokenError(format string, args ...any) error {
	return &UnknownTokenError{
		&ConfigurationError{newX402Error(fmt.Sprintf(format, args...))},
	}
}
