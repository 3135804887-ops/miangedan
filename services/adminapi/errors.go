// Package adminapi 提供后台服务错误集（TASK-080）。
package adminapi

import "errors"

var (
	// ErrNotFound 记录不存在。
	ErrNotFound = errors.New("admin record not found")
	// ErrInvalidInput 输入非法。
	ErrInvalidInput = errors.New("invalid admin input")
	// ErrStateConflict 状态冲突。
	ErrStateConflict = errors.New("admin state conflict")
	// ErrForbidden 权限不足或红线操作。
	ErrForbidden = errors.New("admin permission denied")
)
