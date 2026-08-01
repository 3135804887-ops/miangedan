package consent

import "errors"

// Code is a stable consent service error code.
type Code string

// Stable consent error codes shared with the HTTP contract.
const (
	CodeUnauthorized        Code = "unauthorized"
	CodeForbidden           Code = "forbidden"
	CodeNotFound            Code = "not_found"
	CodeConflict            Code = "conflict"
	CodeValidationFailed    Code = "validation_failed"
	CodeIdempotencyConflict Code = "idempotency_conflict"
	CodeRegionMismatch      Code = "region_mismatch"
	CodeInternal            Code = "internal"
)

// DomainError contains only safe, user-facing details.
type DomainError struct {
	Code      Code
	Message   string
	Details   map[string]any
	Retryable bool
	cause     error
}

func (e *DomainError) Error() string { return e.Message }

func (e *DomainError) Unwrap() error { return e.cause }

func domainError(code Code, message string, retryable bool, cause error) *DomainError {
	return &DomainError{Code: code, Message: message, Retryable: retryable, cause: cause}
}

func validationError() *DomainError {
	return domainError(
		CodeValidationFailed,
		"授权请求未受理，未创建或修改授权证据。请修正输入后重试；不计费且不影响评分。",
		false,
		nil,
	)
}

func unauthorizedError() *DomainError {
	return domainError(
		CodeUnauthorized,
		"身份验证失败，授权数据未读取或修改。请重新登录后重试；不计费且不影响评分。",
		false,
		nil,
	)
}

func forbiddenError() *DomainError {
	return domainError(
		CodeForbidden,
		"该授权不符合当前隐私规则，未创建或修改授权证据。其他授权与核心服务状态保持不变；不计费且不影响评分。",
		false,
		nil,
	)
}

func notFoundError() *DomainError {
	return domainError(
		CodeNotFound,
		"未找到可撤回的有效授权，历史授权证据保持不变。请刷新授权中心后重试；不计费且不影响评分。",
		false,
		nil,
	)
}

func conflictError() *DomainError {
	return domainError(
		CodeConflict,
		"当前授权状态已变化，本次未追加新版本。请刷新授权中心后重试；不计费且不影响评分。",
		false,
		nil,
	)
}

func idempotencyConflictError() *DomainError {
	return domainError(
		CodeIdempotencyConflict,
		"同一幂等键对应了不同授权请求；本次未执行，已有授权与审计保持不变。请使用新幂等键重试；不计费且不影响评分。",
		false,
		nil,
	)
}

func internalError(cause error) *DomainError {
	return domainError(
		CodeInternal,
		"授权服务暂时无法完成操作，事务已回滚且在线权限保持原状态。请稍后使用同一幂等键重试；不计费且不影响评分。",
		true,
		cause,
	)
}

// NewValidationError returns the safe transport error for malformed input.
func NewValidationError() *DomainError { return validationError() }

// NewUnauthorizedError returns the safe transport error for failed identity verification.
func NewUnauthorizedError() *DomainError { return unauthorizedError() }

// NewRegionMismatchError returns the safe error for a cross-region token.
func NewRegionMismatchError() *DomainError {
	return domainError(
		CodeRegionMismatch,
		"身份令牌与当前数据区不一致，授权数据未读取或修改。请在账户所属区域重新登录；不计费且不影响评分。",
		false,
		nil,
	)
}

// AsDomainError converts any implementation error to a safe DomainError.
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

// ErrorCodeOf returns only the stable code and never exposes a cause.
func ErrorCodeOf(err error) Code {
	if err == nil {
		return ""
	}
	return AsDomainError(err).Code
}
