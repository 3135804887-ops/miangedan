package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
)

// IdempotencyExecutor serializes an operation/key pair and returns the first
// completed result for identical request content. Production persistence is
// defined by identity_idempotency in migration 0010.
type IdempotencyExecutor interface {
	Execute(
		ctx context.Context,
		operation, dataRegion, key, requestHash string,
		fn func() ([]byte, error),
	) ([]byte, error)
}

type idempotencyEntry struct {
	requestHash string
	done        chan struct{}
	result      []byte
	err         error
}

// MemoryIdempotency is a concurrent reference executor. Retryable failures are
// released so the same key can safely retry; completed and deterministic error
// results remain stable for the process lifetime.
type MemoryIdempotency struct {
	mu      sync.Mutex
	entries map[string]*idempotencyEntry
}

// NewMemoryIdempotency creates an empty idempotency executor.
func NewMemoryIdempotency() *MemoryIdempotency {
	return &MemoryIdempotency{entries: make(map[string]*idempotencyEntry)}
}

// Execute serializes one operation, region and key and replays its completed result.
func (m *MemoryIdempotency) Execute(
	ctx context.Context,
	operation, dataRegion, key, requestHash string,
	fn func() ([]byte, error),
) ([]byte, error) {
	if ctx == nil || fn == nil {
		return nil, errors.New("idempotency requires context and callback")
	}
	mapKey := operation + "\x00" + dataRegion + "\x00" + key
	m.mu.Lock()
	if existing, ok := m.entries[mapKey]; ok {
		if existing.requestHash != requestHash {
			m.mu.Unlock()
			return nil, domainError(
				CodeIdempotencyConflict,
				"同一幂等键对应了不同请求；本次未执行且已有数据保持不变。请使用新幂等键重试；不计费且不影响评分。",
				false,
				nil,
			)
		}
		done := existing.done
		m.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-done:
			return append([]byte(nil), existing.result...), existing.err
		}
	}
	entry := &idempotencyEntry{requestHash: requestHash, done: make(chan struct{})}
	m.entries[mapKey] = entry
	m.mu.Unlock()

	result, err := fn()
	entry.result = append([]byte(nil), result...)
	entry.err = err
	m.mu.Lock()
	if domain := AsDomainError(err); err != nil && domain.Retryable {
		delete(m.entries, mapKey)
	}
	close(entry.done)
	m.mu.Unlock()
	return append([]byte(nil), result...), err
}

func executeJSON[T any](
	ctx context.Context,
	executor IdempotencyExecutor,
	operation, dataRegion, key, requestHash string,
	fn func() (T, error),
) (T, error) {
	var zero T
	result, err := executor.Execute(ctx, operation, dataRegion, key, requestHash, func() ([]byte, error) {
		value, callErr := fn()
		if callErr != nil {
			return nil, callErr
		}
		encoded, encodeErr := json.Marshal(value)
		if encodeErr != nil {
			return nil, internalError(encodeErr)
		}
		return encoded, nil
	})
	if err != nil {
		return zero, err
	}
	var value T
	if err := json.Unmarshal(result, &value); err != nil {
		return zero, internalError(err)
	}
	return value, nil
}

func hashRequest(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
