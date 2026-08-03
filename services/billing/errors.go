package billing

import "errors"

// 错误集。
var (
	ErrInvalidInput  = errors.New("invalid billing input")
	ErrNotFound      = errors.New("billing record not found")
	ErrQuoteFrozen   = errors.New("billing version frozen after start")
	ErrInsufficient  = errors.New("insufficient entitlement")
	ErrStateConflict = errors.New("billing state conflict")
)
