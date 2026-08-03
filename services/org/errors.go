// Package org 提供机构租户错误集（TASK-070）。
package org

import "errors"

var (
	// ErrNotFound 记录不存在。
	ErrNotFound = errors.New("org record not found")
	// ErrInvalidInput 输入非法。
	ErrInvalidInput = errors.New("invalid org input")
	// ErrStateConflict 状态冲突。
	ErrStateConflict = errors.New("org state conflict")
	// ErrForbidden 权限不足。
	ErrForbidden = errors.New("org permission denied")
)
