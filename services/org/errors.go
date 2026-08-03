// Package org 提供机构租户错误集（TASK-070）。
package org

import "errors"

var (
	ErrNotFound      = errors.New("org record not found")
	ErrInvalidInput  = errors.New("invalid org input")
	ErrStateConflict = errors.New("org state conflict")
	ErrForbidden     = errors.New("org permission denied")
)
