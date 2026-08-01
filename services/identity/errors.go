package identity

import "errors"

// ErrorCode is stable across the Go service and OpenAPI contract.
type ErrorCode string

const (
	CodeUnauthorized             ErrorCode = "unauthorized"
	CodeForbidden                ErrorCode = "forbidden"
	CodeNotFound                 ErrorCode = "not_found"
	CodeConflict                 ErrorCode = "conflict"
	CodeValidationFailed         ErrorCode = "validation_failed"
	CodeIdempotencyConflict      ErrorCode = "idempotency_conflict"
	CodeRateLimited              ErrorCode = "rate_limited"
	CodeRiskVerificationRequired ErrorCode = "risk_verification_required"
	CodeVerificationInvalid      ErrorCode = "verification_invalid"
	CodeVerificationExpired      ErrorCode = "verification_expired"
	CodeIdentityConflict         ErrorCode = "identity_conflict"
	CodeProviderUnavailable      ErrorCode = "provider_unavailable"
	CodeRegionMismatch           ErrorCode = "region_mismatch"
	CodeInternal                 ErrorCode = "internal"
)

// DomainError carries a safe user-facing message. Messages and Details must
// never contain an email address, provider subject, authorization code, OTP or
// token. Retryable errors are not permanently cached by idempotency handling.
type DomainError struct {
	Code      ErrorCode
	Message   string
	Details   map[string]any
	Retryable bool
	cause     error
}

func (e *DomainError) Error() string { return e.Message }

func (e *DomainError) Unwrap() error { return e.cause }

func domainError(code ErrorCode, message string, retryable bool, cause error) *DomainError {
	return &DomainError{Code: code, Message: message, Retryable: retryable, cause: cause}
}

func validationError() *DomainError {
	return domainError(
		CodeValidationFailed,
		"请求未受理，未创建或修改身份数据。请修正输入后重试；不计费且不影响评分。",
		false,
		nil,
	)
}

func internalError(cause error) *DomainError {
	return domainError(
		CodeInternal,
		"身份服务暂时无法完成操作，已有账户数据保持不变。请稍后重试；不计费且不影响评分。",
		true,
		cause,
	)
}

// AsDomainError returns a safe DomainError for any service error.
func AsDomainError(err error) *DomainError {
	if err == nil {
		return nil
	}
	var domain *DomainError
	if errors.As(err, &domain) {
		return domain
	}
	return internalError(err)
}

// ErrorCodeOf is a test/caller helper that never exposes an underlying error.
func ErrorCodeOf(err error) ErrorCode {
	return AsDomainError(err).Code
}
