package export

import "errors"

// 错误集。
var (
	ErrInvalidInput  = errors.New("invalid export/deletion request")
	ErrNotFound      = errors.New("task not found")
	ErrStateConflict = errors.New("task state conflict")
)
