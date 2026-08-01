package ingestion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrUploadNotFound indicates that no upload exists for the requested identifier.
var ErrUploadNotFound = errors.New("upload not found")

// MemoryRepository is a deterministic adapter for tests and local development. Production uses the
// PostgreSQL contract in migrations/0012_resume_uploads.sql with identical uniqueness semantics.
type MemoryRepository struct {
	mu          sync.Mutex
	byID        map[string]Upload
	byIdem      map[string]string
	retryByIdem map[string]string
}

// NewMemoryRepository creates an empty concurrency-safe in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{byID: map[string]Upload{}, byIdem: map[string]string{}, retryByIdem: map[string]string{}}
}

// CreateOrGet persists a new upload or returns the matching idempotent result.
func (r *MemoryRepository) CreateOrGet(_ context.Context, candidate Upload) (Upload, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := candidate.DataRegion + ":" + candidate.UserID + ":" + candidate.IdempotencyKey
	if uploadID, ok := r.byIdem[key]; ok {
		existing := r.byID[uploadID]
		if existing.ContentFingerprint != candidate.ContentFingerprint || existing.Filename != candidate.Filename {
			return Upload{}, false, ErrIdempotencyConflict
		}
		return existing, false, nil
	}
	r.byID[candidate.UploadID] = candidate
	r.byIdem[key] = candidate.UploadID
	return candidate, true, nil
}

// Get returns an upload by identifier.
func (r *MemoryRepository) Get(_ context.Context, uploadID string) (Upload, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	upload, ok := r.byID[uploadID]
	if !ok {
		return Upload{}, ErrUploadNotFound
	}
	return upload, nil
}

// Update replaces the stored state for an existing upload.
func (r *MemoryRepository) Update(_ context.Context, upload Upload) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[upload.UploadID]; !ok {
		return ErrUploadNotFound
	}
	r.byID[upload.UploadID] = upload
	return nil
}

// BeginRetry atomically claims an idempotent retry for a retryable upload.
func (r *MemoryRepository) BeginRetry(_ context.Context, uploadID, idempotencyKey string) (Upload, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	upload, ok := r.byID[uploadID]
	if !ok {
		return Upload{}, false, ErrUploadNotFound
	}
	key := uploadID + ":" + idempotencyKey
	if existingID, exists := r.retryByIdem[key]; exists {
		return r.byID[existingID], false, nil
	}
	if upload.Status != StatusRetryableFailed {
		return Upload{}, false, ErrNotRetryable
	}
	r.retryByIdem[key] = uploadID
	return upload, true, nil
}

// MemoryUploadStore models the uploads bucket for tests and local development.
type MemoryUploadStore struct {
	mu           sync.Mutex
	objects      map[string][]byte
	PutCount     int
	PromoteCount int
	DeleteCount  int
}

// NewMemoryUploadStore creates an empty concurrency-safe uploads store.
func NewMemoryUploadStore() *MemoryUploadStore {
	return &MemoryUploadStore{objects: map[string][]byte{}}
}

// PutQuarantine writes content only under a regional uploads quarantine prefix.
func (s *MemoryUploadStore) PutQuarantine(_ context.Context, dataRegion, key string, content []byte) (ObjectRef, error) {
	if !regionBucket(dataRegion) || !strings.HasPrefix(key, "quarantine/") {
		return ObjectRef{}, errors.New("uploads bucket or quarantine prefix violation")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ref := ObjectRef{Bucket: dataRegion + "-uploads", Key: key}
	s.objects[ref.Bucket+"/"+ref.Key] = append([]byte(nil), content...)
	s.PutCount++
	return ref, nil
}

// ReadQuarantine returns a defensive copy only for a regional uploads quarantine object.
func (s *MemoryUploadStore) ReadQuarantine(_ context.Context, ref ObjectRef) ([]byte, error) {
	if !strings.HasSuffix(ref.Bucket, "-uploads") || !strings.HasPrefix(ref.Key, "quarantine/") {
		return nil, errors.New("only quarantined uploads objects can be read")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	content, ok := s.objects[ref.Bucket+"/"+ref.Key]
	if !ok {
		return nil, errors.New("quarantine object missing")
	}
	return append([]byte(nil), content...), nil
}

// PromoteAccepted moves a quarantined object into the accepted uploads prefix.
func (s *MemoryUploadStore) PromoteAccepted(_ context.Context, ref ObjectRef) (ObjectRef, error) {
	if !strings.HasSuffix(ref.Bucket, "-uploads") || !strings.HasPrefix(ref.Key, "quarantine/") {
		return ObjectRef{}, errors.New("only quarantined uploads objects can be promoted")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	oldKey := ref.Bucket + "/" + ref.Key
	content, ok := s.objects[oldKey]
	if !ok {
		return ObjectRef{}, errors.New("quarantine object missing")
	}
	accepted := ObjectRef{Bucket: ref.Bucket, Key: strings.Replace(ref.Key, "quarantine/", "accepted/", 1)}
	s.objects[accepted.Bucket+"/"+accepted.Key] = content
	delete(s.objects, oldKey)
	s.PromoteCount++
	return accepted, nil
}

// Delete removes an object only when its reference belongs to an uploads bucket.
func (s *MemoryUploadStore) Delete(_ context.Context, ref ObjectRef) error {
	if !strings.HasSuffix(ref.Bucket, "-uploads") {
		return fmt.Errorf("refusing delete outside uploads bucket: %s", ref.Bucket)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, ref.Bucket+"/"+ref.Key)
	s.DeleteCount++
	return nil
}

// Has reports whether the referenced uploads object exists.
func (s *MemoryUploadStore) Has(ref ObjectRef) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.objects[ref.Bucket+"/"+ref.Key]
	return ok
}

func regionBucket(value string) bool { return value == "cn" || value == "eu" || value == "intl" }
